# PR Review - Architecture

A personal PR review tool. Replaces the GitHub review UI with a fast, owned experience.

## Stack

- **Go** - HTTP server, GitHub API client, SQLite via sqlc + goose migrations
- **SolidJS + Vite** - frontend, not React
- **TanStack Query** (solid adapter) - data fetching
- **TanStack Router** (solid adapter) - URL state
- **@pierre/diffs** - diff rendering via CodeView vanilla JS API. See `docs/diffs-skill.md` and `docs/diffs-references/`
- **marked + DOMPurify** - markdown rendering in PR description tab
- **Port:** 7331 by default, `server.addr` in the config

## Project structure

```
pr-review/
├── cmd/
│   ├── pr-review/            - the binary: setup, doctor, serve, migrate
│   │   ├── main.go           - subcommand dispatch, embeds web/dist
│   │   ├── setup.go          - interactive first run
│   │   ├── doctor.go         - preflight report + shared CLI printing
│   │   ├── serve.go          - wiring, from config to listener
│   │   └── migrate.go        - goose runner over the configured db
│   ├── checkout/main.go      - checkout a PR head sha, print the path
│   ├── tmux/main.go          - open files in a tmux session by hand
│   └── opencode/main.go      - one-shot prompt against a running opencode
├── .opencode/
│   ├── agents/pr-reviewer.md - the review agent, git-tracked markdown
│   └── tools/report.ts       - custom tool the agent calls to submit a review
├── internal/
│   ├── config/               - TOML config, defaults, commented file template
│   ├── ghauth/               - device-code login, token resolution + storage
│   ├── preflight/            - dependency and configuration checks
│   ├── term/                 - ANSI constants + tty detection, shared by logs and CLI
│   ├── github/              - read-only GitHub API client
│   ├── checkout/             - per-repo clone + detached checkout at a sha
│   ├── opencode/             - HTTP client for `opencode serve` on :4420
│   ├── reviewer/             - checkout → session → prompt → parse → store
│   ├── orchestrator/         - review lifecycle, SSE fan-out keyed by PR
│   ├── store/                - sqlc-generated queries + Store wrapper
│   │   ├── schema/           - goose migrations (00001_init.sql, 00002_repos.sql)
│   │   └── queries/          - sqlc SQL (pr.sql, repo.sql)
│   └── server/               - HTTP handlers
├── docs/                     - architecture + library references
└── web/                      - SolidJS frontend
    └── src/
        ├── components/
        │   ├── TopBar.tsx    - org-grouped repo selector, PR title centered, stats right
        │   ├── PRList.tsx    - PR list for selected repo
        │   ├── PRDetail.tsx  - tabs: description | review
        │   ├── DiffView.tsx  - @pierre/diffs CodeView, driven by /api/diff proxy
        │   ├── Markdown.tsx  - marked + DOMPurify, images allowed
        │   └── ReviewPanel.tsx - SSE-driven review side panel
        ├── queries/          - TanStack Query hooks (useOrgs, useRepos, usePRs, usePRDetail)
        ├── routes/           - PRListPage, PRDetailPage
        ├── router.tsx        - TanStack Router (/$owner/$repo, /$owner/$repo/$pr)
        ├── api.ts            - typed fetch wrappers
        └── style.css         - Gruvbox dark palette
```

## API endpoints

| Method | Path                                       | Description                                                                                                                 |
| ------ | ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------- |
| GET    | /api/orgs                                  | distinct owners from repos table                                                                                            |
| GET    | /api/repos                                 | all repos                                                                                                                   |
| GET    | /api/repos/{owner}                         | repos by owner                                                                                                              |
| POST   | /api/repos                                 | add repo `{owner, name}`                                                                                                    |
| DELETE | /api/repos/{owner}/{name}                  | remove repo                                                                                                                 |
| GET    | /api/prs/{owner}/{repo}                    | list open PRs - always fetched live from GitHub, no cache                                                                   |
| GET    | /api/prs/{owner}/{repo}/{number}           | get PR + file patches - always fetched live from GitHub, no cache                                                           |
| GET    | /api/diff/{owner}/{repo}/{number}          | proxy - fetches full unified diff from GitHub API (`Accept: application/vnd.github.diff`), streams raw patch text to client |
| GET    | /api/asset                                 | authenticated proxy for GitHub-hosted images in PR bodies                                                                   |
| POST   | /api/review/{owner}/{repo}/{number}        | start a review - returns immediately, all progress arrives on the stream                                                    |
| GET    | /api/review/{owner}/{repo}/{number}/stream | SSE for one PR - `snapshot` on connect, then `review` on every state change                                                 |
| GET    | /api/reviews/stream                        | SSE for every PR - `snapshot` is the in-flight list, then `review` per state change                                         |
| GET    | /api/reviews/history                       | paged review history - `?limit&offset`, plus optional `?owner&repo` to scope to one repo. Returns `{sessions, hasMore}`     |

