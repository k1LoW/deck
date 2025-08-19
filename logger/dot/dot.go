package dot

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/k1LoW/errors"
	"github.com/mattn/go-colorable"
)

var (
	yellow = color.New(color.FgYellow, color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	gray   = color.New(color.FgHiBlack).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
)

var _ slog.Handler = (*dotHandler)(nil)

type dotHandler struct {
	handler slog.Handler
	spinner *spinner.Spinner
	stdout  io.Writer
	prefix  []byte
}

func New(h slog.Handler) (_ *dotHandler, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	stdout := colorable.NewColorableStdout()
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(stdout))
	if err := s.Color("yellow"); err != nil {
		return nil, err
	}
	s.Start()
	s.Disable()
	return &dotHandler{
		handler: h,
		spinner: s,
		stdout:  stdout,
	}, nil
}

func (h *dotHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func buildRepeatedSymbols(r slog.Record, c rune) string {
	var count int
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "count" {
			count = int(attr.Value.Int64())
			return false
		}
		return true
	})
	return strings.Repeat(string(c), count)
}

func (h *dotHandler) Handle(ctx context.Context, r slog.Record) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	if strings.HasPrefix(r.Message, "retrying") {
		if !h.spinner.Enabled() {
			h.spinner.Enable()
		}
		return nil
	}
	if h.spinner.Enabled() {
		h.spinner.Disable()
		_, _ = h.stdout.Write(h.prefix)
	}

	switch r.Message {
	case "applied pages":
		msg := buildRepeatedSymbols(r, '.')
		return h.write([]byte(yellow(msg)))
	case "deleted pages":
		msg := buildRepeatedSymbols(r, '-')
		return h.write([]byte(gray(msg)))
	case "appended pages":
		msg := buildRepeatedSymbols(r, '+')
		return h.write([]byte(yellow(msg)))
	case "moved page":
		var from, to int64
		r.Attrs(func(attr slog.Attr) bool {
			if attr.Key == "from_index" {
				from = attr.Value.Int64()
			}
			if attr.Key == "to_index" {
				to = attr.Value.Int64()
			}
			return true
		})
		switch {
		case from < to:
			if err := h.write([]byte(green("↓"))); err != nil {
				return err
			}
		case from > to:
			if err := h.write([]byte(green("↑"))); err != nil {
				return err
			}
		}
		return nil
	}

	if strings.Contains(r.Message, "because freeze:true") {
		if err := h.write([]byte(cyan("*"))); err != nil {
			return err
		}
		return nil
	}
	if strings.Contains(r.Message, "failed to") {
		if err := h.write([]byte(red("!"))); err != nil {
			return err
		}
		_, err = h.stdout.Write([]byte("\n"))
		return err
	}
	if strings.Contains(r.Message, "apply completed") || r.Message == "applied changes" {
		_, err = h.stdout.Write([]byte("\n"))
		return err
	}
	return nil
}

func (h *dotHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dotHandler{handler: h.handler.WithAttrs(attrs), spinner: h.spinner, stdout: h.stdout}
}

func (h *dotHandler) WithGroup(name string) slog.Handler {
	return &dotHandler{handler: h.handler.WithGroup(name), spinner: h.spinner, stdout: h.stdout}
}

func (h *dotHandler) write(s []byte) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	_, err = h.stdout.Write(s)
	if err != nil {
		return err
	}
	h.prefix = append(h.prefix, s...)
	return nil
}
