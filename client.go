package deck

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/deck/version"
	"github.com/k1LoW/errors"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/slides/v1"
)

// Authentication environment variables.
// These are primarily for service account usage and override the default OAuth2 flow.
const (
	// EnvServiceAccountKey - JSON content of a service account key (not a file path).
	// Example: DECK_SERVICE_ACCOUNT_KEY='{"type":"service_account","project_id":"..."}'.
	EnvServiceAccountKey = "DECK_SERVICE_ACCOUNT_KEY"

	// EnvEnableADC - Enables Application Default Credentials (ADC) via google.DefaultClient.
	// Primarily for GitHub Actions with Workload Identity Federation.
	// ref. https://github.com/google-github-actions/auth
	EnvEnableADC = "DECK_ENABLE_ADC"

	// EnvAccessToken - Pre-existing OAuth2 access token for authentication.
	// Useful for tasks >5 minutes since GitHub OIDC tokens expire in 5 minutes.
	EnvAccessToken = "DECK_ACCESS_TOKEN"
)

var userAgent = "k1LoW-deck/" + version.Version + " (+https://github.com/k1LoW/deck)"

// getHTTPClient returns the appropriate client option based on available credentials.
func (d *Deck) getHTTPClient(ctx context.Context) (*http.Client, error) {
	client, err := func(ctx context.Context) (*http.Client, error) {
		if credsJSON := os.Getenv(EnvServiceAccountKey); credsJSON != "" {
			d.logger.Debug("using service account key authentication")
			return d.getServiceAccountHTTPClient(ctx, credsJSON)
		}
		if os.Getenv(EnvEnableADC) != "" {
			d.logger.Debug("using Application Default Credentials")
			return google.DefaultClient(ctx,
				"https://www.googleapis.com/auth/presentations", "https://www.googleapis.com/auth/drive")
		}
		if token := os.Getenv(EnvAccessToken); token != "" {
			d.logger.Debug("using access token authentication")
			return oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: token,
			})), nil
		}
		// default OAuth2 user authentication
		d.logger.Debug("using OAuth2 user authentication")
		return d.getDefaultHTTPClient(ctx)
	}(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = client
	retryClient.RetryMax = 10
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.Logger = newAPILogger(d.logger)

	return retryClient.StandardClient(), nil
}

func (d *Deck) getOAuthConfig() (*oauth2.Config, error) {
	creds := filepath.Join(config.DataHomePath(), "credentials.json")
	if d.profile != "" {
		p := filepath.Join(config.DataHomePath(), fmt.Sprintf("credentials-%s.json", d.profile))
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			creds = p
		}
	}
	b, err := os.ReadFile(creds)
	if err != nil {
		return nil, err
	}

	c, err := google.ConfigFromJSON(b, slides.PresentationsScope, slides.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("failed to get oauth config from json: %w", err)
	}
	return c, nil
}

func (d *Deck) getDefaultHTTPClient(ctx context.Context) (_ *http.Client, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	cfg, err := d.getOAuthConfig()
	if err != nil {
		return nil, err
	}

	tokenPath := filepath.Join(config.StateHomePath(), "token.json")
	if d.profile != "" {
		tokenPath = filepath.Join(config.StateHomePath(), fmt.Sprintf("token-%s.json", d.profile))
	}
	token, err := d.tokenFromFile(tokenPath)
	if err != nil {
		token, err = d.getTokenFromWeb(ctx, cfg)
		if err != nil {
			return nil, err
		}
		if err := d.saveToken(tokenPath, token); err != nil {
			return nil, err
		}
	} else if token.Expiry.Before(time.Now()) {
		// Token has expired, refresh it using the refresh token
		d.logger.Info("token has expired, refreshing")
		if token.RefreshToken != "" {
			tokenSource := cfg.TokenSource(ctx, token)
			newToken, err := tokenSource.Token()
			if err != nil {
				d.logger.Info("failed to refresh token, getting new token from web", slog.String("error", err.Error()))
				// If refresh fails, get a new token from the web
				newToken, err = d.getTokenFromWeb(ctx, cfg)
				if err != nil {
					return nil, err
				}
			} else {
				d.logger.Info("token refreshed successfully")
			}

			// Save the new token
			if err := d.saveToken(tokenPath, newToken); err != nil {
				return nil, err
			}
			token = newToken
		} else {
			// No refresh token available, get a new token from the web
			d.logger.Info("no refresh token available, getting new token from web")
			token, err = d.getTokenFromWeb(ctx, cfg)
			if err != nil {
				return nil, err
			}
			if err := d.saveToken(tokenPath, token); err != nil {
				return nil, err
			}
		}
	}
	return cfg.Client(ctx, token), nil
}

