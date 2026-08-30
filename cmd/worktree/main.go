package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/harrylawton/pr-review/internal/worktree"
)

func main() {
	stateDir := flag.String("state", "data", "state directory for repo clones and worktrees")
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: worktree [-state dir] <owner/repo> <pr-number> <head-sha>")
		os.Exit(1)
	}

	owner, repo, ok := strings.Cut(args[0], "/")
	if !ok {
		fmt.Fprintln(os.Stderr, "owner/repo must be in the form owner/repo")
		os.Exit(1)
	}

	prNumber, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "pr-number must be an integer")
		os.Exit(1)
	}

	m := worktree.New(*stateDir)
	path, err := m.EnsureWorktree(owner, repo, prNumber, args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println(path)
}
