# Heckl - Architecture

Local PR review with your opencode agents. One Go binary, embedded SolidJS UI,
SQLite.

## Stack

Go + sqlc + goose. SolidJS/Vite, TanStack Query and Router. `@pierre/diffs`
via the vanilla `CodeView` API (see `docs/diffs-references/`). marked +
DOMPurify for PR bodies. Binds `127.0.0.1:7331` by default.

## Layout

```
cmd/heckl/          setup, doctor, serve, migrate. main.go embeds web/dist
internal/bundle/    agents/*.md + tools/report.ts, embedded
internal/config/    TOML, defaults, commented template
internal/ghauth/    device flow, token resolution
internal/preflight/ Check list shared by doctor and serve
internal/github/    read-only API client
internal/checkout/  clone + worktree per (PR, sha)
internal/opencode/  client for `opencode serve`
internal/reviewer/  checkout -> session -> prompt -> parse -> store
internal/orchestrator/ lifecycle + SSE fan-out
internal/store/     sqlc queries, schema/ migrations
internal/server/    HTTP handlers
web/src/            components, queries, routes, style.css
```

`.opencode/agents` and `.opencode/tools` in the checkout are symlinks into
`internal/bundle`. One source of truth, embedded and served from the same files.

## API

```
GET    /api/orgs | /api/repos | /api/repos/{owner}
POST   /api/repos                              {owner, name}
DELETE /api/repos/{owner}/{name}
GET    /api/prs/{owner}/{repo}[/{number}]      live, never cached
GET    /api/diff/{owner}/{repo}/{number}       proxy, raw unified patch
GET    /api/blob/{owner}/{repo}/{number}       ?path&prev&baseSha&baseRef&headSha&headRef
POST   /api/prefetch/{owner}/{repo}            {base,head}: {sha, ref, paths}
GET    /api/asset                              authenticated image proxy
POST   /api/review/{owner}/{repo}/{number}     returns immediately
GET    /api/review/{owner}/{repo}/{number}/stream   SSE, one PR
GET    /api/reviews/stream                     SSE, all PRs
GET    /api/reviews/history                    ?limit&offset[&owner&repo]
```

Routes: `/?page=N` history, `/$owner/$repo` list, `/$owner/$repo/$pr` detail.

## Startup and config

TOML at `~/.config/heckl/config.toml`, `HECKL_CONFIG` overrides, generated from
a commented template so the file documents itself. Config and token under
`~/.config`; db and worktrees under `~/.local/share`. Setup only asks what it
cannot work out, and re-running preserves hand-edits.

`serve` runs the same `preflight` checks as `doctor` and refuses to start on a
failure. It never creates the db or applies migrations; the failure names the
command instead.

**Binds loopback by default.** There is no auth in front of the UI and it can
post comments as you. `0.0.0.0:7331` to expose it, or put a proxy in front.
`serve` logs the bound address only after `net.Listen` returns.

## Agents

`internal/bundle` embeds `agents/*.md` and `tools/report.ts`. `setup` writes
any file missing from `project_dir/.opencode/` and **never overwrites**, so
edits survive upgrades. Deleting a file and re-running setup restores the
shipped version. New agents in a later release appear because they are absent.

The cost is deliberate: an edited agent never receives prompt improvements.

`project_dir` defaults to `~/.local/share/heckl`. Its base64 also forms the
opencode session deep link. Both directories are read once at `opencode serve`
startup, so edits need a restart.

`@opencode-ai/plugin` is installed by opencode itself, not shipped, so the
version tracks the user's opencode. Preflight warns rather than fails if it is
absent, since a fresh setup resolves it on first start.

## GitHub auth

`GITHUB_TOKEN`, then `token_file` (0600), then `gh auth token` if `use_gh_cli`.
First hit wins; the server logs the source.

Device flow is the front door. The OAuth client ID is baked in
(`ghauth.DefaultClientID`) because device flow has no secret and an ID is
public by construction; `github.oauth_client_id` overrides it for GHE. Scopes
are `repo read:org` - broad, but `repo` is the only classic scope that reads
private diffs, and device flow cannot issue fine-grained tokens.

Tokens do not expire, because expiry requires refresh handling that does not
exist yet. See `todo.txt`.

## Review pipeline

