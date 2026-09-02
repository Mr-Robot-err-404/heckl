import { For, Show } from "solid-js"
import { useNavigate } from "@tanstack/solid-router"
import { usePRs } from "../queries"
import { RepoHistory } from "./RepoHistory"
import { PRStatus } from "./PRStatus"
import { useActiveReviews } from "../activeReviews"

type Props = {
  owner: string
  repo: string
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
  const navigate = useNavigate()
  const prs = usePRs(() => props.owner, () => props.repo)
  const { active } = useActiveReviews()

  const isReviewing = (number: number) =>
    active().some(
      (r) => r.owner === props.owner && r.repo === props.repo && r.prNumber === number,
    )

  return (
    <div class="repo-page">
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
              <li
                class="pr-row"
                onClick={() => navigate({
                  to: "/$owner/$repo/$pr",
                  params: { owner: props.owner, repo: props.repo, pr: String(pr.Number) },
                  search: { tab: "description" },
                })}
              >
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
                <PRStatus pr={pr} reviewing={isReviewing(pr.Number)} />
                <div class="pr-row-arrow">→</div>
              </li>
            )}
          </For>
        </ul>
      </div>
      <RepoHistory owner={props.owner} repo={props.repo} />
    </div>
  )
}
