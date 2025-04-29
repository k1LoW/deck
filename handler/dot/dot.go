package dot

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

var (
	yellow = color.New(color.FgYellow, color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
)

var _ slog.Handler = (*dotHandler)(nil)

func New(h slog.Handler) *dotHandler {
	return &dotHandler{
		handler: h,
		stdout:  colorable.NewColorableStdout(),
	}
}

type dotHandler struct {
	handler slog.Handler
	stdout  io.Writer
}

func (h *dotHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *dotHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Message == "applied page" {
		_, _ = h.stdout.Write([]byte(yellow(".")))
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
	return &dotHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *dotHandler) WithGroup(name string) slog.Handler {
	return &dotHandler{handler: h.handler.WithGroup(name)}
}
