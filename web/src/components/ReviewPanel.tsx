import { createSignal, For, onCleanup, Show } from "solid-js"
import { api } from "../api"
import type { Concern, Review, ReviewStage } from "../types"

type Props = {
  owner: string
  repo: string
  prNumber: number
}

const stageLabels: Record<string, string> = {
  worktree: "checking out",
  session: "starting agent",
  prompt: "reviewing",
  parse: "reading result",
  store: "saving",
}

export function ReviewPanel(props: Props) {
  const [review, setReview] = createSignal<Review | null>(null)
  const [starting, setStarting] = createSignal(false)
  const [error, setError] = createSignal("")
  let stop: (() => void) | undefined

  onCleanup(() => stop?.())

  const start = async () => {
    setStarting(true)
    setError("")
    try {
      const started = await api.review.start(props.owner, props.repo, props.prNumber)
      setReview(started)
      stop?.()
      stop = api.review.stream(started.id, setReview)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setStarting(false)
    }
  }

  const running = () => {
    const r = review()
    return r?.status === "pending" || r?.status === "running"
  }

  const activeStage = () => review()?.stages.find((s) => s.status === "running")

  return (
    <aside class="review-panel">
      <div class="review-panel-head">
        <span class="review-panel-title">agent review</span>
        <button class="review-run" onClick={start} disabled={starting() || running()}>
          {running() ? "reviewing..." : review() ? "re-run" : "run"}
        </button>
      </div>

      <Show when={error()}>
        <div class="review-error">{error()}</div>
      </Show>

      <Show when={review()} keyed>
        {(r) => (
          <>
            <Show when={running()}>
              <div class="review-stages">
                <For each={r.stages}>{(stage) => <StageRow stage={stage} />}</For>
              </div>
              <Show when={activeStage()?.name === "prompt"}>
                <div class="review-hint">agent is reading the diff</div>
              </Show>
            </Show>

            <Show when={r.status === "error"}>
              <div class="review-error">{r.error}</div>
            </Show>

            <Show when={r.status === "done"}>
              <Show when={r.summary}>
                <p class="review-summary">{r.summary}</p>
              </Show>
              <Show
                when={r.concerns.length > 0}
                fallback={<div class="review-clear">no concerns</div>}
              >
                <div class="review-concerns">
                  <For each={r.concerns}>{(concern) => <ConcernCard concern={concern} />}</For>
                </div>
              </Show>
            </Show>
          </>
        )}
      </Show>

      <Show when={!review() && !error()}>
        <p class="review-idle">run a single-pass agent review of this PR</p>
      </Show>
    </aside>
  )
}

function StageRow(props: { stage: ReviewStage }) {
  return (
    <div class={`review-stage ${props.stage.status}`}>
      <span class="review-stage-dot" />
      <span class="review-stage-name">{stageLabels[props.stage.name] ?? props.stage.name}</span>
      <Show when={props.stage.status === "done" && props.stage.durationMs > 0}>
        <span class="review-stage-time">{formatMs(props.stage.durationMs)}</span>
      </Show>
    </div>
  )
}

function ConcernCard(props: { concern: Concern }) {
  return (
    <div class={`review-concern sev-${props.concern.severity}`}>
      <div class="review-concern-head">
        <span class="review-concern-title">{props.concern.title}</span>
        <span class="review-concern-sev">{props.concern.severity}</span>
      </div>
      <div class="review-concern-loc">
        {props.concern.file}
        <Show when={props.concern.line}>:{props.concern.line}</Show>
      </div>
      <p class="review-concern-body">{props.concern.body}</p>
    </div>
  )
}

function formatMs(ms: number) {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}
