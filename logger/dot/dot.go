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
	if r.Message == "applied page" {
		if err := h.write([]byte(yellow("."))); err != nil {
			return err
		}
		return nil
	}
	if r.Message == "deleted page" {
		if err := h.write([]byte(gray("-"))); err != nil {
			return err
		}
		return nil
	}
	if r.Message == "appended page" {
		if err := h.write([]byte(yellow("+"))); err != nil {
			return err
		}
		return nil
	}
	if r.Message == "moved page" {
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
	if r.Message == "inserted page" {
		if err := h.write([]byte(yellow("+"))); err != nil {
			return err
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
