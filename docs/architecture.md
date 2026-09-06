# PR Review — Architecture

A personal PR review tool. Replaces the GitHub review UI with a fast, owned experience.

## Stack

- **Go** — HTTP server, GitHub API client, SQLite via sqlc + goose migrations
- **SolidJS + Vite** — frontend, not React
- **TanStack Query** (solid adapter) — data fetching
- **TanStack Router** (solid adapter) — URL state
- **@pierre/diffs** — diff rendering via CodeView vanilla JS API. See `docs/diffs-skill.md` and `docs/diffs-references/`
- **marked + DOMPurify** — markdown rendering in PR description tab
- **Port:** 7331 by default, `server.addr` in the config

## Project structure

```
pr-review/
├── cmd/
│   ├── pr-review/            — the binary: setup, doctor, serve, migrate
│   │   ├── main.go           — subcommand dispatch, embeds web/dist
│   │   ├── setup.go          — interactive first run
│   │   ├── doctor.go         — preflight report + shared CLI printing
│   │   ├── serve.go          — wiring, from config to listener
│   │   └── migrate.go        — goose runner over the configured db
│   ├── checkout/main.go      — checkout a PR head sha, print the path
│   ├── tmux/main.go          — open files in a tmux session by hand
│   └── opencode/main.go      — one-shot prompt against a running opencode
├── .opencode/
│   ├── agents/pr-reviewer.md — the review agent, git-tracked markdown
│   └── tools/report.ts       — custom tool the agent calls to submit a review
├── internal/
│   ├── config/               — TOML config, defaults, commented file template
│   ├── ghauth/               — device-code login, token resolution + storage
│   ├── preflight/            — dependency and configuration checks
│   ├── term/                 — ANSI constants + tty detection, shared by logs and CLI
│   ├── github/              — read-only GitHub API client
│   ├── checkout/             — per-repo clone + detached checkout at a sha
│   ├── opencode/             — HTTP client for `opencode serve` on :4420
│   ├── reviewer/             — checkout → session → prompt → parse → store
│   ├── orchestrator/         — review lifecycle, SSE fan-out keyed by PR
│   ├── store/                — sqlc-generated queries + Store wrapper
│   │   ├── schema/           — goose migrations (00001_init.sql, 00002_repos.sql)
│   │   └── queries/          — sqlc SQL (pr.sql, repo.sql)
│   └── server/               — HTTP handlers
├── docs/                     — architecture + library references
└── web/                      — SolidJS frontend
    └── src/
        ├── components/
        │   ├── TopBar.tsx    — org-grouped repo selector, PR title centered, stats right
        │   ├── PRList.tsx    — PR list for selected repo
        │   ├── PRDetail.tsx  — tabs: description | review
        │   ├── DiffView.tsx  — @pierre/diffs CodeView, driven by /api/diff proxy
        │   ├── Markdown.tsx  — marked + DOMPurify, images allowed
        │   └── ReviewPanel.tsx — SSE-driven review side panel
        ├── queries/          — TanStack Query hooks (useOrgs, useRepos, usePRs, usePRDetail)
        ├── routes/           — PRListPage, PRDetailPage
        ├── router.tsx        — TanStack Router (/$owner/$repo, /$owner/$repo/$pr)
        ├── api.ts            — typed fetch wrappers
        └── style.css         — Gruvbox dark palette
```

## API endpoints

