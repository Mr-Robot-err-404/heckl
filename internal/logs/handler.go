package logs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/harrylawton/pr-review/internal/term"
)

const (
	reset  = term.Reset
	dim    = term.Dim
	bold   = term.Bold
	red    = term.Red
	green  = term.Green
	yellow = term.Yellow
	blue   = term.Blue
	cyan   = term.Cyan
	grey   = term.Grey
)

const (
	methodWidth  = 6
	pathWidth    = 44
	labelWidth   = 5
	messageWidth = 34
)

type Handler struct {
	mu    *sync.Mutex
	out   io.Writer
	level slog.Leveler
	color bool
	attrs []slog.Attr
}

func New(out *os.File, level slog.Leveler) *Handler {
	return &Handler{
		mu:    &sync.Mutex{},
		out:   out,
		level: level,
		color: term.Enabled(out),
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *Handler) WithGroup(string) slog.Handler { return h }

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	attrs := append([]slog.Attr{}, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})

	var b strings.Builder
	b.WriteString(h.paint(grey, r.Time.Format("15:04:05")))
	b.WriteByte(' ')

	if r.Message == "request" {
		h.writeRequest(&b, attrs)
	} else {
		h.writeEvent(&b, r.Level, r.Message, attrs)
	}
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func (h *Handler) writeRequest(b *strings.Builder, attrs []slog.Attr) {
	var method, path, duration string
	status := 0
	rest := attrs[:0:0]

	for _, a := range attrs {
		switch a.Key {
		case "method":
			method = a.Value.String()
		case "path":
			path = a.Value.String()
		case "status":
			status = int(a.Value.Int64())
		case "duration_ms":
			duration = formatMs(a.Value.Int64())
		default:
			rest = append(rest, a)
		}
	}

	b.WriteString(h.paint(methodColor(method), pad(method, methodWidth)))
	b.WriteByte(' ')
	b.WriteString(h.paint(statusColor(status), strconv.Itoa(status)))
	b.WriteByte(' ')
	b.WriteString(pad(path, pathWidth))
	b.WriteByte(' ')
	b.WriteString(h.paint(grey, lpad(duration, 7)))
	h.writeAttrs(b, rest)
}

func (h *Handler) writeEvent(b *strings.Builder, level slog.Level, msg string, attrs []slog.Attr) {
	b.WriteString(h.paint(levelColor(level), pad(levelLabel(level), labelWidth)))
	b.WriteByte(' ')
	b.WriteString(pad(msg, messageWidth))
	h.writeAttrs(b, attrs)
}

func (h *Handler) writeAttrs(b *strings.Builder, attrs []slog.Attr) {
	for _, a := range attrs {
		b.WriteByte(' ')
		if a.Key == "err" || a.Key == "error" {
			b.WriteString(h.paint(red, a.Key+"="+a.Value.String()))
			continue
		}
		b.WriteString(h.paint(grey, a.Key+"="))
		b.WriteString(h.paint(dim, a.Value.String()))
	}
}

func (h *Handler) paint(color, s string) string {
	if !h.color || color == "" {
		return s
	}
	return color + s + reset
}

func methodColor(method string) string {
	switch method {
	case "GET":
		return green
	case "POST":
		return yellow
	case "PUT", "PATCH":
		return blue
	case "DELETE":
		return red
	default:
		return cyan
	}
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return red
	case status >= 400:
		return yellow
	case status >= 300:
		return cyan
	default:
		return green
	}
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return bold + red
	case level >= slog.LevelWarn:
		return yellow
	case level >= slog.LevelInfo:
		return blue
	default:
		return grey
	}
}

func levelLabel(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DBUG"
	}
}

func formatMs(ms int64) string {
	if ms < 1000 {
		return strconv.FormatInt(ms, 10) + "ms"
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func lpad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
