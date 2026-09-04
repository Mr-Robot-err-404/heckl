package tmux

import (
	"context"
	"fmt"
	"os/exec"
)

type Tmux struct {
}

const (
	ListSessions string = "list-sessions"
)

func run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tmux %v: %w: %s", args, err, out)
	}
	return string(out), nil
}

func (tm *Tmux) HasServer(ctx context.Context) bool {
	_, err := run(ctx, ListSessions)
	return err == nil
}