| Method | Path                                       | Description                                                                                                                 |
| ------ | ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------- |
| GET    | /api/orgs                                  | distinct owners from repos table                                                                                            |
| GET    | /api/repos                                 | all repos                                                                                                                   |
| GET    | /api/repos/{owner}                         | repos by owner                                                                                                              |
| POST   | /api/repos                                 | add repo `{owner, name}`                                                                                                    |
| DELETE | /api/repos/{owner}/{name}                  | remove repo                                                                                                                 |
| GET    | /api/prs/{owner}/{repo}                    | list open PRs — always fetched live from GitHub, no cache                                                                   |
| GET    | /api/prs/{owner}/{repo}/{number}           | get PR + file patches — always fetched live from GitHub, no cache                                                           |
| GET    | /api/diff/{owner}/{repo}/{number}          | proxy — fetches full unified diff from GitHub API (`Accept: application/vnd.github.diff`), streams raw patch text to client |
| GET    | /api/asset                                 | authenticated proxy for GitHub-hosted images in PR bodies                                                                   |
| POST   | /api/review/{owner}/{repo}/{number}        | start a review — returns immediately, all progress arrives on the stream                                                    |
| GET    | /api/review/{owner}/{repo}/{number}/stream | SSE for one PR — `snapshot` on connect, then `review` on every state change                                                 |
| GET    | /api/reviews/stream                        | SSE for every PR — `snapshot` is the in-flight list, then `review` per state change                                         |
| GET    | /api/reviews/history                       | paged review history — `?limit&offset`, plus optional `?owner&repo` to scope to one repo. Returns `{sessions, hasMore}`     |

## URL routes

```
/?page=N                 — review history (landing page)
/$owner/$repo            — PR list
/$owner/$repo/$pr        — PR detail (description + review tabs)
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

## Theme — Gruvbox (dark, medium contrast)

```css
--base: #282828 --mantle: #3c3836 --crust: #504945 --surface0: #3c3836 --surface1: #504945
  --surface2: #665c54 --text: #ebdbb2 --subtext1: #d5c4a1 --subtext0: #bdae93 --overlay1: #928374
  --green: #b8bb26 --red: #fb4934 --blue: #83a598 --yellow: #fabd2f;
```

`DiffView`'s `CodeView` is locked to the `gruvbox-dark-medium` Shiki theme (single theme name, not a `{dark, light}` pair) so it always matches the app chrome instead of following OS `prefers-color-scheme` — that mismatch (light app UI, OS-dark diff view) was the original bug that prompted the switch away from Evergarden.

## Startup, configuration and onboarding

One binary, four subcommands: `setup`, `doctor`, `serve`, `migrate`. The old
`cmd/server` and `cmd/migrate` are gone — a shipped tool that needs you to know
which of two binaries to run has already failed at onboarding.

**Nothing is implicit at startup.** `serve` loads the config, runs the same
preflight checks as `doctor`, and refuses to start if a required one fails,
printing the same annotated list with a fix for each line. It does **not**
create the database and does **not** apply migrations — that stays an explicit
act, as it always has. The difference is that the failure now names the command
to run instead of surfacing as a SQL error three layers down.

**opencode is asserted before anything else.** `setup` exits immediately if it
is not on `PATH`, rather than collecting answers and reporting the failure at
the end. It runs every review; a config written without it is a config for a
tool that cannot do its one job.

`internal/preflight` returns `[]Check` with an `OK`/`Warn`/`Fail` status, a
detail, and a hint. Required dependencies fail; optional ones warn. It is one
list consumed by two callers, so `doctor` and `serve` can never disagree about
what a healthy install looks like.

### Config

TOML at `~/.config/pr-review/config.toml`, overridable with `PR_REVIEW_CONFIG`.
The file is generated from a commented template, so the artefact on disk
documents itself and there is no second copy of the docs to drift.

**Everything user-owned lives under that one directory** — config, database,
worktrees, token. Splitting across `~/.config` and `~/.local/share` is the
correct XDG reading, and it is the wrong call here: it doubles the number of
places to back up, delete or point at another disk, for a tool with a single
user. One directory, one thing to move.

**Setup only asks what it cannot work out.** tmux on or off, which editor, and
how to authenticate. Paths, ports and the opencode URL all have workable
defaults and a commented line in the file — asking about them makes onboarding
longer without making it better informed, since a first-time user has no basis
to answer. Re-running setup preserves hand-edits.

**The split is process-level vs. user-level.** Anything the process needs before
it can serve a request — listen address, database and data paths, opencode URL
and project dir, tmux editor and window cap, GitHub auth — is config. Anything
that is a live UI preference — theme, per-agent model and prompt overrides —
stays in the database, reachable from the UI. Duplicating either across both
would create two sources of truth with no arbitration.

`opencode.project_dir` is load-bearing and cannot sensibly be defaulted: it is
where `.opencode/agents` and `.opencode/tools` are read from at opencode
startup, and it is also the directory whose base64 forms the session deep link.
`setup` defaults it to the working directory and preflight verifies
`pr-reviewer.md` actually exists under it.

`REMOTE_HOST` and `LOG_LEVEL` still override their config equivalents, because
both are things you want to flip for one run without editing a file.

### GitHub auth

`gh auth token` is no longer the mechanism, only the last fallback. Resolution
order is `GITHUB_TOKEN`, then `github.token_file` (0600), then `gh` if
`use_gh_cli` is on. First hit wins, and the server logs which source it used —
"which token is this even using" is otherwise unanswerable.

The primary path is the OAuth device flow: `setup` prints a user code and a
URL, polls, verifies the result against `GET /user`, and stores it. It needs an
OAuth app client ID, which the user must create once. That is real friction and
there is no way around it — GitHub has no device flow without a client ID — so
`setup` states the exact steps rather than failing with a 401. Client IDs are
public; treating one as a secret would be cargo-culting.

The token is a separate file from the config on purpose. Config is something
you might paste into an issue; a token is not.

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
make vet          # go build + go vet + tsc -b — the verification command
```

