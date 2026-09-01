import { For, Show } from "solid-js"
import { fileName, formatMs, type ReviewState } from "../review"
import type { RankedConcern } from "../types"

type Props = {
  state: ReviewState
  activeRank?: number
  onFocusConcern: (concern: RankedConcern) => void
  onOpenReview: () => void
}

const stageLabels: Record<string, string> = {
  fetch: "fetching pr",
  checkout: "checking out",
  session: "starting agent",
  prompt: "reviewing",
  parse: "reading result",
  store: "saving",
}

export function ReviewPanel(props: Props) {
  const s = () => props.state
  const activeStage = () => s().review()?.stages.find((stage) => stage.status === "running")

  return (
    <aside class="review-panel">
      <div class="review-panel-head">
        <span class="review-panel-title">agent review</span>
        <button class="review-run" onClick={s().start} disabled={!s().canRun()}>
          {!s().synced() ? "connecting..." : s().busy() ? "reviewing..." : s().review() ? "re-run" : "run"}
        </button>
      </div>

      <Show when={s().error()}>
        <div class="review-error">{s().error()}</div>
      </Show>

      <Show when={s().synced() && !s().connected()}>
        <div class="review-stale">connection lost — reconnecting</div>
      </Show>

      <Show when={!s().synced()}>
        <p class="review-idle">syncing with server...</p>
      </Show>

      <Show when={s().busy()}>
        <div class="review-progress">
          <span class="review-stage-dot" />
          <span class="review-stage-name">
            {stageLabels[activeStage()?.name ?? ""] ?? "starting"}
          </span>
          <span class="review-stage-time">{formatMs(s().elapsed())}</span>
        </div>
      </Show>

      <Show when={s().synced() && !s().busy() && !s().review()}>
        <p class="review-idle">no review yet</p>
      </Show>

      <Show when={s().review()?.status === "error"}>
        <div class="review-error">{s().review()?.error}</div>
      </Show>

      <Show when={s().review()?.status === "done"}>
        <Show
          when={s().concerns().length > 0}
          fallback={<div class="review-clear">no concerns</div>}
        >
          <ul class="concern-index">
            <For each={s().concerns()}>
              {(concern) => (
                <ConcernRow
                  concern={concern}
                  active={props.activeRank === concern.rank}
                  onFocus={() => props.onFocusConcern(concern)}
                />
              )}
            </For>
          </ul>
        </Show>
        <button class="review-open" onClick={props.onOpenReview}>
          full review →
        </button>
      </Show>
    </aside>
  )
}

function ConcernRow(props: {
  concern: RankedConcern
  active: boolean
  onFocus: () => void
}) {
  const locatable = () => props.concern.line != null && props.concern.side != null

  return (
    <li
      class={`concern-row sev-${props.concern.severity} ${props.active ? "active" : ""} ${locatable() ? "locatable" : ""}`}
      onClick={() => locatable() && props.onFocus()}
    >
      <span class="concern-rank">{props.concern.rank}</span>
      <span class="concern-row-title">{props.concern.title}</span>
      <span class="concern-row-file">
        {fileName(props.concern.file)}
        <Show when={props.concern.line} fallback={<span class="concern-unpinned"> · file</span>}>
          :{props.concern.line}
        </Show>
      </span>
    </li>
  )
}
