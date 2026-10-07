import { createSignal, For, Show } from "solid-js"
import { useActiveReviews } from "../activeReviews"
import { useRepoReviewHistory, usePrefetch, resolved } from "../queries"
import { ReviewRow, activeRow, historyRow, type Row } from "./ReviewRow"
import { ChevronLeftIcon, ChevronRightIcon } from "./icons"

type Props = {
  owner: string
  repo: string
}

export function RepoHistory(props: Props) {
  const { active } = useActiveReviews()
  const [page, setPage] = createSignal(0)
  const prefetch = usePrefetch()

  const history = useRepoReviewHistory(
    () => props.owner,
    () => props.repo,
    page,
  )
  const historyData = resolved(history)

  const activeHere = () =>
    active().filter((r) => r.owner === props.owner && r.repo === props.repo)

  const rows = (): Row[] => {
    const stored = (historyData()?.sessions ?? []).map(historyRow)
    return page() === 0 ? [...activeHere().map(activeRow), ...stored] : stored
  }

  return (
    <aside class="dashboard-main">
      <div class="dashboard-head">
        <h1>agent reviews</h1>
        <Show when={activeHere().length > 0}>
          <span class="badge running">{activeHere().length} in progress</span>
        </Show>
      </div>

      <div class="dashboard-scroll">
        <Show when={history.error}>
          <div class="empty error">{String(history.error)}</div>
        </Show>
        <Show when={rows().length > 0} fallback={<div class="empty">{history.isPending || history.isFetching ? "loading…" : "no reviews for this repo yet"}</div>}>
          <ul class="review-rows" classList={{ "is-stale": history.isFetching }}>
            <For each={rows()}>{(row) => <ReviewRow row={row} compact />}</For>
          </ul>
        </Show>
      </div>

      <div class="pager">
        <button
          class="topbar-btn icon-btn"
          title="previous page"
          aria-label="previous page"
          disabled={page() === 0}
          onClick={() => setPage((n) => n - 1)}
        >
          <ChevronLeftIcon />
        </button>
        <span class="muted">page {page() + 1}</span>
        <button
          class="topbar-btn icon-btn"
          title="next page"
          aria-label="next page"
          disabled={!historyData()?.hasMore}
          onMouseEnter={() => historyData()?.hasMore && prefetch.repoHistory(props.owner, props.repo, page() + 1)}
          onClick={() => setPage((n) => n + 1)}
        >
          <ChevronRightIcon />
        </button>
      </div>
    </aside>
  )
}