## Review pipeline

`POST /api/review/{owner}/{repo}/{number}` does no work. It calls
`orchestrator.Start(owner, repo, number)`, which creates the review in
`running`, broadcasts it to every SSE subscriber, and returns. Everything
after that happens on the orchestrator's own goroutine and reaches the
client only through the stream.

Stages, in order, each broadcast on entry and on exit:

```
fetch     — GetPR + GetPRDiff (inside the goroutine, not the handler)
checkout  — clone-if-missing + fetch refs/pull/N/head + checkout --detach
session   — opencode session, read-only permissions scoped to the checkout
prompt    — single json_schema-constrained prompt, the long pole
parse     — unmarshal structured output
store     — review session + concerns
```

The GitHub fetch used to run in the HTTP handler before `Start` was called,
which meant the client sat on a dead POST for a second or two with nothing
on the stream. That was the whole "client is blind" bug — the fan-out was
always fine, it was being starved.

`finish(err)` closes out any stage still marked `running`, so a failure
anywhere can't leave a stage spinning forever in the UI.

## Structured output via a custom tool, not `format.json_schema`

The reviewer used to pass `format: {type: json_schema}` on the prompt and read
`msg.Info.Structured`. That worked, but opencode persists `format` on the user
message and then cannot deserialise its own stored value on read — any
`json_schema` value, including a minimal three-line one, makes
`GET /session/{id}/message` return 400. The review succeeded and the session
became permanently unopenable in the web UI and via `opencode attach`.

Replaced with `.opencode/tools/report.ts`, a custom tool the agent calls once
as its final action. The filename is the tool name. Its Zod args are the
schema — that is where enforcement lives now, so nothing was given up by
dropping `json_schema`. `reviewer.go` reads the call's `state.input` off the
returned `ToolPart` (`MessageResponse.ToolInput`). No `format` is ever sent,
so review sessions read back cleanly and are fully browsable.

`opencode.PromptRequest` still has a `Format` field because the endpoint
accepts one. **Do not use it.** It writes sessions that cannot be read back.

A custom tool is also strictly better than a schema here: the tool call shows
up as an ordinary part in the transcript, so what the agent submitted is
visible in the session rather than hidden in message metadata.

**The tool is loaded at server startup.** Editing `.opencode/tools/*.ts`
requires restarting `opencode serve`; a running server will not pick it up,
and the review fails with "agent never called the report tool".

## Continuing a review in opencode

