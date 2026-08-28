import { For, Show } from "solid-js"
import { usePRs } from "../queries"
import type { PR, Repo } from "../types"

type Props = {
  repo: Repo
  onSelect: (pr: PR) => void
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const h = Math.floor(diff / 36e5)
  if (h < 24) return `${h}h ago`
  const d = Math.floor(h / 24)
  if (d < 30) return `${d}d ago`
  return `${Math.floor(d / 30)}mo ago`
}

export function PRList(props: Props) {
  const prs = usePRs(
    () => props.repo.Owner,
    () => props.repo.Name,
  )

  return (
    <div class="pr-list-page">
      <div class="pr-list-meta">
        <Show when={prs.data}>
          <span>{prs.data!.length} open pull requests</span>
        </Show>
        <Show when={prs.isLoading}>
          <span class="muted">loading...</span>
        </Show>
      </div>
      <ul class="pr-list">
        <For each={prs.data}>
          {(pr) => (
            <li class="pr-row" onClick={() => props.onSelect(pr)}>
              <div class="pr-row-main">
                <span class="pr-row-title">
                  <Show when={pr.Draft}>
                    <span class="badge draft">draft</span>
                  </Show>
                  {pr.Title}
                </span>
                <div class="pr-row-meta">
                  <span class="pr-number">#{pr.Number}</span>
                  <span>{pr.Author}</span>
                  <span class="muted">{timeAgo(pr.UpdatedAt)}</span>
                </div>
              </div>
              <div class="pr-row-arrow">→</div>
            </li>
          )}
        </For>
      </ul>
    </div>
  )
}