## URL routes

```
/?page=N                 - review history (landing page)
/$owner/$repo            - PR list
/$owner/$repo/$pr        - PR detail (description + review tabs)
```

## Diff rendering

`DiffView.tsx` fetches `/api/diff` → raw unified patch text → `parsePatchFiles()` → `CodeView.setItems()`.

The Go proxy authenticates with the resolved GitHub token. The frontend uses the vanilla JS `CodeView` class directly, not the React wrapper.

```ts
const patches = parsePatchFiles(patch, cacheKeyPrefix)
const items = patches.flatMap(p => p.files.map(fileDiff => ({
  id: fileDiff.name,
  type: 'diff',
  fileDiff,
})))

const view = new CodeView({ theme: { dark: 'pierre-dark', light: 'pierre-light' }, ... })
view.setup(host)
view.setItems(items)
view.render()
```

See `docs/diffs-references/recipe-code-view.md` for full CodeView API.
See `docs/diffs-references/recipe-vanilla.md` for FileDiff single-file usage.

## Theme - Gruvbox (dark, medium contrast)

```css
--base: #282828 --mantle: #3c3836 --crust: #504945 --surface0: #3c3836 --surface1: #504945
  --surface2: #665c54 --text: #ebdbb2 --subtext1: #d5c4a1 --subtext0: #bdae93 --overlay1: #928374
  --green: #b8bb26 --red: #fb4934 --blue: #83a598 --yellow: #fabd2f;
```

`DiffView`'s `CodeView` is locked to the `gruvbox-dark-medium` Shiki theme (single theme name, not a `{dark, light}` pair) so it always matches the app chrome instead of following OS `prefers-color-scheme` - that mismatch (light app UI, OS-dark diff view) was the original bug that prompted the switch away from Evergarden.

## Startup, configuration and onboarding

One binary, four subcommands: `setup`, `doctor`, `serve`, `migrate`.

`serve` loads the config, runs the same preflight checks as `doctor`, and
refuses to start if a required one fails. It never creates the database or
applies migrations - that stays explicit; the failure just names the command
to run. `setup` exits immediately if opencode is not on `PATH`.

`internal/preflight` returns `[]Check` with `OK`/`Warn`/`Fail`, a detail and a
hint. One list, two callers, so `doctor` and `serve` cannot disagree.

### Config

TOML at `~/.config/pr-review/config.toml`, `PR_REVIEW_CONFIG` overrides.
Generated from a commented template, so the file documents itself.

- **Config and token live under `~/.config`, db and worktrees under
  `~/.local/share`** (`$XDG_DATA_HOME`). A SQLite db and cloned repos are not
  config; only the config directory is worth backing up.
- **Setup only asks what it cannot work out** - tmux, editor, auth. Paths and
  ports have defaults and a commented line. Re-running preserves hand-edits.
- **Config is process-level, the db is user-level.** Anything needed before
  serving a request is config; live UI preferences (theme, agent overrides)
  stay in the db.
- `opencode.project_dir` cannot be defaulted safely: it holds `.opencode/` and
  its base64 forms the session deep link. Setup uses the cwd, preflight checks
  `pr-reviewer.md` is really there.
- `REMOTE_HOST` and `LOG_LEVEL` override their config equivalents.

### GitHub auth

`GITHUB_TOKEN`, then `github.token_file` (0600), then `gh auth token` if
`use_gh_cli`. First hit wins; the server logs which source it used.

The primary path is the OAuth device flow - `setup` prints a code, polls,
verifies against `GET /user`, stores it. It needs an OAuth app client ID the
user creates once; GitHub offers no device flow without one. Client IDs are
public. The token is a separate file from the config because config is
pasteable and a token is not.

## Responsive layout

Desktop unchanged. One `@media (max-width: 720px)` block at the end of
`style.css`; the existing 1100px breakpoint already stacks the PR list.

- **`--topbar-h` is no longer load-bearing.** The topbar wraps on narrow
  screens, so `.pr-detail`'s `calc(100vh - var(--topbar-h))` became a silent
  mismeasure. It is `height: 100%` against `.content` now.
- **The files tab stacks panel-first** - `.review-layout` goes column, the
  panel takes `order: -1`, capped at `40vh` and scrolling internally, diff
  takes the rest with `min-height: 0`. Both need definite heights because
  `CodeView` measures its host.
