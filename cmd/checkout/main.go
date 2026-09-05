package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/harrylawton/pr-review/internal/checkout"
)

func main() {
	stateDir := flag.String("state", "data", "state directory for repo clones")
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: checkout [-state dir] <owner/repo> <pr-number> <head-sha>")
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

	worktree, err := checkout.New(*stateDir).Worktree(context.Background(), owner, repo, prNumber, args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println(worktree)
}
