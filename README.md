# pr-review

Review pull requests with agents, in a UI I own. Go + SolidJS, SQLite, one
binary. Reviews link into opencode to continue the session - opencode owns the
transcript, there is no in-app follow-up.

## Requirements

| Dependency | Required | Why                                                               |
| ---------- | -------- | ----------------------------------------------------------------- |
| git        | yes      | clones and per-PR worktrees, done in Go, never delegated to an agent |
| opencode   | yes      | runs the review agents; spawned automatically if not already up    |
| tmux       | no       | opens picked diff lines in an editor, one window per file          |
| nvim       | no       | the default `tmux.editor`; any `+<line>`-capable editor works      |
| gh         | no       | only used as a token fallback if you skip the device-code login    |

`pr-review doctor` tells you which of these you are missing and what to do
about it.

## Quickstart

```bash
make build          # web bundle + bin/pr-review
bin/pr-review setup # config, github login, database
bin/pr-review serve
```

`setup` refuses to run without opencode, then asks three things: tmux on or
off, which editor, and how to authenticate. Everything else - paths, port,
opencode URL - has a default and a commented line in the config file. It is
idempotent; re-running preserves anything you edited by hand.

`~/.config/pr-review/` holds config and token. `~/.local/share/pr-review/`
holds the database and the `data/` directory of clones and worktrees; only the
former is worth backing up, and `$XDG_DATA_HOME` overrides the latter.

## GitHub auth

Tokens are resolved in this order, first hit wins:

1. `GITHUB_TOKEN` - nothing is written to disk
2. `github.token_file` - written by `setup`, mode 0600
3. `gh auth token` - only if `github.use_gh_cli = true`

`setup` offers a device-code login, which needs an OAuth app client ID. Create
one at <https://github.com/settings/developers> with device flow enabled and
paste the client ID when asked; it is public, not a secret. If you would rather
not, paste a personal access token with the `repo` scope, or lean on `gh`.

## Configuration

`~/.config/pr-review/config.toml`, or wherever `PR_REVIEW_CONFIG` points. The
file `setup` writes is commented; that file is the reference, not this README.

Runtime preferences that belong to the UI - theme, per-agent model and prompt
overrides - live in the database, not here. The config file is for things the
process needs before it can serve a request.

`REMOTE_HOST` and `LOG_LEVEL` override their config equivalents.

## Commands

```
pr-review setup             first run, and every change to it afterwards
pr-review doctor            dependency and config check, non-zero on failure
pr-review serve             run the server
pr-review migrate <cmd>     up, down, down-to, redo, status, reset
```

`serve` runs the same checks as `doctor` and refuses to start if a required one
fails. Migrations are never applied implicitly by `serve`.

The `Makefile` wraps all of these (`make setup`, `make doctor`, `make server`,
`make up`), plus `make vet` - build, vet and `tsc -b`, the command to run after
any change.

## Docs

- `docs/architecture.md` - how it fits together and why
- `todo.txt` - live task list