`POST /api/review/...` does no work. `orchestrator.Start` creates the review in
`running`, broadcasts, and returns. Everything else runs on the orchestrator
goroutine and reaches the client only on the stream.

```
fetch -> checkout -> session -> prompt -> parse -> store
```

Each stage broadcasts on entry and exit. `finish(err)` closes any stage still
`running`, so a failure cannot leave one spinning in the UI.

**Structured output comes from a tool call, not `format.json_schema`.** opencode
persists `format` on the user message and then cannot deserialise it: any
`json_schema` makes `GET /session/{id}/message` return 400 and the session
permanently unopenable. `PromptRequest.Format` exists because the endpoint
accepts it. **Do not use it.** `report.ts` is called once as the agent's final
action; the filename is the tool name and its Zod args are the schema.
`reviewer.go` reads `state.input` off the returned `ToolPart`. A completed
`report` call defines success, not the absence of an error.

`Agent` must be passed on the `Prompt` call, not just `CreateSession`.

## Streams

`Hub` fans out to per-PR and `global` subscribers. **`Publish` must reach
`global` unconditionally**; an early return starves them silently.
`Subscription.pending` is keyed by PR and `Take()` returns a slice, because
coalescing into one slot drops events for an all-PR subscriber. `SubscribeAll`
registers before snapshotting: the reverse can lose an event, this order can
only duplicate one, and the client keys by PR.

History and liveness come from different places by necessity, since an
in-flight review has no db row. A PR appearing twice is legitimate. Paging
avoids `COUNT(*)` by asking for `limit+1`.

## Checkout

One clone per repo at `data/repos/{owner}/{repo}`, `--filter=blob:none
--no-checkout`; it is an object store and nothing is checked out into it.
Worktrees at `data/worktrees/{owner}/{repo}/{pr}-{sha:12}`.

Keying by sha replaced a shared checkout that force-checked-out under open nvim
buffers and discarded edits. **`git worktree prune` runs before every `add`**,
or a manually removed directory leaves a record that fails every later `add`. A
tree whose `rev-parse HEAD` does not match is torn down, not reused. Worktree
links are absolute and stored twice, so a moved data dir silently re-clones.

Worktrees are never reaped.

### Blob prefetch

The clone is blobless, so blobs are not local and `git show <sha>:<path>` on a
missing one lazily fetches it from the promisor remote as **one object per
network roundtrip**. Measured on `ghostty-org/ghostty`: 1288 ms cold, 2 ms warm.
The blob endpoint reads both sides, so an unwarmed expand paid two of those plus
a `GetPR` call, which is where the multi-second first click per file came from.

`git archive --format=tar <sha> -- <paths>` goes through git's bulk promisor
path and pulls every missing blob in a single request. It is flat in file count:
6 files 2642 ms, 40 files 1949 ms, against 7852 ms for the same 6 one at a time.
`cat-file --batch` does **not** batch (7693 ms for 6) and is not an alternative.
The tar goes to `/dev/null`; the point is the objects it leaves behind.

Consequences that shaped the design:

- **Warm is 5 ms**, so `Prefetch` is called unconditionally and needs no cache
  state, TTL or invalidation. The in-flight map only stops concurrent duplicates.
- **`git archive` is fail-fast on a pathspec that matches nothing**, so a file
  added or deleted by the PR would abort the whole prefetch for the side it is
  absent from. `existingPaths` filters via `ls-tree`, which reads only trees and
  costs 3 ms. Do not use `ls-tree -l`: reporting size needs the blob, so it
  lazy-fetches every one (17.9 s on 1351 entries).
- **Fetched objects are permanent and reachability does not matter.** Verified:
  `gc --prune=now` took 36 promisor packs to 1 and kept a blob reachable from no
  ref at all. Git retains promisor-pack contents regardless of refs. So PR head
  commits can stay `FETCH_HEAD`-only, revisiting a PR is warm forever, and `gc`
  is safe to run against the pack growth.
- **The per-repo lock covers the clone only.** `FileAt` and `Prefetch` run their
  git commands unlocked, or an expand would queue behind an in-progress prefetch
  or review checkout and end up slower than before. Git's object store is safe
  for concurrent reads and pack writes.
- Content addressing means a PR that gains commits is cold only for content that
  genuinely changed; the base side and untouched files stay warm.