Every review already stores its `opencode_session_id`. The web UI that
`opencode serve` exposes on :4420 routes sessions at `/:dir/session/:id`,
where `:dir` is base64url (no padding) of the session's working directory —
`/home/schultz/toolbox/pr-review`, the `projectDir` the server was spawned
from. `opencode.SessionPath` builds that path.

The server emits only the **path**, never a full URL. The browser prepends
its own `location.hostname` and port 4420 (`opencodeUrl` in `review.ts`).
That's deliberate: the Go server reaches opencode on `127.0.0.1`, but the
link has to resolve in a browser on the work laptop over Tailscale, where
loopback is the wrong machine. Server owns the base64 of its own filesystem
path; the client owns the host it can actually reach. No config either side.

The link appears as soon as the `session` stage completes, not when the
review finishes, so a running review can be watched live. This replaces any
in-app pushback/follow-up flow — opencode already owns the transcript.

## Review history and the global stream

`/` is the landing page and shows review history, newest first, 20 per page.
In-flight reviews are prepended to page 1 only — they are the newest thing
there is, and "in progress" is meaningless on page 3.

History and liveness come from different places, and that is inherent, not a
wart: an in-flight review has no database row until it finishes. So the page
merges two sources client-side — `/api/reviews/history` for stored rows and
the global SSE stream for in-flight ones. A PR can legitimately appear twice
(reviewed yesterday, being re-reviewed now); both rows are true.

`Hub` fans out to two sets of subscribers: per-PR (`entries[key].subs`) and
global (`global`). `Publish` reaches global subscribers **even when no per-PR
watcher exists** — the old early-return would have silently starved them.

`Subscription.pending` is a map keyed by PR, not a single slot. Coalescing by
replacement is correct for one PR (latest state wins) but drops events when a
subscriber watches all of them, so `Take()` returns every distinct PR's latest
state rather than one review.

`SubscribeAll` registers with the hub _before_ snapshotting the runner. The
reverse order can lose an event in the gap; this order can only duplicate one,
and the client keys by PR so a duplicate is a no-op.

No polling. The stream carries whole `Review` values rather than a "something
changed" ping, so the same events drive the in-progress badge, the page-1 rows,
and history invalidation on completion — one connection, three consumers.

Paging avoids `COUNT(*)`: the handler asks for `limit+1` rows and reports
`hasMore` from the overflow.

## Checkout — one worktree per (PR, head sha)

One clone per repo at `data/repos/{owner}/{repo}`, created with
`--filter=blob:none --no-checkout` so the first clone is cheap and blobs are
fetched lazily. It is an object store, not a working tree — nothing is ever
checked out into it.

Working trees live at `data/worktrees/{owner}/{repo}/{pr}-{sha:12}`, one per
PR head, created with `git worktree add --detach`.

This replaced a single shared checkout per repo that every review
force-checked-out. That was safe only while reviews were the sole consumer
and the per-repo lock was held for the whole review. **tmux sessions broke
that invariant**: the session outlives the request that created it, so a
later `checkout --force` would swap the tree under open nvim buffers —
silently, because nvim doesn't reread on disk change. Writing then put one
PR's content into another PR's tree, and `--force` discarded any edits
outright.

Keying by sha rather than by PR is what makes it safe: a new push produces a
new directory, so a session on the old sha is never disturbed. Re-reviewing
the same sha reuses the same tree, which is correct — nothing changed.

`Worktree()` holds the per-repo lock only while shared state is mutated —
clone, fetch, worktree registry — and releases before returning. Reviews of
different PRs in the same repo now run in parallel; previously they
serialised.

**The registry and the filesystem are reconciled before either is trusted.**
`git worktree prune` runs before every `add`, because a directory removed
without a prune leaves a record behind and every later `add` at that path
fails with "missing but already registered". A directory that exists but
whose `rev-parse HEAD` doesn't match the expected sha is torn down and
rebuilt rather than reused.

Worktrees are never reaped. See `todo.txt`.

## tmux review sessions

Clicking a line in the diff records a pick — one per file, a second click on
the same file overwrites it. `POST /api/tmux/{owner}/{repo}/{number}` resolves
the head sha, takes a worktree, and creates a detached tmux session with one
`nvim +line file` window per pick.

