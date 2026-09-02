import { For, Show } from "solid-js"
import { useNavigate, useSearch } from "@tanstack/solid-router"
import { useActiveReviews } from "../activeReviews"
import { useReviewHistory, usePrefetch, resolved } from "../queries"
import { ReviewRow, activeRow, historyRow, type Row } from "../components/ReviewRow"

const skeletonRows = Array.from({ length: 8 })

export function DashboardPage() {
  const search = useSearch({ from: "/" })
  const navigate = useNavigate()
  const { active } = useActiveReviews()

  const page = () => search().page
  const history = useReviewHistory(page)
  const historyData = resolved(history)
  const prefetch = usePrefetch()

  const rows = (): Row[] => {
    const stored = (historyData()?.sessions ?? []).map(historyRow)
    return page() === 0 ? [...active().map(activeRow), ...stored] : stored
  }

  const goto = (delta: number) =>
    navigate({ to: "/", search: { page: Math.max(0, page() + delta) } })

  return (
    <div class="dashboard">
      <div class="dashboard-head">
        <h1>history</h1>
        <Show when={active().length > 0}>
          <span class="badge running">{active().length} in progress</span>
        </Show>
      </div>

      <Show when={history.error}>
        <div class="empty error">{String(history.error)}</div>
      </Show>

      <Show
        when={rows().length > 0}
        fallback={
          <Show when={history.isPending} fallback={<div class="empty">no reviews yet</div>}>
            <ul class="review-rows">
              <For each={skeletonRows}>
                {() => (
                  <li class="review-row is-skeleton">
                    <span class="skeleton skeleton-title" />
                    <span class="skeleton skeleton-meta" />
                  </li>
                )}
              </For>
            </ul>
          </Show>
        }
      >
        <ul class="review-rows" classList={{ "is-stale": history.isFetching }}>
          <For each={rows()}>{(row) => <ReviewRow row={row} showRepo />}</For>
        </ul>
      </Show>

      <div class="pager">
        <button
          class="topbar-btn"
          disabled={page() === 0}
          onMouseEnter={() => page() > 0 && prefetch.historyPage(page() - 1)}
          onClick={() => goto(-1)}
        >
          prev
        </button>
        <span class="muted">page {page() + 1}</span>
        <button
          class="topbar-btn"
          disabled={!historyData()?.hasMore}
          onMouseEnter={() => historyData()?.hasMore && prefetch.historyPage(page() + 1)}
          onClick={() => goto(1)}
        >
          next
        </button>
      </div>
    </div>
  )
}