Prefetch is client-driven: the frontend has already parsed the patch into files
and holds the shas, so the server does no diff parsing and makes no GitHub call.
Capped at `maxPrefetchPaths` (50) per side, beyond which expand falls back to
per-click fetching. A viewport-aware scheme that warms files as they approach the
screen is the intended follow-up.

Three layers cover the rest:

- `/api/blob` is a real query (`blobOptions`, `staleTime: Infinity`, keyed on the
  head sha so a force-push invalidates naturally) and `loadDiffFiles` goes
  through `fetchQuery`. Re-expanding a file after the `CodeView` is rebuilt (a
  theme change does this) is now a cache hit rather than a refetch.
- Hovering an expand button `prefetchQuery`s that file. This is what covers files
  past the 50 cap, moving their cost from the click to the hover. Delegated from
  a single `mouseover` on the host: the buttons live in per-file shadow roots,
  but the event is composed, so `composedPath` finds both the button and the
  shadow host, and the host's slotted `.diff-collapse-slot[data-file]` names the
  file. Repeat mouseovers are free, tanstack dedupes in-flight and fresh keys.
- A spinner on the button after `SPINNER_DELAY_MS`, since a warm expand is 2ms
  and would only flicker. `pointer-events: none` while in flight, styled from
  `unsafeCSS` through `:host([data-expanding])`, which is the only way to reach
  inside the shadow root. `FileDiff.loadFilesIfNecessary` already drops a second
  load for a file with one pending, so this is feedback, not request control.

## tmux

A click in the diff records one pick per file. Sessions are named
`owner/repo/number`, sanitised, since tmux forbids `.` and `:`; targets use
`=name` because matching is otherwise prefix-based. Commands are argv, never a
shell string. `tmux_sessions.worktree` is absolute.

**tmux is the authority on liveness**, not the db: `GET` verifies with
`has-session` and deletes stale rows.

**Removed lines degrade to a file-level pick.** The worktree is at head, so a
deleted line has no honest number. Same reasoning as `resolve.go` falling back
to `fileLevel()`.

Picking semantics come from `@pierre/diffs`: pointerdown must land in the line
number gutter (`requireNumberColumn: true`), and re-clicking fires the callback
with `null`. That `null` is the only un-pick gesture and carries no file id, so
`DiffView` tracks the last selected file. `setSelectedLines(..., { notify:
false })` stops a side-panel click registering as a pick.

The attach command is built per request: `r.Host` is loopback exactly when the
browser is local, and otherwise already resolves here. `REMOTE_HOST` overrides
the SSH target and never applies to a local request.

## Frontend notes

Themes need registering in three places: the CSS block, `web/src/theme.ts`,
`internal/server/theme.go`. `CodeView` is locked to a single Shiki theme name
rather than a `{dark, light}` pair, so it cannot follow OS
`prefers-color-scheme` against a light app chrome.

Signals holding arrays of objects break `<For>`; use `createStore`. Icons live
in `web/src/components/icons.tsx`. `navigator.clipboard` is undefined on http
to a non-localhost host, so use `clipboard.copyText`.

`.pr-tabs` has no left padding; the gutter is on `.pr-tab:first-child`.
`display: contents` is what makes `.agent-picker` children grid items. A
`<select>` has an intrinsic min-width from its widest option, so `min-width: 0`
is required for a mobile cap to hold.

## Verification

`make vet` - `go build`, `go vet`, `tsc -b`. Never a bare `tsc --noEmit`: the
root tsconfig is solution-style and passes vacuously.

`cmd/heckl/dist` must exist before `go build`, since `//go:embed` reads at
compile time. The `all:` prefix matters; plain `//go:embed dist` skips dotfiles
and 404s at runtime with no build error.

`make redo` for schema edits, not `make reset` (which is goose reset, not a
state wipe). `goose.CollectMigrations` returns `ErrNoMigrationFiles` when
nothing is pending; zero pending is not an error.

## Decisions

- **No agent shell commands.** git and other deterministic work runs in Go
  (`internal/checkout`), never delegated via `bash`.
- `@pierre/diffs` via vanilla API.
- **modernc/sqlite**, pure Go, so cross-compiling is `CGO_ENABLED=0` plus
  `GOOS`/`GOARCH`.
- **DiffsHub** dropped as an iframe embed: localStorage auth cannot be injected
  for private repos. Replicated with a Go proxy and local rendering.

See `todo.txt`.
