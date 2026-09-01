import { createEffect, createSignal, For, Show } from "solid-js"
import { usePRDetail, usePrefetchDiff } from "../queries"
import { createReview } from "../review"
import { DiffView } from "./diff/DiffView"
import { Markdown } from "./Markdown"
import { ReviewPanel } from "./ReviewPanel"
import { ReviewPage } from "./ReviewPage"
import type { Tab } from "../types"

type Props = {
  owner: string
  repo: string
  prNumber: number
  tab: Tab
  onTabChange: (tab: Tab) => void
  onBack: () => void
}

const tabs: Tab[] = ["description", "files", "review"]

export function PRDetail(props: Props) {
  const detail = usePRDetail(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const review = createReview(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const [filesMounted, setFilesMounted] = createSignal(false)
  const prefetchDiff = usePrefetchDiff()

  createEffect(() => {
    if (props.tab === "files") setFilesMounted(true)
  })

  const prefetchFiles = () => {
    prefetchDiff(props.owner, props.repo, props.prNumber)
  }

  return (
    <div class="pr-detail">
      <div class="pr-tabs">
        <For each={tabs}>
          {(tab) => (
            <button
              class={`pr-tab ${props.tab === tab ? "active" : ""}`}
              onClick={() => props.onTabChange(tab)}
              onMouseEnter={tab === "files" ? prefetchFiles : undefined}
            >
              {tab}
              <Show when={tab === "review" && review.concerns().length > 0}>
                <span class="pr-tab-badge">{review.concerns().length}</span>
              </Show>
            </button>
          )}
        </For>
      </div>

      <div class="pr-tab-content">
        <Show when={props.tab === "description"}>
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

        <Show when={filesMounted()}>
          <div class={`diff-tab-panel ${props.tab === "files" ? "" : "hidden"}`}>
            <div class="review-layout">
              <div class="review-diff">
                <DiffView owner={props.owner} repo={props.repo} prNumber={props.prNumber} />
              </div>
              <ReviewPanel state={review} onOpenReview={() => props.onTabChange("review")} />
            </div>
          </div>
        </Show>

        <Show when={props.tab === "review"}>
          <ReviewPage state={review} />
        </Show>
      </div>
    </div>
  )
}
