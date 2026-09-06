package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Mr-Robot-err-404/heckl/internal/config"
	"github.com/Mr-Robot-err-404/heckl/internal/preflight"
	"github.com/Mr-Robot-err-404/heckl/internal/term"
)

var out = term.NewPrinter(os.Stdout)

func runDoctor() {
	cfg, err := config.Load()
	if err != nil {
		fatal(err.Error())
	}

	fmt.Printf("%s %s\n\n", out.Paint(term.Bold, "config"), out.Paint(term.Grey, cfg.Path()))
	checks := preflight.Run(context.Background(), cfg)
	printChecks(checks)

	if preflight.Failed(checks) {
		os.Exit(1)
	}
}

func printChecks(checks []preflight.Check) {
	for _, c := range checks {
		mark, colour := "ok", term.Green
		switch c.Status {
		case preflight.Warn:
			mark, colour = "warn", term.Yellow
		case preflight.Fail:
			mark, colour = "fail", term.Red
		}

		fmt.Printf("  %-6s %-16s %s\n",
			out.Paint(colour, mark),
			c.Name,
			out.Paint(term.Grey, c.Detail),
		)
		if c.Status != preflight.OK && c.Hint != "" {
			fmt.Printf("         %s\n", out.Paint(term.Grey, "→ "+c.Hint))
		}
	}
	fmt.Println()
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", out.Paint(term.Red, "error"), msg)
	os.Exit(1)
}
