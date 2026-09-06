# Session Context — pick up here

## State

Clean tree at `c535c79`. Three commits this session: `b914d5a refactors`,
`0ea73b3 onboarding`, `c535c79 responsive`.

`make vet` and `make test` are green. Nothing below has been seen in a browser.

## What changed

**1. Un-pick (`b914d5a`).** Clicking an already-picked line now removes the
pick. `onSelectedLinesChange` fires with `null` and no file id, so `DiffView`
tracks `selectedFile` itself and calls `onUnpickLine`. The old
`if (!selection) return` guard made picks permanent until reload.

Answered the open question from the last handover, from `@pierre/diffs` source
rather than a browser — see architecture.md. Short version: single click does
pick, but only in the line-number gutter.

Also: severity boxes → coloured pipes; `api.ts` now surfaces the server's error
body instead of `${status} ${path}`; `prPath()` replaced eight copies of the
same handler preamble; dead code removed.

**2. Onboarding (`0ea73b3`).** One binary, four subcommands — `setup`,
`doctor`, `serve`, `migrate`. `cmd/server` and `cmd/migrate` are gone.

- TOML config at `~/.config/pr-review/config.toml`, `PR_REVIEW_CONFIG`
  overrides. **All user data is under that one directory** — db, worktrees,
  token included.
- `internal/ghauth` — device flow, then token file, then `gh` as fallback.
- `internal/preflight` — one `[]Check` list, consumed by both `doctor` and
  `serve`. `serve` refuses to start on a required failure; it still never
  migrates implicitly.
- `server.New` takes `*config.Config`. `tmux.NvimWindow` → `EditorWindow`,
  editor and window cap now configurable.

**3. Responsive (`c535c79`).** One `@media (max-width: 720px)` block at the end
of `style.css`. Desktop untouched. Files tab stacks panel-above-diff with a
collapse toggle. `--topbar-h` is no longer in any `calc()`.

Architecture.md was cut from ~455 to 375 lines.

## Unverified — needs eyes, not a compiler

- **The device flow has never run.** No OAuth client ID to test with. Request
  and poll shapes follow GitHub's docs; every error branch is handled; none of
  it has touched the live endpoint. The `gh` fallback is what has actually been
  exercised.
- **Mobile has never been on a phone.** Two specific doubts: touch targets were
  not scaled up, and line picking needs a tap in a ~20px gutter. If that is
  unusable the fix is a wider tap zone, not CSS.
- Un-pick and the pipe styling — both typecheck-only.
- Carried over, all still true: every theme visually (especially the two light
  ones); whether `System` replaces or appends to an agent's `.md` prompt (if it
  replaces, a non-empty extra prompt guts the instructions and the agent stops
  calling `report`); whether changing an agent's model changes what runs.

## Open question

**Did the database get moved?** The default is now
`~/.config/pr-review/pr-review.db`. Real review history is in `./pr-review.db`
at the repo root. If setup ran with defaults and the file was not copied, the
history is not gone — it is just not being read.

## Load-bearing, easy to break

New:

- **A `null` from `onSelectedLinesChange` is the un-pick gesture.** Guarding it
  away silently breaks the feature.
- **Line selection needs the line-number gutter** (`requireNumberColumn: true`).
  Clicking code text does nothing.
- **`setSelectedLines(..., { notify: false })`** is what stops a side-panel
  click registering as a pick.
- **`goose.CollectMigrations` errors with `ErrNoMigrationFiles` when nothing is
  pending.** Zero pending is not an error; `PendingMigrations` translates it.
- **`setup` must `MkdirAll` the db parent and data dir.** SQLite error 14 was
  exactly this, and it only reproduces on a path that does not exist yet — so
  test onboarding against a fresh directory, never a scratch one you made first.
- **Preflight is one list, two callers.** Add checks there, not in `serve`.
- **Mobile rules stay inside the media query.** `.collapsed` deliberately does
  nothing above 720px so a phone collapse cannot survive a resize.

Carried forward, all still true:

- **Never send `PromptRequest.Format`.** Writes sessions that can't be read back.
- **`Agent` must be passed on the `Prompt` call**, not just `CreateSession`.
- **`.opencode/agents/` and `tools/` load at `opencode serve` startup.**
- **`Publish` must fan out to `global` unconditionally.**
- **`Subscription.Take()` returns a slice.**
- **`Review.clone()` deep-copies `Agent.Stages`.**
- **`make redo` for schema edits, not `make reset`.**
- **A completed `report` call defines success**, not the absence of an error.
- **The `agents` table is an override table.** Empty means "use the default".
- **A new theme needs registering in three places** — CSS block,
  `web/src/theme.ts`, `internal/server/theme.go`.
- **`git worktree prune` must run before every `add`.**
- **A worktree is only reused if `rev-parse HEAD` matches the expected sha.**
- **tmux is the authority on session liveness, not the db.**
- **Removed lines degrade to a file-level pick.**
- **tmux commands are argv, never a shell string.** Session names are sanitised;
  targets use `=name`.
- **`navigator.clipboard` is undefined on http to a non-localhost host.** Use
  `clipboard.copyText`. Every call site now does.

## Next up

- **`skipped` files are still silent.** The API returns them, the client throws
  them away. Pick four, get three windows, no explanation.
- **Capabilities endpoint.** The UI cannot tell that tmux is disabled or absent,
  so it offers the button and then fails. Needs `/api/capabilities` plus a
  frontend that respects it.
- Worktree reaping — see `todo.txt` for the two constraints that make it
  non-trivial.
- Bundle `.opencode/agents` and `tools` into the binary. `opencode.project_dir`
  exists only because those files must be on disk; but the session deep link is
  base64 of that same dir, so both have to move together.
- `serve` checks a token exists, never that it works — an expired one surfaces
  as a 401 on first use.
- No systemd unit; nothing survives a reboot.
- `Toast.tsx` is still unused.
- `POST /session?directory=<checkout>`, then restore `read` for `pr-reviewer`.
- The tmux section of architecture.md is the longest left; trim it if you touch
  it.

## Conventions

- `make vet` **and** `make test`. Never bare `tsc --noEmit` — the root tsconfig
  is solution-style and passes vacuously. `tsc -b` is the real check.
- **Never trigger a review to test something.** Costs real tokens. Ask.
- **Never run the dev server or `serve` to check a change.** Ask the user to.
- **No tests unless explicitly asked.**
- **Use the edit tool, not `python3`/`sed`/heredocs.**
- **No comments in code.** Reasoning goes in `docs/architecture.md`, tasks in
  `todo.txt`. Config-file comments are for the user and are fine.
- **Docs are terse.** Assertions and consequences, not essays.
- **Look before reaching for a glyph.** Icons live in `web/src/components/icons.tsx`.
- **Signals holding arrays of objects break `<For>`.** Use `createStore`.
- No prettier. Keep `roadmap.txt` and `todo.txt` terse.
