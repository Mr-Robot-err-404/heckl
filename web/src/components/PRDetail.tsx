import { createEffect, createSignal, For, onCleanup, Show } from "solid-js"
import { createStore, produce } from "solid-js/store"
import { copyText } from "../clipboard"
import { usePRComments, usePRDetail, usePrefetch, resolved } from "../queries"
import { createReview } from "../review"
import { DiffView } from "./diff/DiffView"
import { Markdown } from "./Markdown"
import { SkeletonLines } from "./Skeleton"
import { ReviewPanel } from "./ReviewPanel"
import { ReviewPage } from "./ReviewPage"
import { TmuxModal } from "./TmuxModal"
import { api } from "../api"
import type { ConcernTarget, RankedConcern, ReviewerNote, Tab, TmuxPick } from "../types"

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
  const detailData = resolved(detail)
  const comments = usePRComments(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const threads = resolved(comments)
  const review = createReview(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const [filesMounted, setFilesMounted] = createSignal(false)
  const [focus, setFocus] = createSignal<ConcernTarget | null>(null)
  const prefetch = usePrefetch()

  const [picks, setPicks] = createStore<{ items: TmuxPick[] }>({ items: [] })
  const [tmuxOpen, setTmuxOpen] = createSignal(false)
  const [tmuxBusy, setTmuxBusy] = createSignal(false)
  const [tmuxError, setTmuxError] = createSignal("")
  const [toast, setToast] = createSignal("")

  let toastTimer: ReturnType<typeof setTimeout> | undefined
  const flash = (message: string) => {
    clearTimeout(toastTimer)
    setToast(message)
    toastTimer = setTimeout(() => setToast(""), 1400)
  }
  onCleanup(() => clearTimeout(toastTimer))

  const pickLine = (file: string, line: number) => {
    const i = picks.items.findIndex((p) => p.file === file)
    setPicks(
      produce((s) => {
        if (i === -1) s.items.push({ file, line })
        else s.items[i].line = line
      }),
    )
  }

  const removePick = (file: string) => {
    setPicks(
      produce((s) => {
        const i = s.items.findIndex((p) => p.file === file)
        if (i !== -1) s.items.splice(i, 1)
      }),
    )
  }

  const openTmuxModal = () => {
    setTmuxError("")
    setTmuxOpen(true)
  }

  const confirmTmux = async () => {
    if (picks.items.length === 0) return
    setTmuxBusy(true)
    setTmuxError("")
    try {
      const result = await api.tmux.open(
        props.owner,
        props.repo,
        props.prNumber,
        picks.items.map((p) => ({ ...p })),
      )
      const copied = await copyText(result.attach)
      setPicks("items", [])
      setTmuxOpen(false)
      const skipped = result.skipped?.length ? ` · ${result.skipped.length} skipped` : ""
      flash(copied ? `copied to clipboard${skipped}` : `${result.attach}${skipped}`)
    } catch (e) {
      setTmuxError(String(e))
    } finally {
      setTmuxBusy(false)
    }
  }

  createEffect(() => {
    if (props.tab === "files") setFilesMounted(true)
  })

  let nonce = 0
  const focusLine = (file: string, line: number, side: "additions" | "deletions", rank: number) => {
    nonce += 1
    setFocus({ file, line, side, rank, nonce })
    prefetch.diff(props.owner, props.repo, props.prNumber)
    props.onTabChange("files")
  }

  const focusConcern = (concern: RankedConcern) => {
    if (concern.line == null || concern.side == null) return
    focusLine(concern.file, concern.line, concern.side, concern.rank)
  }

  const focusNote = (note: ReviewerNote) => {
    if (!note.file || note.line == null || note.side == null) return
    focusLine(note.file, note.line, note.side, 0)
  }

  const prefetchFiles = () => {
    prefetch.diff(props.owner, props.repo, props.prNumber)
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

        <button
          class={`tmux-btn ${picks.items.length > 0 ? "picked" : ""}`}
          title="open selected lines in nvim"
          onClick={openTmuxModal}
        >
          tmux
          <Show when={picks.items.length > 0}>
            <span class="tmux-btn-count">{picks.items.length}</span>
          </Show>
        </button>
      </div>

      <div class="pr-tab-content">
        <Show when={props.tab === "description"}>
          <div class="pr-description">
            <Show when={detailData()} keyed>
              {(d) => (
                <Show when={d.pr.Body} fallback={<span class="muted">no description</span>}>
                  <Markdown content={d.pr.Body} />
                </Show>
              )}
            </Show>
            <Show when={detail.isPending}>
              <SkeletonLines />
            </Show>
          </div>
        </Show>

        <Show when={filesMounted()}>
          <div class={`diff-tab-panel ${props.tab === "files" ? "" : "hidden"}`}>
            <div class="review-layout">
              <div class="review-diff">
                <DiffView
                  owner={props.owner}
                  repo={props.repo}
                  prNumber={props.prNumber}
                  focus={focus()}
                  onPickLine={pickLine}
                />
              </div>
              <ReviewPanel
                state={review}
                activeRank={focus()?.rank}
                threads={threads() ?? []}
                threadsPending={comments.isPending}
                onFocusConcern={focusConcern}
                onFocusNote={focusNote}
                onOpenReview={() => props.onTabChange("review")}
              />
            </div>
          </div>
        </Show>

        <Show when={props.tab === "review"}>
          <ReviewPage state={review} onFocusConcern={focusConcern} />
        </Show>
      </div>

      <Show when={tmuxOpen()}>
        <TmuxModal
          picks={picks.items}
          busy={tmuxBusy()}
          error={tmuxError()}
          onRemove={removePick}
          onConfirm={confirmTmux}
          onClose={() => setTmuxOpen(false)}
        />
      </Show>

      <Show when={toast()}>
        <div class="toast">{toast()}</div>
      </Show>
    </div>
  )
}
