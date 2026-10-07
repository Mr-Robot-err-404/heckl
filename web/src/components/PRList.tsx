import { For, Show } from "solid-js"
import { usePRs, resolved } from "../queries"
import { RepoHistory } from "./RepoHistory"
import { useActiveReviews } from "../activeReviews"
import { PRRow } from "./PRRow"

type Props = {
  owner: string
  repo: string
}

const skeletonRows = [
  { titleWidth: "68%", reviewers: Array.from({ length: 3 }) },
  { titleWidth: "52%", reviewers: Array.from({ length: 2 }) },
  { titleWidth: "76%", reviewers: Array.from({ length: 4 }) },
  { titleWidth: "61%", reviewers: Array.from({ length: 1 }) },
  { titleWidth: "44%", reviewers: Array.from({ length: 2 }) },
  { titleWidth: "71%", reviewers: Array.from({ length: 3 }) },
]

export function PRList(props: Props) {
  const prs = usePRs(() => props.owner, () => props.repo)
  const rows = resolved(prs)
  const { active } = useActiveReviews()

  const isReviewing = (number: number) =>
    active().some(
      (r) => r.owner === props.owner && r.repo === props.repo && r.prNumber === number,
    )

  return (
    <div class="dashboard">
      <div class="dashboard-cols">
        <section class="recent-pr-panel">
          <div class="dashboard-head">
            <h1>{props.owner}/{props.repo}</h1>
          </div>
          <div class="dashboard-scroll">
            <Show when={prs.error}>
              <div class="empty error">{String(prs.error)}</div>
            </Show>
            <Show when={prs.isPending && !rows()}>
              <ul class="pr-list">
                <For each={skeletonRows}>
                  {(row) => (
                    <li class="pr-row is-skeleton">
                      <div class="pr-row-main">
                        <span class="skeleton skeleton-title" style={{ width: row.titleWidth }} />
                        <div class="pr-row-meta">
                          <span class="skeleton skeleton-number" />
                          <span class="skeleton skeleton-author" />
                        </div>
                      </div>
                      <div class="pr-status skeleton-reviewers">
                        <For each={row.reviewers}>
                          {() => <span class="skeleton skeleton-avatar" />}
                        </For>
                      </div>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
            <Show when={rows()}>
              {(list) => (
                <Show when={list().length > 0} fallback={<div class="empty">no pull requests</div>}>
                  <ul class="pr-list" classList={{ "is-stale": prs.isFetching }}>
                    <For each={list()}>
                      {(pr) => <PRRow pr={pr} reviewing={isReviewing(pr.Number)} />}
                    </For>
                  </ul>
                </Show>
              )}
            </Show>
          </div>
        </section>
        <RepoHistory owner={props.owner} repo={props.repo} />
      </div>
    </div>
  )
}
