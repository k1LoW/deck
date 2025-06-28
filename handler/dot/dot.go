package dot

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

var (
	yellow = color.New(color.FgYellow, color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	gray   = color.New(color.FgHiBlack).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
)

var _ slog.Handler = (*dotHandler)(nil)

type dotHandler struct {
	handler slog.Handler
	spinner *spinner.Spinner
	stdout  io.Writer
}

func New(h slog.Handler) (*dotHandler, error) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	if err := s.Color("yellow"); err != nil {
		return nil, err
	}
	return &dotHandler{
		handler: h,
		spinner: s,
		stdout:  colorable.NewColorableStdout(),
	}, nil
}

func (h *dotHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *dotHandler) Handle(ctx context.Context, r slog.Record) error {
	h.spinner.Stop()
	if r.Message == "applied page" {
		_, _ = h.stdout.Write([]byte(yellow(".")))
		return nil
	}
	if r.Message == "deleted page" {
		_, _ = h.stdout.Write([]byte(gray("x")))
		return nil
	}
	if r.Message == "appended page" {
		_, _ = h.stdout.Write([]byte(yellow("+")))
		return nil
	}
	if r.Message == "moved page" {
		_, _ = h.stdout.Write([]byte(green("-")))
		return nil
	}
	if r.Message == "inserted page" {
		_, _ = h.stdout.Write([]byte(yellow("+")))
		return nil
	}
	if strings.HasPrefix(r.Message, "retrying") {
		h.spinner.Start()
		return nil
	}
	if strings.Contains(r.Message, "because freeze:true") {
		_, _ = h.stdout.Write([]byte(cyan("*")))
		return nil
	}
	if r.Message == "apply completed" {
		_, _ = h.stdout.Write([]byte("\n"))
		return nil
	}
	return nil
}

func (h *dotHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dotHandler{handler: h.handler.WithAttrs(attrs), spinner: h.spinner, stdout: h.stdout}
}

func (h *dotHandler) WithGroup(name string) slog.Handler {
	return &dotHandler{handler: h.handler.WithGroup(name), spinner: h.spinner, stdout: h.stdout}
}
