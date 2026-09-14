import { For, Show } from "solid-js"
import { useNavigate } from "@tanstack/solid-router"
import { relativeTime } from "../review"
import { usePrefetch } from "../queries"
import type { Review, ReviewHistoryRow } from "../types"

export type Severity = "critical" | "warning" | "low"

export type Concerns = {
  total: number
  counts: Record<Severity, number>
}

export type Row = {
  key: string
  owner: string
  repo: string
  prNumber: number
  status: "running" | "done" | "error"
  label: string
  title: string
  concerns: Concerns | null
  note: string
  date: string
}

const severities: Severity[] = ["critical", "warning", "low"]

export function activeRow(review: Review): Row {
  const stage = review.stages.find((s) => s.status === "running")
  return {
    key: `active-${review.id}`,
    owner: review.owner,
    repo: review.repo,
    prNumber: review.prNumber,
    status: "running",
    label: "running",
    title: review.summary || `reviewing #${review.prNumber}`,
    concerns: null,
    note: stage ? `${stage.name}…` : "starting…",
    date: "",
  }
}

export function historyRow(session: ReviewHistoryRow): Row {
  const failed = session.status === "error"
  return {
    key: `session-${session.id}`,
    owner: session.owner,
    repo: session.repo,
    prNumber: session.prNumber,
    status: failed ? "error" : "done",
    label: failed ? "failed" : "done",
    title: failed ? `#${session.prNumber} review failed` : session.summary,
    concerns: failed
      ? null
      : {
          total: session.concernCount,
          counts: {
            critical: session.highCount,
            warning: session.mediumCount,
            low: session.lowCount,
          },
        },
    note: failed ? (session.error ?? "") : "",
    date: relativeTime(session.createdAt),
  }
}

export function ReviewRow(props: { row: Row; showRepo?: boolean; showDetails?: boolean }) {
  const navigate = useNavigate()
  const prefetch = usePrefetch()

  const warm = () =>
    prefetch.pr(props.row.owner, props.row.repo, props.row.prNumber)

  const open = () =>
    navigate({
      to: "/$owner/$repo/$pr",
      params: {
        owner: props.row.owner,
        repo: props.row.repo,
        pr: String(props.row.prNumber),
      },
      search: { tab: "files" },
    })

  return (
    <li
      class={`review-row is-${props.row.status}`}
      onMouseEnter={warm}
      onClick={open}
    >
      <div class="review-row-top">
        <span class={`pill pill-${props.row.status}`}>{props.row.label}</span>
        <Show when={props.showRepo}>
          <span class="review-row-repo" title={`${props.row.owner}/${props.row.repo}`}>
            {props.row.owner}/{props.row.repo}
          </span>
        </Show>
        <span class="review-row-pr muted">#{props.row.prNumber}</span>
        <span class="review-row-title">{props.row.title}</span>
        <span class="review-row-date muted">{props.row.date}</span>
      </div>

      <Show when={props.showDetails && (!!props.row.concerns || !!props.row.note)}>
        <div class="review-row-sub">
          <Show when={props.row.concerns} keyed>
            {(concerns) => (
              <Show
                when={concerns.total > 0}
                fallback={<span class="sev-none">no concerns</span>}
              >
                <span class="sev-list">
                  <For each={severities}>
                    {(severity, i) => (
                      <>
                        <Show when={i() > 0}>
                          <span class="sev-sep">|</span>
                        </Show>
                        <span
                          class={`sev sev-${severity}`}
                          classList={{ "is-zero": concerns.counts[severity] === 0 }}
                        >
                          {concerns.counts[severity]} {severity}
                        </span>
                      </>
                    )}
                  </For>
                </span>
              </Show>
            )}
          </Show>
          <Show when={props.row.note}>
            <span class="review-row-note">{props.row.note}</span>
          </Show>
        </div>
      </Show>
    </li>
  )
}