**Removed lines degrade to a file-level pick.** The worktree is at head, so a
deleted line has no honest line number — the same reasoning, and the same
resolution, as `resolve.go` falling back to `fileLevel()` when it cannot locate
a concern. Pointing confidently at the wrong line is worse than not pointing.
A file deleted outright by the PR fails `os.Stat` and lands in `skipped`.

**Line picking rides `onSelectedLinesChange`, and the semantics come from
`@pierre/diffs`, not from us.** Three behaviours are load-bearing and all are
confirmed in `InteractionManager.js`:

- Selection starts on **pointerdown**, so a single click picks — no drag needed.
- The pointerdown must land in the **line-number gutter**
  (`requireNumberColumn: true`). Clicking the code body does nothing.
- Clicking an already-selected single line **unselects** it and fires the
  callback with `null`. That is the only un-pick gesture, and the callback
  carries no file id, so `DiffView` tracks the last selected file itself and
  reports it to `onUnpickLine`. Dropping the `null` — which is the obvious
  defensive guard to write — makes picks permanent until reload.

`setSelectedLines(..., { notify: false })` is what stops a side-panel concern
click from registering as a pick; the flag is forwarded down to the interaction
manager, so it genuinely suppresses the callback.

Session names are `owner/repo/number`, sanitised — tmux forbids `.` and `:` in
session names, and repo names routinely contain dots. Targets are addressed as
`=name` because tmux target matching is otherwise prefix-based.

Commands are passed to tmux as argv, never as a shell string. The paths come
from the API, and tmux 3.x executes multi-argument `shell-command` directly
rather than via `/bin/sh`, so there is nothing to quote and nothing to inject.

`tmux_sessions(owner, repo, pr_number, name, worktree, head_sha, windows)` maps
a PR to its session — the one thing tmux cannot tell us. **tmux remains the
authority on liveness.** `GET` verifies with `has-session` and deletes the row
if it is gone, because a stale row means a blue button offering an attach
command that fails.

The attach command is built per request. `r.Host` is loopback exactly when the
browser is on this machine, and otherwise is a name that already resolves here
from wherever the browser is — so it answers both "local or remote" and "ssh to
what" from one value. `REMOTE_HOST` overrides the target for the case where the
HTTP name is not a valid SSH target; it never applies to a local request.

## Verification

`make vet` is the check to run after any change. `make build` is for producing
binaries, not for verifying — it drags in the vite bundle and the dist copy for
no extra signal.

Do not verify the frontend with a bare `tsc --noEmit`. The root `tsconfig.json`
is solution-style (references only, no `files`), so that command typechecks
nothing and passes vacuously. `tsc -b` follows the project references and is
what `npm run build` actually uses.

## Key decisions

- **Auth** — device-code OAuth, token on disk at 0600, `gh auth token` kept only as a fallback
- **No cache** — PRs, file patches, and diffs are all fetched live from GitHub on every request. A single-user tool never approaches GitHub's 5000 req/hr rate limit, and a cache with no invalidation path is worse than no cache — it was causing PR lists to go stale forever after the first fetch. Removed entirely rather than patched.
- **No React** — SolidJS throughout. `@pierre/diffs` used via vanilla JS API only.
- **DiffsHub** — explored as iframe embed, dropped because localStorage auth can't be injected for private repos. Replicated their approach instead: Go proxy + local CodeView rendering.
- **Repos grouped by org** — single `<optgroup>` dropdown, no separate org selector step.
- **sqlc** — type-safe queries. **goose** — migrations behind `pr-review migrate`, never applied implicitly by `serve`.
- **modernc/sqlite** — pure Go, no CGO.
- **No agent shell commands** — git (clone/fetch/checkout) and other
  deterministic ops run in our own Go code (`internal/checkout`), never
  delegated to an agent via `bash`. Agents only touch things once there's
  no deterministic way to do it ourselves.

See `todo.txt` for the live task list.
