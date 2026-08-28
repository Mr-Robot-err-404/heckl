import { createSignal, Show } from "solid-js"
import { usePRDetail } from "../queries"
import { DiffView } from "./DiffView"
import { Markdown } from "./Markdown"

type Tab = "description" | "review"

type Props = {
  owner: string
  repo: string
  prNumber: number
  onBack: () => void
}

export function PRDetail(props: Props) {
  const detail = usePRDetail(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const [tab, setTab] = createSignal<Tab>("description")

  return (
    <div class="pr-detail">
      <div class="pr-tabs">
        <button
          class={`pr-tab ${tab() === "description" ? "active" : ""}`}
          onClick={() => setTab("description")}
        >
          description
        </button>
        <button
          class={`pr-tab ${tab() === "review" ? "active" : ""}`}
          onClick={() => setTab("review")}
        >
          review
        </button>
      </div>
      <div class="pr-tab-content">
        <Show when={tab() === "description"}>
          <div class="pr-description">
            <Show when={detail.data} keyed>
              {(d) => (
                <Show when={d.pr.Body} fallback={<span class="muted">no description</span>}>
                  <Markdown content={d.pr.Body} />
                </Show>
              )}
            </Show>
            <Show when={detail.isLoading}>
              <span class="muted">loading...</span>
            </Show>
          </div>
        </Show>
        <Show when={tab() === "review"}>
          <DiffView owner={props.owner} repo={props.repo} prNumber={props.prNumber} />
        </Show>
      </div>
    </div>
  )
}
