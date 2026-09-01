import { createEffect, createSignal, For, onCleanup, Show } from "solid-js"
import { api } from "../api"
import type { Concern, Review, ReviewStage } from "../types"

type Props = {
  owner: string
  repo: string
  prNumber: number
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
  const [review, setReview] = createSignal<Review | null>(null)
  const [starting, setStarting] = createSignal(false)
  const [error, setError] = createSignal("")
  const [now, setNow] = createSignal(Date.now())

  createEffect(() => {
    const { owner, repo, prNumber } = props
    setReview(null)
    setStarting(false)
    setError("")
    const close = api.review.stream(owner, repo, prNumber, (next) => {
      if (next) setStarting(false)
      setReview(next)
    })
    onCleanup(close)
  })

  const inFlight = () => {
    const status = review()?.status
    return status === "pending" || status === "running"
  }
  const busy = () => starting() || inFlight()

  createEffect(() => {
    if (!busy()) return
    const timer = setInterval(() => setNow(Date.now()), 200)
    onCleanup(() => clearInterval(timer))
  })

  const elapsed = () => {
    const r = review()
    if (!r) return 0
    const end = r.endedAt ? Date.parse(r.endedAt) : now()
    return end - Date.parse(r.startedAt)
  }

  const start = async () => {
    setError("")
    setStarting(true)
    try {
      setReview(await api.review.start(props.owner, props.repo, props.prNumber))
    } catch (e) {
      setStarting(false)
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <aside class="review-panel">
      <div class="review-panel-head">
        <span class="review-panel-title">agent review</span>
        <button class="review-run" onClick={start} disabled={busy()}>
          {busy() ? "reviewing..." : review() ? "re-run" : "run"}
        </button>
      </div>

      <Show when={error()}>
        <div class="review-error">{error()}</div>
      </Show>

      <Show when={starting() && !review()}>
        <div class="review-stages">
          <div class="review-stage running">
            <span class="review-stage-dot" />
            <span class="review-stage-name">starting</span>
          </div>
        </div>
      </Show>

      <Show when={review()} keyed fallback={<Show when={!starting()}><p class="review-idle">no review yet</p></Show>}>
        {(r) => (
          <>
            <Show when={r.status !== "done"}>
              <div class="review-stages">
                <For each={r.stages}>
                  {(stage) => <StageRow stage={stage} now={now()} />}
                </For>
              </div>
            </Show>

            <Show when={r.status === "error"}>
              <div class="review-error">{r.error}</div>
            </Show>

            <Show when={r.status === "done"}>
              <div class="review-meta">
                {r.stages.length} stages · {formatMs(elapsed())}
              </div>
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
    </aside>
  )
}

function StageRow(props: { stage: ReviewStage; now: number }) {
  const live = () => {
    if (props.stage.status !== "running" || !props.stage.startedAt) return 0
    return props.now - Date.parse(props.stage.startedAt)
  }

  return (
    <div class={`review-stage ${props.stage.status}`}>
      <span class="review-stage-dot" />
      <span class="review-stage-name">{stageLabels[props.stage.name] ?? props.stage.name}</span>
      <Show when={props.stage.detail}>
        <span class="review-stage-detail">{props.stage.detail}</span>
      </Show>
      <Show when={props.stage.status === "done" && props.stage.durationMs > 0}>
        <span class="review-stage-time">{formatMs(props.stage.durationMs)}</span>
      </Show>
      <Show when={live() > 0}>
        <span class="review-stage-time">{formatMs(live())}</span>
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
