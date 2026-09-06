package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harrylawton/pr-review/internal/config"
	"github.com/harrylawton/pr-review/internal/ghauth"
	"github.com/harrylawton/pr-review/internal/preflight"
	"github.com/harrylawton/pr-review/internal/store"
	"github.com/harrylawton/pr-review/internal/term"
	_ "modernc.org/sqlite"
)

var stdin = bufio.NewReader(os.Stdin)

func runSetup() {
	ctx := context.Background()
	path := config.Path()

	fmt.Printf("\n%s\n%s\n\n",
		out.Paint(term.Bold, "pr-review setup"),
		out.Paint(term.Grey, "config → "+path),
	)

	requireOpencode()

	cfg := loadOrDefault(path)
	promptConfig(cfg)

	if err := cfg.Save(path); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("%s wrote %s\n\n", out.Paint(term.Green, "ok"), path)

	setupAuth(ctx, cfg)
	setupDatabase(ctx, cfg)

	fmt.Printf("%s\n", out.Paint(term.Bold, "checks"))
	checks := preflight.Run(ctx, cfg)
	printChecks(checks)

	if preflight.Failed(checks) {
		fmt.Printf("%s setup finished with failures — fix them, then run %s\n\n",
			out.Paint(term.Yellow, "note"),
			out.Paint(term.Cyan, "pr-review doctor"),
		)
		os.Exit(1)
	}

	fmt.Printf("%s start it with %s, then open %s\n\n",
		out.Paint(term.Green, "done"),
		out.Paint(term.Cyan, "pr-review serve"),
		out.Paint(term.Cyan, "http://localhost"+cfg.Server.Addr),
	)
}

func requireOpencode() {
	if _, err := exec.LookPath("opencode"); err == nil {
		return
	}
	fmt.Printf("%s opencode is not on PATH, and nothing here works without it.\n", out.Paint(term.Red, "stop"))
	fmt.Printf("  it runs every review. install it from %s, then run setup again.\n\n",
		out.Paint(term.Cyan, "https://opencode.ai"))
	os.Exit(1)
}

func loadOrDefault(path string) *config.Config {
	if existing, err := config.Load(); err == nil {
		fmt.Printf("%s existing config found — enter keeps the current value\n\n", out.Paint(term.Grey, "note"))
		return existing
	}
	cfg := config.Defaults()
	if cwd, err := os.Getwd(); err == nil {
		cfg.OpenCode.ProjectDir = cwd
	}
	return &cfg
}

func promptConfig(cfg *config.Config) {
	cfg.Tmux.Enabled = askBool("enable tmux review sessions", cfg.Tmux.Enabled)
	if cfg.Tmux.Enabled {
		cfg.Tmux.Editor = ask("editor for tmux windows", cfg.Tmux.Editor)
	}
	fmt.Println()
}

func setupAuth(ctx context.Context, cfg *config.Config) {
	fmt.Printf("%s\n", out.Paint(term.Bold, "github"))

	if tok, err := ghauth.Resolve(authOptions(cfg)); err == nil {
		if login, err := ghauth.Login(ctx, tok.Value); err == nil {
			fmt.Printf("%s signed in as %s %s\n\n",
				out.Paint(term.Green, "ok"),
				out.Paint(term.Bold, login),
				out.Paint(term.Grey, "("+string(tok.Source)+")"),
			)
			if !askBool("sign in again", false) {
				fmt.Println()
				return
			}
			fmt.Println()
		} else {
			fmt.Printf("%s existing token rejected: %s\n", out.Paint(term.Yellow, "warn"), err)
		}
	}

	token := acquireToken(ctx, cfg)
	if token == "" {
		fmt.Printf("%s skipped — the server will not start without a token\n\n", out.Paint(term.Yellow, "warn"))
		return
	}

	login, err := ghauth.Login(ctx, token)
	if err != nil {
		fatal(err.Error())
	}
	if err := ghauth.SaveToken(cfg.GitHub.TokenFile, token); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("%s signed in as %s, token saved to %s\n\n",
		out.Paint(term.Green, "ok"),
		out.Paint(term.Bold, login),
		out.Paint(term.Grey, cfg.GitHub.TokenFile),
	)
}

