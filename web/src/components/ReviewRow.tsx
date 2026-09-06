import { createSignal, For, Show } from "solid-js"
import { useNavigate } from "@tanstack/solid-router"
import { formatMs, relativeTime } from "../review"
import { usePrefetch } from "../queries"
import type { Review, ReviewHistoryRow } from "../types"
import { ChevronIcon } from "./icons"

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
  meta: string
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
    meta: "",
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
    meta: [relativeTime(session.createdAt), session.durationMs ? formatMs(session.durationMs) : ""]
      .filter(Boolean)
      .join(" · "),
  }
}

export function ReviewRow(props: { row: Row; showRepo?: boolean; collapsible?: boolean }) {
  const navigate = useNavigate()
  const prefetch = usePrefetch()
  const [expanded, setExpanded] = createSignal(false)

  const hasSub = () => !!props.row.concerns || !!props.row.note
  const showSub = () => hasSub() && (!props.collapsible || expanded())

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
      classList={{ expanded: props.collapsible && expanded() }}
      onMouseEnter={warm}
      onClick={open}
    >
      <div class="review-row-top">
        <Show when={props.collapsible && hasSub()}>
          <button
            class="review-row-caret"
            classList={{ collapsed: !expanded() }}
            aria-expanded={expanded()}
            aria-label={expanded() ? "collapse review detail" : "expand review detail"}
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded())
            }}
          >
            <ChevronIcon />
          </button>
        </Show>
        <span class={`pill pill-${props.row.status}`}>{props.row.label}</span>
        <span class="review-row-pr">
          <Show when={props.showRepo}>
            {props.row.owner}/{props.row.repo}{" "}
          </Show>
          <span class="muted">#{props.row.prNumber}</span>
        </span>
        <span class="review-row-title">{props.row.title}</span>
        <span class="review-row-meta muted">{props.row.meta}</span>
      </div>

      <Show when={showSub()}>
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