- The panel's collapse toggle is `display: none` above the breakpoint and
  `.collapsed` only applies inside the media query, so a phone collapse cannot
  survive into the desktop layout.

## Makefile

```bash
make setup        # pr-review setup
make doctor       # pr-review doctor
make server       # pr-review serve
make up           # goose migrate up
make down         # goose migrate down
make status       # goose migration status
make reset        # goose reset
make build        # npm build + copy dist + go build bin/pr-review
make dev          # vite dev server on :5173, proxies /api to :7331
make vet          # go build + go vet + tsc -b - the verification command
```

## Review pipeline

`POST /api/review/{owner}/{repo}/{number}` does no work. It calls
`orchestrator.Start(owner, repo, number)`, which creates the review in
`running`, broadcasts it to every SSE subscriber, and returns. Everything
after that happens on the orchestrator's own goroutine and reaches the
client only through the stream.

Stages, in order, each broadcast on entry and on exit:

```
fetch     - GetPR + GetPRDiff (inside the goroutine, not the handler)
checkout  - clone-if-missing + fetch refs/pull/N/head + checkout --detach
session   - opencode session, read-only permissions scoped to the checkout
prompt    - single json_schema-constrained prompt, the long pole
parse     - unmarshal structured output
store     - review session + concerns
```

The GitHub fetch used to run in the handler before `Start`, so the client sat
on a dead POST with nothing on the stream - the fan-out was fine, it was being
starved.

`finish(err)` closes out any stage still marked `running`, so a failure
anywhere can't leave a stage spinning forever in the UI.

## Structured output via a custom tool, not `format.json_schema`

The reviewer used to pass `format: {type: json_schema}` and read
`msg.Info.Structured`. opencode persists `format` on the user message and then
cannot deserialise its own stored value - any `json_schema`, however minimal,
makes `GET /session/{id}/message` return 400 and the session permanently
unopenable.

Replaced with `.opencode/tools/report.ts`, a tool the agent calls once as its
final action. The filename is the tool name; its Zod args are the schema, so
nothing was given up. `reviewer.go` reads `state.input` off the returned
`ToolPart`. The call also shows up as an ordinary part in the transcript, so
what the agent submitted is visible.

`opencode.PromptRequest` still has a `Format` field because the endpoint
accepts one. **Do not use it.**

**The tool is loaded at server startup.** Editing `.opencode/tools/*.ts`
requires restarting `opencode serve`, or the review fails with "agent never
called the report tool".

## Continuing a review in opencode

Every review stores its `opencode_session_id`. opencode's web UI routes
sessions at `/:dir/session/:id`, where `:dir` is base64url (no padding) of the
session's working directory - the `projectDir` it was spawned from.
`opencode.SessionPath` builds that path.

The server emits only the **path**. The browser prepends its own
`location.hostname` and port 4420 (`opencodeUrl` in `review.ts`): the Go server
reaches opencode on `127.0.0.1`, but the link has to resolve in a browser over
Tailscale where loopback is the wrong machine. Server owns the base64 of its
own path, client owns the host it can reach, no config either side.

The link appears when the `session` stage completes, not when the review
finishes, so a running review can be watched live.

## Review history and the global stream

`/` shows review history, newest first, 20 per page. In-flight reviews are
prepended to page 1 only.

History and liveness come from different places by necessity - an in-flight
review has no db row until it finishes - so the page merges
`/api/reviews/history` with the global SSE stream. A PR can legitimately appear
twice (reviewed yesterday, re-reviewed now); both rows are true.

- `Hub` fans out to per-PR (`entries[key].subs`) and `global` subscribers.
  **`Publish` must reach `global` even when no per-PR watcher exists** - the old
  early-return starved them silently.
- `Subscription.pending` is a map keyed by PR, not a single slot. Coalescing by
  replacement drops events for a subscriber watching every PR, so `Take()`
  returns each PR's latest state.
- `SubscribeAll` registers with the hub *before* snapshotting the runner. The
  reverse can lose an event; this order can only duplicate one, and the client
  keys by PR.

No polling. The stream carries whole `Review` values, so one connection drives
the in-progress badge, the page-1 rows and history invalidation.

Paging avoids `COUNT(*)`: ask for `limit+1`, report `hasMore` from the overflow.

## Checkout - one worktree per (PR, head sha)

One clone per repo at `data/repos/{owner}/{repo}`, made with
`--filter=blob:none --no-checkout`. It is an object store; nothing is ever
checked out into it. Working trees live at
`data/worktrees/{owner}/{repo}/{pr}-{sha:12}` via `git worktree add --detach`.

