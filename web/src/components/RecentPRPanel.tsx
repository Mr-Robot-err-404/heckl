import { For, Show } from "solid-js"
import { useActiveReviews } from "../activeReviews"
import {
  resolved,
  useRecentPRFilter,
  useRecentPRs,
  useRepos,
  useSaveRecentPRFilter,
} from "../queries"
import type { PR, RecentPRFilter } from "../types"
import { FilterIcon } from "./icons"
import { PRRow } from "./PRRow"

const skeletonRows = Array.from({ length: 6 })

export function RecentPRPanel() {
  const recent = useRecentPRs()
  const recentData = resolved(recent)
  const filters = useRecentPRFilter()
  const filterData = resolved(filters)
  const repos = useRepos()
  const repoData = resolved(repos)
  const saveFilter = useSaveRecentPRFilter()
  const { active } = useActiveReviews()

  const rows = () => recentData()?.pullRequests ?? []
  const unavailable = () => recentData()?.unavailable ?? []
  const organizations = () => [...new Set((repoData() ?? []).map((repo) => repo.Owner))].sort()
  const filterCount = () =>
    (filterData()?.organizations.length ?? 0) + (filterData()?.repositories.length ?? 0)
  const isReviewing = (pr: PR) =>
    active().some(
      (review) => review.owner === pr.Owner && review.repo === pr.Repo && review.prNumber === pr.Number,
    )

  const toggle = (kind: keyof RecentPRFilter, value: string) => {
    const current = filterData() ?? { organizations: [], repositories: [] }
    const values = current[kind]
    saveFilter.mutate({
      ...current,
      [kind]: values.includes(value) ? values.filter((item) => item !== value) : [...values, value],
    })
  }

  return (
    <section class="recent-pr-panel">
      <div class="dashboard-head recent-pr-head">
        <h1>recent pull requests</h1>
        <Show when={rows().length > 0}>
          <span class="muted">{rows().length}</span>
        </Show>
        <details class="recent-pr-filter">
          <summary class="topbar-btn icon-btn" title="filter pull requests" aria-label="filter pull requests">
            <FilterIcon />
            <Show when={filterCount() > 0}>
              <span class="filter-count">{filterCount()}</span>
            </Show>
          </summary>
          <div class="recent-pr-filter-menu">
            <div class="recent-pr-filter-title">hide activity from</div>
            <Show when={filters.error || repos.error}>
              <div class="input-error">{String(filters.error ?? repos.error)}</div>
            </Show>
            <For each={organizations()}>
              {(owner) => (
                <div class="recent-pr-filter-group">
                  <label>
                    <input
                      type="checkbox"
                      checked={filterData()?.organizations.includes(owner)}
                      disabled={saveFilter.isPending}
                      onChange={() => toggle("organizations", owner)}
                    />
                    <span>{owner}</span>
                  </label>
                  <For each={(repoData() ?? []).filter((repo) => repo.Owner === owner)}>
                    {(repo) => {
                      const key = `${repo.Owner}/${repo.Name}`
                      return (
                        <label class="recent-pr-filter-repo">
                          <input
                            type="checkbox"
                            checked={filterData()?.repositories.includes(key)}
                            disabled={filterData()?.organizations.includes(owner) || saveFilter.isPending}
                            onChange={() => toggle("repositories", key)}
                          />
                          <span>{repo.Name}</span>
                        </label>
                      )
                    }}
                  </For>
                </div>
              )}
            </For>
            <Show when={(repoData()?.length ?? 0) === 0 && repos.isSuccess}>
              <span class="muted">no followed repositories</span>
            </Show>
          </div>
        </details>
      </div>

      <div class="dashboard-scroll">
        <Show when={recent.error || saveFilter.error}>
          <div class="empty error">{String(recent.error ?? saveFilter.error)}</div>
        </Show>
        <Show when={unavailable().length > 0}>
          <div class="recent-pr-warning" title={unavailable().join(", ")}>
            {unavailable().length} repositories unavailable
          </div>
        </Show>

        <Show
          when={rows().length > 0}
          fallback={
            <Show when={recent.isPending} fallback={<div class="empty recent-pr-empty">no recent pull requests</div>}>
              <ul class="review-rows">
                <For each={skeletonRows}>
                  {() => (
                    <li class="recent-pr-row is-skeleton">
                      <span class="skeleton skeleton-title" />
                      <span class="skeleton skeleton-repo" />
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          }
        >
          <ul class="pr-list" classList={{ "is-stale": recent.isFetching }}>
            <For each={rows()}>
              {(pr) => (
                <PRRow pr={pr} reviewing={isReviewing(pr)} showRepo />
              )}
            </For>
          </ul>
        </Show>
      </div>
    </section>
  )
}
