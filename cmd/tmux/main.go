package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harrylawton/pr-review/internal/tmux"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	name := flag.String("name", "", "tmux session name")
	dir := flag.String("dir", ".", "working directory for every window")
	force := flag.Bool("force", false, "replace an existing session with the same name")
	flag.Usage = usage
	flag.Parse()

	if *name == "" {
		return fmt.Errorf("-name is required")
	}
	if flag.NArg() == 0 {
		return fmt.Errorf("no files given")
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		return err
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return fmt.Errorf("not a directory: %s", root)
	}

	windows := make([]tmux.Window, 0, flag.NArg())
	for _, arg := range flag.Args() {
		path, line := splitLine(arg)
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			return fmt.Errorf("no such file under %s: %s", root, path)
		}
		windows = append(windows, tmux.EditorWindow("nvim", path, line))
	}

	ctx := context.Background()
	tm := tmux.Init()
	session := tmux.SessionName(*name)

	if tm.HasSession(ctx, session) {
		if !*force {
			return fmt.Errorf("session %q already exists (pass -force to replace it)", session)
		}
		if err := tm.KillSession(ctx, session); err != nil {
			return err
		}
	}

	if err := tm.CreateSession(ctx, session, root, windows); err != nil {
		return err
	}

	fmt.Printf("session %s - %d window(s) in %s\n", session, len(windows), root)
	fmt.Printf("attach: tmux attach -t %s\n", session)
	return nil
}

func splitLine(arg string) (string, int) {
	path, num, ok := strings.Cut(arg, ":")
	if !ok {
		return arg, 0
	}
	line, err := strconv.Atoi(num)
	if err != nil || line < 1 {
		return arg, 0
	}
	return path, line
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tmux -name NAME [-dir DIR] [-force] file[:line] ...")
	flag.PrintDefaults()
}