This replaced a single shared checkout that every review force-checked-out,
safe only while reviews were the sole consumer. A tmux session outlives its
request, so a later `--force` swapped the tree under open nvim buffers,
silently, and discarded edits. Keying by sha fixes it: a new push gets a new
directory, so a session on the old sha is never disturbed.

`Worktree()` holds the per-repo lock only for clone, fetch and registry
mutation, then releases - reviews of different PRs in one repo run in parallel.

**The registry and the filesystem are reconciled before either is trusted.**
`git worktree prune` runs before every `add`, or a directory removed without a
prune leaves a record and every later `add` there fails with "missing but
already registered". A tree whose `rev-parse HEAD` doesn't match the expected
sha is torn down, not reused.

Worktrees are never reaped. See `todo.txt`.

## tmux review sessions

Clicking a line in the diff records a pick - one per file, a second click on
the same file overwrites it. `POST /api/tmux/{owner}/{repo}/{number}` resolves
the head sha, takes a worktree, and creates a detached tmux session with one
`nvim +line file` window per pick.

**Removed lines degrade to a file-level pick.** The worktree is at head, so a
deleted line has no honest line number - the same reasoning, and the same
resolution, as `resolve.go` falling back to `fileLevel()` when it cannot locate
a concern. Pointing confidently at the wrong line is worse than not pointing.
A file deleted outright by the PR fails `os.Stat` and lands in `skipped`.

**Line picking semantics come from `@pierre/diffs`**, confirmed in
`InteractionManager.js`: selection starts on pointerdown (a single click picks,
no drag), the pointerdown must land in the line-number gutter
(`requireNumberColumn: true`), and clicking an already-selected line unselects
it and fires the callback with `null`. That `null` is the only un-pick gesture
and carries no file id, so `DiffView` tracks the last selected file itself.
Guarding it away makes picks permanent until reload.

`setSelectedLines(..., { notify: false })` is what stops a side-panel concern
click from registering as a pick.

Session names are `owner/repo/number`, sanitised - tmux forbids `.` and `:` in
session names, and repo names routinely contain dots. Targets are addressed as
`=name` because tmux target matching is otherwise prefix-based.

Commands are passed to tmux as argv, never as a shell string. The paths come
from the API, and tmux 3.x executes multi-argument `shell-command` directly
rather than via `/bin/sh`, so there is nothing to quote and nothing to inject.

`tmux_sessions(owner, repo, pr_number, name, worktree, head_sha, windows)` maps
a PR to its session - the one thing tmux cannot tell us. **tmux remains the
authority on liveness.** `GET` verifies with `has-session` and deletes the row
if it is gone, because a stale row means a blue button offering an attach
command that fails.

The attach command is built per request. `r.Host` is loopback exactly when the
browser is on this machine, and otherwise is a name that already resolves here
from wherever the browser is - so it answers both "local or remote" and "ssh to
what" from one value. `REMOTE_HOST` overrides the target for the case where the
HTTP name is not a valid SSH target; it never applies to a local request.

## Verification

`make vet` is the check to run after any change. `make build` is for producing
binaries, not for verifying - it drags in the vite bundle and the dist copy for
no extra signal.

Do not verify the frontend with a bare `tsc --noEmit`. The root `tsconfig.json`
is solution-style (references only, no `files`), so that command typechecks
nothing and passes vacuously. `tsc -b` follows the project references and is
what `npm run build` actually uses.

## Key decisions

- **Auth** - device-code OAuth, token on disk at 0600, `gh auth token` kept only as a fallback
- **No cache** - PRs, file patches, and diffs are all fetched live from GitHub on every request. A single-user tool never approaches GitHub's 5000 req/hr rate limit, and a cache with no invalidation path is worse than no cache - it was causing PR lists to go stale forever after the first fetch. Removed entirely rather than patched.
- **No React** - SolidJS throughout. `@pierre/diffs` used via vanilla JS API only.
- **DiffsHub** - explored as iframe embed, dropped because localStorage auth can't be injected for private repos. Replicated their approach instead: Go proxy + local CodeView rendering.
- **Repos grouped by org** - single `<optgroup>` dropdown, no separate org selector step.
- **sqlc** - type-safe queries. **goose** - migrations behind `pr-review migrate`, never applied implicitly by `serve`.
- **modernc/sqlite** - pure Go, no CGO.
- **No agent shell commands** - git (clone/fetch/checkout) and other
  deterministic ops run in our own Go code (`internal/checkout`), never
  delegated to an agent via `bash`. Agents only touch things once there's
  no deterministic way to do it ourselves.

See `todo.txt` for the live task list.
