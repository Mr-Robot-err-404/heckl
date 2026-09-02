import { createSignal, For, Show } from "solid-js"
import { useActiveReviews } from "../activeReviews"
import { historyPageSize, useRepoReviewHistory, usePrefetch, resolved } from "../queries"
import { ReviewRow, activeRow, historyRow, type Row } from "./ReviewRow"

type Props = {
  owner: string
  repo: string
}

export function RepoHistory(props: Props) {
  const { active } = useActiveReviews()
  const [limit, setLimit] = createSignal(historyPageSize)
  const prefetch = usePrefetch()

  const history = useRepoReviewHistory(
    () => props.owner,
    () => props.repo,
    limit,
  )
  const historyData = resolved(history)

  const activeHere = () =>
    active().filter((r) => r.owner === props.owner && r.repo === props.repo)

  const rows = (): Row[] => [
    ...activeHere().map(activeRow),
    ...(historyData()?.sessions ?? []).map(historyRow),
  ]

  return (
    <aside class="repo-history">
      <div class="repo-history-head">
        <h2>reviews</h2>
        <Show when={activeHere().length > 0}>
          <span class="badge running">{activeHere().length} running</span>
        </Show>
      </div>

      <Show when={history.error}>
        <div class="repo-history-empty error">{String(history.error)}</div>
      </Show>

      <Show
        when={rows().length > 0}
        fallback={
          <div class="repo-history-empty muted">
            {history.isPending ? "loading…" : "no reviews for this repo yet"}
          </div>
        }
      >
        <ul class="review-rows compact">
          <For each={rows()}>{(row) => <ReviewRow row={row} />}</For>
        </ul>
      </Show>

      <Show when={historyData()?.hasMore}>
        <button
          class="topbar-btn load-more"
          disabled={history.isFetching}
          onMouseEnter={() =>
            prefetch.repoHistory(props.owner, props.repo, limit() + historyPageSize)
          }
          onClick={() => setLimit((n) => n + historyPageSize)}
        >
          {history.isFetching ? "loading…" : "load more"}
        </button>
      </Show>
    </aside>
  )
}
