import { For, Show } from "solid-js"
import { usePRs } from "../queries"
import type { PR, Repo } from "../types"

type Props = {
  repo: Repo
  selected: PR | null
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
    <div class="pr-list">
      <div class="pr-list-header">
        <span>{props.repo.Owner}/{props.repo.Name}</span>
        <Show when={prs.data}>
          <span class="pr-count">{prs.data!.length} open</span>
        </Show>
      </div>
      <ul>
        <For each={prs.data}>
          {(pr) => (
            <li
              class={`pr-item ${props.selected?.ID === pr.ID ? "active" : ""}`}
              onClick={() => props.onSelect(pr)}
            >
              <div class="pr-title">
                <Show when={pr.Draft}>
                  <span class="badge draft">draft</span>
                </Show>
                {pr.Title}
              </div>
              <div class="pr-meta">
                <span>#{pr.Number}</span>
                <span>{pr.Author}</span>
                <span>{timeAgo(pr.UpdatedAt)}</span>
              </div>
            </li>
          )}
        </For>
      </ul>
    </div>
  )
}
