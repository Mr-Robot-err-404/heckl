# PR Review — Architecture

A personal PR review tool. Replaces the GitHub review UI with a fast, owned experience.

## Stack

- **Go** — HTTP server, GitHub API client, SQLite via sqlc + goose migrations
- **SolidJS + Vite** — frontend, not React
- **TanStack Query** (solid adapter) — data fetching
- **TanStack Router** (solid adapter) — URL state
- **@pierre/diffs** — diff rendering via CodeView vanilla JS API. See `docs/diffs-skill.md` and `docs/diffs-references/`
- **marked + DOMPurify** — markdown rendering in PR description tab
- **Port:** 7331

## Project structure

```
pr-review/
├── cmd/
│   ├── server/main.go        — HTTP server entrypoint, embeds web/dist
│   ├── migrate/main.go       — goose migration runner
│   ├── checkout/main.go      — checkout a PR head sha, print the path
│   └── opencode/main.go      — one-shot prompt against a running opencode
├── internal/
│   ├── github/               — read-only GitHub API client, auth via `gh auth token`
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

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/orgs | distinct owners from repos table |
| GET | /api/repos | all repos |
| GET | /api/repos/{owner} | repos by owner |
| POST | /api/repos | add repo `{owner, name}` |
| DELETE | /api/repos/{owner}/{name} | remove repo |
| GET | /api/prs/{owner}/{repo} | list open PRs — always fetched live from GitHub, no cache |
| GET | /api/prs/{owner}/{repo}/{number} | get PR + file patches — always fetched live from GitHub, no cache |
| GET | /api/diff/{owner}/{repo}/{number} | proxy — fetches full unified diff from GitHub API (`Accept: application/vnd.github.diff`), streams raw patch text to client |
| GET | /api/asset | authenticated proxy for GitHub-hosted images in PR bodies |
| POST | /api/review/{owner}/{repo}/{number} | start a review — returns immediately, all progress arrives on the stream |
| GET | /api/review/{owner}/{repo}/{number} | persisted review sessions + concerns for this PR |
| GET | /api/review/{owner}/{repo}/{number}/live | current in-memory review, or null |
| GET | /api/review/{owner}/{repo}/{number}/stream | SSE — `snapshot` on connect, then `review` on every state change |

## URL routes

```
/                        — empty state
/$owner/$repo            — PR list
/$owner/$repo/$pr        — PR detail (description + review tabs)
```

## Diff rendering

`DiffView.tsx` fetches `/api/diff` → raw unified patch text → `parsePatchFiles()` → `CodeView.setItems()`.

The Go proxy authenticates with `gh auth token` — no manual PAT needed. The frontend uses the vanilla JS `CodeView` class directly, not the React wrapper.

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
--base:     #282828
--mantle:   #3c3836
--crust:    #504945
--surface0: #3c3836
--surface1: #504945
--surface2: #665c54
--text:     #ebdbb2
--subtext1: #d5c4a1
--subtext0: #bdae93
--overlay1: #928374
--green:    #b8bb26
--red:      #fb4934
--blue:     #83a598
--yellow:   #fabd2f
```

`DiffView`'s `CodeView` is locked to the `gruvbox-dark-medium` Shiki theme (single theme name, not a `{dark, light}` pair) so it always matches the app chrome instead of following OS `prefers-color-scheme` — that mismatch (light app UI, OS-dark diff view) was the original bug that prompted the switch away from Evergarden.

## Makefile

```bash
make server       # go run ./cmd/server
make up           # goose migrate up
make down         # goose migrate down
make status       # goose migration status
make reset        # goose reset
make build        # npm build + copy dist + go build binaries
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

## Checkout, not worktrees

One plain clone per repo at `data/repos/{owner}/{repo}`, created with
`--filter=blob:none --no-checkout` so the first clone is cheap and blobs
are fetched lazily for the shas actually reviewed.

`Acquire` takes a per-repo lock and returns a `Handle`; the caller holds it
for the whole review and `Release()`s on defer. This serialises reviews of
two PRs in the same repo — acceptable for a single user, and the tradeoff
for deleting the entire worktree bookkeeping layer. Reviews are read-only,
so there is nothing a worktree bought us.

## Verification

`make vet` is the check to run after any change. `make build` is for producing
binaries, not for verifying — it drags in the vite bundle and the dist copy for
no extra signal.

Do not verify the frontend with a bare `tsc --noEmit`. The root `tsconfig.json`
is solution-style (references only, no `files`), so that command typechecks
nothing and passes vacuously. `tsc -b` follows the project references and is
what `npm run build` actually uses.

## Key decisions

- **Auth** — `gh auth token` at startup, no PAT management
- **No cache** — PRs, file patches, and diffs are all fetched live from GitHub on every request. A single-user tool never approaches GitHub's 5000 req/hr rate limit, and a cache with no invalidation path is worse than no cache — it was causing PR lists to go stale forever after the first fetch. Removed entirely rather than patched.
- **No React** — SolidJS throughout. `@pierre/diffs` used via vanilla JS API only.
- **DiffsHub** — explored as iframe embed, dropped because localStorage auth can't be injected for private repos. Replicated their approach instead: Go proxy + local CodeView rendering.
- **Repos grouped by org** — single `<optgroup>` dropdown, no separate org selector step.
- **sqlc** — type-safe queries. **goose** — migrations in a separate `cmd/migrate` binary, not run on server startup.
- **modernc/sqlite** — pure Go, no CGO.
- **No agent shell commands** — git (clone/fetch/checkout) and other
  deterministic ops run in our own Go code (`internal/checkout`), never
  delegated to an agent via `bash`. Agents only touch things once there's
  no deterministic way to do it ourselves.

See `todo.txt` for the live task list.
