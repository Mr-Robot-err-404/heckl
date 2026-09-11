import { createMemo, createSignal, For, Show } from "solid-js"
import { useNavigate } from "@tanstack/solid-router"
import { resolved, useCleanupTmux, usePrefetch, useTmuxSessions } from "../queries"
import { relativeTime } from "../review"
import type { TmuxRef, TmuxRow } from "../types"

const refOf = (row: TmuxRow): TmuxRef => ({
  owner: row.owner,
  repo: row.repo,
  prNumber: row.prNumber,
})

const keyOf = (row: TmuxRow) => `${row.owner}/${row.repo}/${row.prNumber}`

export function TmuxPanel() {
  const sessions = useTmuxSessions()
  const data = resolved(sessions)
  const cleanup = useCleanupTmux()
  const navigate = useNavigate()
  const prefetch = usePrefetch()

  const [selecting, setSelecting] = createSignal(false)
  const [picked, setPicked] = createSignal<string[]>([])

  const rows = () => data() ?? []
  const isPicked = (row: TmuxRow) => picked().includes(keyOf(row))
  const allPicked = createMemo(
    () => rows().length > 0 && rows().every((row) => isPicked(row)),
  )

  const toggle = (row: TmuxRow) => {
    const key = keyOf(row)
    setPicked((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]))
  }

  const toggleAll = () => setPicked(allPicked() ? [] : rows().map(keyOf))

  const stopSelecting = () => {
    setSelecting(false)
    setPicked([])
  }

  const confirmKill = () => {
    const refs = rows().filter(isPicked).map(refOf)
    if (refs.length === 0) return
    cleanup.mutate(refs, { onSuccess: stopSelecting })
  }

  const open = (row: TmuxRow) =>
    navigate({
      to: "/$owner/$repo/$pr",
      params: { owner: row.owner, repo: row.repo, pr: String(row.prNumber) },
      search: { tab: "files" },
    })

  const activate = (row: TmuxRow) => (selecting() ? toggle(row) : open(row))

  return (
    <section class="tmux-panel" classList={{ "is-selecting": selecting() }}>
      <div class="dashboard-head">
        <h1>tmux</h1>
        <Show when={rows().length > 0}>
          <span class="muted">{rows().length}</span>
        </Show>
      </div>

      <Show when={sessions.error}>
        <div class="empty error">{String(sessions.error)}</div>
      </Show>
      <Show when={cleanup.error}>
        <div class="empty error">{String(cleanup.error)}</div>
      </Show>

      <Show
        when={rows().length > 0}
        fallback={
          <Show when={!sessions.isPending}>
            <div class="empty">no tmux sessions</div>
          </Show>
        }
      >
        <ul class="review-rows" classList={{ "is-stale": sessions.isFetching }}>
          <li class="tmux-row is-head">
            <input
              type="checkbox"
              checked={allPicked()}
              tabIndex={selecting() ? 0 : -1}
              aria-label="select every tmux session"
              onChange={toggleAll}
            />
            <div class="tmux-actions">
              <Show
                when={selecting()}
                fallback={
                  <button class="topbar-btn" onClick={() => setSelecting(true)}>
                    kill
                  </button>
                }
              >
                <button class="topbar-btn" disabled={cleanup.isPending} onClick={stopSelecting}>
                  cancel
                </button>
                <button
                  class="topbar-btn"
                  classList={{ danger: picked().length > 0 }}
                  disabled={picked().length === 0 || cleanup.isPending}
                  onClick={confirmKill}
                >
                  {cleanup.isPending ? "killing..." : "kill"}
                </button>
              </Show>
            </div>
          </li>

          <For each={rows()}>
            {(row) => (
              <li
                class="review-row tmux-row"
                classList={{
                  "is-live": row.live,
                  "is-gone": !row.live,
                  "is-picked": selecting() && isPicked(row),
                }}
                onMouseEnter={() => prefetch.pr(row.owner, row.repo, row.prNumber)}
                onClick={() => activate(row)}
              >
                <input
                  type="checkbox"
                  checked={isPicked(row)}
                  tabIndex={selecting() ? 0 : -1}
                  aria-label={`select ${row.session}`}
                  onClick={(e) => e.stopPropagation()}
                  onChange={() => toggle(row)}
                />
                <span class={`pill pill-${row.live ? "live" : "gone"}`}>
                  {row.live ? "live" : "stale"}
                </span>
                <span class="tmux-row-pr" title={row.session}>
                  {row.repo} <span class="muted">#{row.prNumber}</span>
                </span>
                <span class="tmux-age muted">{relativeTime(row.createdAt)}</span>
                <span class="tmux-windows" title={`${row.windows} windows`}>
                  {row.windows}
                </span>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </section>
  )
}
