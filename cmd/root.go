/*
Copyright © 2025 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/deck/version"
	"github.com/k1LoW/errors"
	"github.com/spf13/cobra"
)

var profile string

var rootCmd = &cobra.Command{
	Use:          "deck",
	Short:        "deck is a tool for creating deck using Markdown and Google Slides",
	Long:         `deck is a tool for creating deck using Markdown and Google Slides.`,
	SilenceUsage: true,
	Version:      fmt.Sprintf("%s (rev:%s)", version.Version, version.Revision),
}

type errorData struct {
	LatestLogs  []any     `json:"latest_logs"`
	StackTraces any       `json:"stack_traces"`
	CreatedAt   time.Time `json:"created_at"`
	Version     string    `json:"version"`
	Revision    string    `json:"revision"`
}

// https://slides.googleapis.com/v1/presentations/xxxxxx
// https://www.googleapis.com/drive/v3/files/xxxxxx
var googleAPIURLRe = regexp.MustCompile(`(https://(?:slides.googleapis.com/v1/presentations|www.googleapis.com/drive/v3/files)/)([^\?"]+)`)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Write stack trace log to state directory
		var latestLogs []any
		for _, line := range tb.Lines() {
			// replace Google API URL last key with a placeholder
			line = googleAPIURLRe.ReplaceAllString(line, "${1}**********************")
			if strings.Contains(line, `"level":"DEBUG"`) && strings.Contains(line, `"request":`) {
				// Skip debug logs that contain request details
				continue
			}
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				latestLogs = append(latestLogs, line)
			} else {
				latestLogs = append(latestLogs, m)
			}
		}
		d := &errorData{
			LatestLogs:  latestLogs,
			StackTraces: errors.StackTraces(err),
			CreatedAt:   time.Now(),
			Version:     version.Version,
			Revision:    version.Revision,
		}
		b, err := json.Marshal(d)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		} else {
			dumpPath := filepath.Join(config.StateHomePath(), "error.json")
			if err := os.WriteFile(dumpPath, b, 0o600); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed to write error.json to %s: %v\n", dumpPath, err)
			}
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "", "", "profile name")
}
