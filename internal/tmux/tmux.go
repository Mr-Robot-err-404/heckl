package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Tmux struct{}

func Init() *Tmux { return &Tmux{} }

func Installed() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

type Window struct {
	Name    string
	Command []string
}

func run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func target(session string) string { return "=" + session }

func (tm *Tmux) HasSession(ctx context.Context, name string) bool {
	_, err := run(ctx, "has-session", "-t", target(name))
	return err == nil
}

func (tm *Tmux) KillSession(ctx context.Context, name string) error {
	_, err := run(ctx, "kill-session", "-t", target(name))
	return err
}

func (tm *Tmux) CreateSession(ctx context.Context, name, directory string, windows []Window) error {
	if len(windows) == 0 {
		return fmt.Errorf("tmux: create session %q: no windows", name)
	}

	first := append([]string{"new-session", "-d", "-s", name, "-c", directory, "-n", windows[0].Name}, windows[0].Command...)
	if _, err := run(ctx, first...); err != nil {
		return err
	}

	for _, w := range windows[1:] {
		args := append([]string{"new-window", "-t", target(name), "-c", directory, "-n", w.Name}, w.Command...)
		if _, err := run(ctx, args...); err != nil {
			tm.KillSession(ctx, name)
			return err
		}
	}

	_, err := run(ctx, "select-window", "-t", target(name)+":^")
	return err
}

func SessionName(parts ...string) string {
	joined := strings.Join(parts, "-")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '.' || r == ':':
			return '-'
		case r < ' ':
			return -1
		}
		return r
	}, joined)
}

func NvimWindow(path string, line int) Window {
	cmd := []string{"nvim"}
	if line > 0 {
		cmd = append(cmd, "+"+strconv.Itoa(line))
	}
	return Window{Name: filepath.Base(path), Command: append(cmd, path)}
}
