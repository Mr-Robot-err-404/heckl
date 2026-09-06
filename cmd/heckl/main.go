package main

import (
	"embed"
	"fmt"
	"os"
)

//go:embed all:dist
var static embed.FS

const usage = `heckl - review pull requests with agents

usage:
  heckl setup           interactive first-run: config, github login, database
  heckl doctor          check dependencies and configuration
  heckl serve           run the server
  heckl migrate <cmd>   goose migrations (up, down, status, redo, reset)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	args := os.Args[2:]
	switch os.Args[1] {
	case "setup":
		runSetup()
	case "doctor":
		runDoctor()
	case "serve":
		runServe()
	case "migrate":
		runMigrate(args)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Printf("unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}
