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
│   └── migrate/main.go       — goose migration runner
├── internal/
│   ├── github/               — read-only GitHub API client, auth via `gh auth token`
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
        │   └── Markdown.tsx  — marked + DOMPurify, images allowed
        ├── queries/          — TanStack Query hooks (useOrgs, useRepos, usePRs, usePRDetail)
        ├── routes/           — PRListPage, PRDetailPage
        ├── router.tsx        — TanStack Router (/$owner/$repo, /$owner/$repo/$pr)
        ├── api.ts            — typed fetch wrappers
        └── style.css         — Evergarden summer palette (light)
```

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/orgs | distinct owners from repos table |
| GET | /api/repos | all repos |
| GET | /api/repos/{owner} | repos by owner |
| POST | /api/repos | add repo `{owner, name}` |
| DELETE | /api/repos/{owner}/{name} | remove repo |
| GET | /api/prs/{owner}/{repo} | list open PRs — fetches from GitHub, caches in SQLite |
| GET | /api/prs/{owner}/{repo}/{number} | get PR + file patches from cache |
| GET | /api/diff/{owner}/{repo}/{number} | proxy — fetches full unified diff from GitHub API (`Accept: application/vnd.github.diff`), streams raw patch text to client |

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

## Theme — Evergarden summer

```css
--base:     #f5efe6
--mantle:   #f2eae1
--crust:    #e8ded5
--surface0: #ede8dd
--surface1: #e6e1d3
--surface2: #ceccbd
--text:     #2b3034
--subtext1: #455355
--subtext0: #576869
--overlay1: #829084
--green:    #91a77a
--red:      #c58687
--blue:     #8294ad
--yellow:   #c4aa80
```

## Makefile

```bash
make server       # go run ./cmd/server
make up           # goose migrate up
make down         # goose migrate down
make status       # goose migration status
make reset        # goose reset
make build        # npm build + copy dist + go build binaries
make dev          # vite dev server on :5173, proxies /api to :7331
```

## Key decisions

- **Auth** — `gh auth token` at startup, no PAT management
- **Cache** — PRs + per-file patches in SQLite. Diff proxy is live and requires network.
- **No React** — SolidJS throughout. `@pierre/diffs` used via vanilla JS API only.
- **DiffsHub** — explored as iframe embed, dropped because localStorage auth can't be injected for private repos. Replicated their approach instead: Go proxy + local CodeView rendering.
- **Repos grouped by org** — single `<optgroup>` dropdown, no separate org selector step.
- **sqlc** — type-safe queries. **goose** — migrations in a separate `cmd/migrate` binary, not run on server startup.
- **modernc/sqlite** — pure Go, no CGO.

## Todo

- [ ] Sync button — force re-fetch a PR from GitHub
- [ ] PR file tree sidebar in review tab
- [ ] @pierre/diffs theme matching Evergarden summer palette
- [ ] Pagination — GitHub caps at 100 PRs per request
- [ ] Keyboard navigation — j/k list, Enter open, Escape back
- [ ] AI review sessions — walkthrough, correctness, logic, bullshit detector