func acquireToken(ctx context.Context, cfg *config.Config) string {
	fmt.Printf("  %s browser login via a github oauth app (device flow)\n", out.Paint(term.Cyan, "1."))
	fmt.Printf("  %s paste a personal access token\n", out.Paint(term.Cyan, "2."))
	fmt.Printf("  %s skip\n\n", out.Paint(term.Cyan, "3."))

	switch ask("choice", "1") {
	case "1":
		return deviceLogin(ctx, cfg)
	case "2":
		return ask("token (needs the repo scope)", "")
	default:
		return ""
	}
}

func deviceLogin(ctx context.Context, cfg *config.Config) string {
	if cfg.GitHub.OAuthClientID == "" {
		fmt.Printf("\n%s no oauth client id configured.\n", out.Paint(term.Yellow, "note"))
		fmt.Printf("  create one at %s — any name, any callback url, tick\n",
			out.Paint(term.Cyan, "https://github.com/settings/developers"))
		fmt.Printf("  \"enable device flow\", then paste the client id here. It is not a secret.\n\n")
		cfg.GitHub.OAuthClientID = ask("oauth client id", "")
		if cfg.GitHub.OAuthClientID == "" {
			return ""
		}
		if err := cfg.Save(config.Path()); err != nil {
			fatal(err.Error())
		}
	}

	code, err := ghauth.RequestDeviceCode(ctx, cfg.GitHub.OAuthClientID)
	if err != nil {
		fatal(err.Error())
	}

	fmt.Printf("\n  open %s and enter %s\n",
		out.Paint(term.Cyan, code.VerificationURI),
		out.Paint(term.Bold, code.UserCode),
	)
	fmt.Printf("  %s\n", out.Paint(term.Grey, "waiting for approval..."))

	token, err := ghauth.PollForToken(ctx, cfg.GitHub.OAuthClientID, code)
	if err != nil {
		fatal(err.Error())
	}
	return token
}

func setupDatabase(ctx context.Context, cfg *config.Config) {
	fmt.Printf("%s\n", out.Paint(term.Bold, "database"))

	for _, dir := range []string{filepath.Dir(cfg.Server.Database), cfg.Server.DataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal(err.Error())
		}
	}

	db, err := sql.Open("sqlite", cfg.Server.Database)
	if err != nil {
		fatal(err.Error())
	}
	defer db.Close()

	before, err := store.PendingMigrations(db)
	if err != nil {
		fatal(err.Error())
	}
	if before == 0 {
		fmt.Printf("%s schema up to date %s\n\n", out.Paint(term.Green, "ok"), out.Paint(term.Grey, cfg.Server.Database))
		return
	}
	if err := store.Migrate(ctx, db); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("%s applied %d migration(s) %s\n\n",
		out.Paint(term.Green, "ok"), before, out.Paint(term.Grey, cfg.Server.Database))
}

func authOptions(cfg *config.Config) ghauth.Options {
	return ghauth.Options{TokenFile: cfg.GitHub.TokenFile, UseGHCLI: cfg.GitHub.UseGHCLI}
}

func ask(label, fallback string) string {
	if fallback == "" {
		fmt.Printf("%s: ", label)
	} else {
		fmt.Printf("%s %s: ", label, out.Paint(term.Grey, "["+fallback+"]"))
	}
	line, err := stdin.ReadString('\n')
	if err != nil {
		return fallback
	}
	if answer := strings.TrimSpace(line); answer != "" {
		return answer
	}
	return fallback
}

func askBool(label string, fallback bool) bool {
	def := "n"
	if fallback {
		def = "y"
	}
	switch strings.ToLower(ask(label+" (y/n)", def)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