func (d *Deck) getTokenFromWeb(ctx context.Context, config *oauth2.Config) (_ *oauth2.Token, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	// Generate code verifier and challenge for PKCE
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	var authCode string

	// Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	listenCtx, listening := context.WithCancel(ctx)
	doneCtx, done := context.WithCancel(ctx)
	// run and stop local server
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("code") == "" {
			return
		}
		authCode = r.URL.Query().Get("code")
		_, _ = w.Write([]byte("Received code. You may now close this tab."))
		done()
	})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	var listenErr error
	go func() {
		ln, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			listenErr = fmt.Errorf("listen: %w", err)
			listening()
			done()
			return
		}
		srv.Addr = ln.Addr().String()
		listening()
		if err := srv.Serve(ln); err != nil {
			if err != http.ErrServerClosed {
				listenErr = fmt.Errorf("serve: %w", err)
				done()
				return
			}
		}
	}()
	<-listenCtx.Done()
	if listenErr != nil {
		return nil, listenErr
	}
	config.RedirectURL = "http://" + srv.Addr + "/"

	// Add PKCE parameters to the authorization URL
	authURL := config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	if err := browser.OpenURL(authURL); err != nil {
		return nil, err
	}

	<-doneCtx.Done()
	if err := srv.Shutdown(ctx); err != nil {
		return nil, err
	}

	// Send code verifier during token exchange
	token, err := config.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (d *Deck) tokenFromFile(file string) (_ *oauth2.Token, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}
	return token, err
}

func (d *Deck) saveToken(path string, token *oauth2.Token) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	return nil
}

// generateCodeVerifier generates a code verifier for PKCE.
// Generates a random string of 43-128 characters in compliance with RFC7636.
func generateCodeVerifier() (_ string, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	// Generate 64 bytes (512 bits) of random data
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge generates a code challenge from the code verifier.
// Calculates SHA-256 hash and applies Base64 URL encoding.
func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// getServiceAccountHTTPClient creates an HTTP client using service account credentials.
func (d *Deck) getServiceAccountHTTPClient(ctx context.Context, credsJSON string) (*http.Client, error) {
	config, err := google.JWTConfigFromJSON([]byte(credsJSON),
		"https://www.googleapis.com/auth/presentations",
		"https://www.googleapis.com/auth/drive",
	)
	if err != nil {
		return nil, err
	}
	return config.Client(ctx), nil
}

var _ retryablehttp.LeveledLogger = (*apiLogger)(nil)

type apiLogger struct {
	l *slog.Logger
}

func (l *apiLogger) Error(msg string, keysAndValues ...any) {
	l.l.Error(msg, append([]any{slog.String("original_log_level", "error")}, keysAndValues...)...)
}
func (l *apiLogger) Info(msg string, keysAndValues ...any) {
	l.l.Info(msg, append([]any{slog.String("original_log_level", "info")}, keysAndValues...)...)
}
func (l *apiLogger) Debug(msg string, keysAndValues ...any) {
	if strings.HasPrefix(msg, "retrying") {
		// If the message starts with "retrying", log it as info instead of debug
		// For displaying spinner messages in the console
		l.l.Info(msg, append([]any{slog.String("original_log_level", "debug")}, keysAndValues...)...)
		return
	}
	l.l.Debug(msg, append([]any{slog.String("original_log_level", "debug")}, keysAndValues...)...)
}
func (l *apiLogger) Warn(msg string, keysAndValues ...any) {
	l.l.Warn(msg, append([]any{slog.String("original_log_level", "warn")}, keysAndValues...)...)
}

func newAPILogger(l *slog.Logger) retryablehttp.LeveledLogger {
	return &apiLogger{
		l: l.WithGroup("api"),
	}
}
