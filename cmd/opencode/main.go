package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/harrylawton/pr-review/internal/opencode"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:4420", "opencode server base URL")
	agent := flag.String("agent", "build", "agent to use")
	worktree := flag.String("worktree", "", "restrict external_directory access to this path only")
	setup := flag.Bool("setup", false, "bootstrap the server via opencode.Setup instead of assuming it's already running")
	projectDir := flag.String("project-dir", ".", "project dir to spawn `opencode serve` from when -setup is used")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: opencode [-url url] [-agent name] [-worktree path] [-setup] [-project-dir path] <prompt text...>")
		os.Exit(1)
	}
	prompt := strings.Join(args, " ")

	var c *opencode.Client
	if *setup {
		var err error
		c, err = opencode.Setup(*url, *projectDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "setup error:", err)
			os.Exit(1)
		}
	} else {
		c = opencode.New(*url)
	}

	req := opencode.CreateSessionRequest{
		Title: "pr-review opencode cli",
		Agent: *agent,
	}
	if *worktree != "" {
		req.Permission = opencode.WorktreePermission(*worktree)
	}

	sess, err := c.CreateSession(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create session error:", err)
		os.Exit(1)
	}
	fmt.Println("session:", sess.ID)

	if _, err := c.Prompt(sess.ID, opencode.PromptRequest{
		Agent: *agent,
		Parts: []opencode.Part{{Type: "text", Text: prompt}},
	}); err != nil {
		fmt.Fprintln(os.Stderr, "prompt error:", err)
		os.Exit(1)
	}

	msgs, err := c.Messages(sess.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "messages error:", err)
		os.Exit(1)
	}

	for _, m := range msgs {
		if m.Info.Role != "assistant" {
			continue
		}
		for _, p := range m.Parts {
			switch p.Type {
			case "text":
				fmt.Println("---")
				fmt.Println(p.Text)
			}
		}
		if m.Info.HasError() {
			fmt.Println("ERROR:", string(m.Info.Error))
		}
	}
}
