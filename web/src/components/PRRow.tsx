import { Show } from "solid-js"
import { useNavigate } from "@tanstack/solid-router"
import { usePrefetch } from "../queries"
import type { PR } from "../types"
import { PRStatus } from "./PRStatus"

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const hours = Math.floor(diff / 36e5)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return `${Math.floor(days / 30)}mo ago`
}

export function PRRow(props: { pr: PR; reviewing: boolean; showRepo?: boolean }) {
  const navigate = useNavigate()
  const prefetch = usePrefetch()

  return (
    <li
      class="pr-row"
      onMouseEnter={() => prefetch.pr(props.pr.Owner, props.pr.Repo, props.pr.Number)}
      onClick={() => navigate({
        to: "/$owner/$repo/$pr",
        params: { owner: props.pr.Owner, repo: props.pr.Repo, pr: String(props.pr.Number) },
        search: { tab: "description" },
      })}
    >
      <div class="pr-row-main">
        <span class="pr-row-title">
          <Show when={props.pr.Draft}>
            <span class="badge draft">draft</span>
          </Show>
          {props.pr.Title}
        </span>
        <div class="pr-row-meta">
          <Show when={props.showRepo}>
            <span class="pr-row-repo">{props.pr.Owner}/{props.pr.Repo}</span>
          </Show>
          <span class="pr-number">#{props.pr.Number}</span>
          <span>{props.pr.Author}</span>
          <span class="muted">{timeAgo(props.pr.UpdatedAt)}</span>
        </div>
      </div>
      <PRStatus pr={props.pr} reviewing={props.reviewing} />
    </li>
  )
}
