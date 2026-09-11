import { createEffect, createSignal, For, Show } from "solid-js"
import { createStore, produce } from "solid-js/store"
import { useQueryClient } from "@tanstack/solid-query"
import {
  usePRComments,
  usePRDetail,
  usePrefetch,
  useTmuxSession,
  resolved,
  tmuxSessionKey,
} from "../queries"
import { createReview } from "../review"
import { DiffView } from "./diff/DiffView"
import { diffSides } from "./diff/loadFiles"
import { Markdown } from "./Markdown"
import { SkeletonLines } from "./Skeleton"
import { ReviewPanel } from "./ReviewPanel"
import { ReviewerComments } from "./ReviewerComments"
import { ReviewPage } from "./ReviewPage"
import { TmuxModal } from "./TmuxModal"
import { useModal } from "../modal"
import { createDiffAnchor, prUrl, type DiffAnchor } from "../github"
import { api, errText } from "../api"
import { dirIcon, fileIcon, iconViewBox, type Icon as FileIcon } from "../fileIcons"
import type {
  ConcernTarget,
  FileTarget,
  RankedConcern,
  ReviewerNote,
  Tab,
  TmuxPick,
} from "../types"

type Props = {
  owner: string
  repo: string
  prNumber: number
  tab: Tab
  onTabChange: (tab: Tab) => void
  onBack: () => void
}

const tabs: Tab[] = ["description", "files", "review"]

const Icon = (props: { of: FileIcon }) => (
  <svg class={`nf nf-${props.of[1]}`} viewBox={iconViewBox} aria-hidden="true">
    <path d={props.of[0]} fill="currentColor" />
  </svg>
)

const dirPart = (path: string) => path.slice(0, path.lastIndexOf("/") + 1)
const basePart = (path: string) => path.slice(path.lastIndexOf("/") + 1)

function groupByDir<T>(files: T[], path: (f: T) => string): { dir: string; files: T[] }[] {
  const groups = new Map<string, T[]>()
  for (const f of files) {
    const dir = dirPart(path(f))
    const existing = groups.get(dir)
    if (existing) existing.push(f)
    else groups.set(dir, [f])
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([dir, files]) => ({ dir, files }))
}

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
  const [fileFocus, setFileFocus] = createSignal<FileTarget | null>(null)
  let fileFocusNonce = 0
  const prefetch = usePrefetch()
  const queryClient = useQueryClient()

  const [picks, setPicks] = createStore<{ items: TmuxPick[] }>({ items: [] })
  const modal = useModal()
  const [tmuxBusy, setTmuxBusy] = createSignal(false)
  const [tmuxError, setTmuxError] = createSignal("")

  const tmuxSession = useTmuxSession(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const live = () => (tmuxSession.isSuccess ? (tmuxSession.data ?? null) : null)

  const tmuxBadge = () => {
    if (picks.items.length > 0) return { kind: "picked", count: picks.items.length }
    if (live()) return { kind: "live", count: live()!.windows }
    return { kind: "", count: 0 }
  }

  const pickLine = (file: string, line: number, side: "additions" | "deletions") => {
    const pinned = side === "deletions" ? undefined : line
    const i = picks.items.findIndex((p) => p.file === file)
    setPicks(
      produce((s) => {
        if (i === -1) s.items.push({ file, line: pinned })
        else s.items[i].line = pinned
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
    modal.open("tmux")
  }

  const confirmTmux = async () => {
    if (picks.items.length === 0) return
    setTmuxBusy(true)
    setTmuxError("")
    try {
      await api.tmux.open(
        props.owner,
        props.repo,
        props.prNumber,
        picks.items.map((p) => ({ ...p })),
      )
      setPicks("items", [])
      await queryClient.invalidateQueries({
        queryKey: tmuxSessionKey(props.owner, props.repo, props.prNumber),
      })
    } catch (e) {
      setTmuxError(errText(e))
    } finally {
      setTmuxBusy(false)
    }
  }

  createEffect(() => {
    if (props.tab === "files") setFilesMounted(true)
  })

  const focusLine = (file: string, line: number, side: "additions" | "deletions", rank: number) => {
    setFocus({ file, line, side, rank })
    prefetch.diff(props.owner, props.repo, props.prNumber)
    props.onTabChange("files")
  }

  const focusFile = (file: string) => {
    setFileFocus({ file, nonce: fileFocusNonce++ })
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

  const anchorTarget = (): DiffAnchor | null => {
    if (props.tab !== "files") return null

    const f = focus()
    if (f) return { file: f.file, line: f.line, side: f.side }

    const picked = picks.items.filter((p) => p.line != null)
    const pick = picked[picked.length - 1]
    return pick ? { file: pick.file, line: pick.line!, side: "additions" } : null
  }

  const anchor = createDiffAnchor(anchorTarget)

  const githubUrl = () => {
    const base = prUrl(props.owner, props.repo, props.prNumber)
    if (props.tab !== "files") return base
    return `${base}/files${anchor() ?? ""}`
  }

  const prefetchFiles = () => {
    prefetch.diff(props.owner, props.repo, props.prNumber)
  }

  return (
    <div class="pr-detail">
      <div class="pr-tabs">
        <div class="pr-tabs-main">
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

          <Show when={detailData()} keyed>
            {(d) => (
              <span class="pr-stats">
                <span class="additions">+{d.files.reduce((n, f) => n + f.Additions, 0)}</span>
                <span class="deletions">-{d.files.reduce((n, f) => n + f.Deletions, 0)}</span>
                <span class="muted">{d.pr.Author}</span>
              </span>
            )}
          </Show>
        </div>

        <div class="pr-tabs-side">
          <a
            class="pr-github-link"
            href={githubUrl()}
            target="_blank"
            rel="noreferrer"
            title="continue in github"
            aria-label="continue in github"
          >
            <span class="brand-mark is-github" />
          </a>

          <button
            class={`tmux-btn ${tmuxBadge().kind}`}
            title={live() ? "tmux session active" : "open selected lines in nvim"}
            aria-label={live() ? "tmux session active" : "open selected lines in nvim"}
            onClick={openTmuxModal}
          >
            <span class="brand-mark is-tmux" />
            <Show when={tmuxBadge().count}>
              <span class="tmux-btn-count">{tmuxBadge().count}</span>
            </Show>
          </button>
        </div>
      </div>

      <div class="pr-tab-content">
        <Show when={props.tab === "description"}>
          <div class="review-layout">
            <div class="review-diff pr-description">
              <div class="pr-description-body">
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
            </div>
            <aside class="review-rail desc-rail">
              <Show when={detailData()} keyed>
                {(d) => (
                  <div class="desc-files">
                    <div class="desc-files-head">files affected</div>
                    <For each={groupByDir(d.files, (f) => f.Filename)}>
                      {(group) => {
                        const addWidth = Math.max(
                          ...group.files.map((f) => `+${f.Additions}`.length),
                        )
                        return (
                          <div class="desc-group">
                            <div class="desc-group-dir" title={group.dir || "/"}>
                              <Icon of={dirIcon(group.dir)} />
                              <span class="desc-group-path">{group.dir || "./"}</span>
                            </div>
                            <For each={group.files}>
                              {(f) => (
                                <button
                                  class="desc-file-row"
                                  onClick={() => focusFile(f.Filename)}
                                  onMouseEnter={prefetchFiles}
                                  title={f.Filename}
                                >
                                  <Icon of={fileIcon(f.Filename)} />
                                  <span class="desc-file-name">{basePart(f.Filename)}</span>
                                  <span class="desc-file-stat">
                                    <span class="additions" style={{ "min-width": `${addWidth}ch` }}>
                                      +{f.Additions}
                                    </span>
                                    <span class="deletions">-{f.Deletions}</span>
                                  </span>
                                </button>
                              )}
                            </For>
                          </div>
                        )
                      }}
                    </For>
                  </div>
                )}
              </Show>
              <Show when={detail.isPending}>
                <div class="desc-files-pending">
                  <SkeletonLines />
                </div>
              </Show>
            </aside>
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
                  sides={diffSides(detailData()?.pr)}
                  focus={focus()}
                  fileFocus={fileFocus()}
                  threads={threads() ?? []}
                  onPickLine={pickLine}
                  onUnpickLine={removePick}
                />
              </div>
              <aside class="review-rail">
                <ReviewPanel
                  state={review}
                  activeRank={focus()?.rank}
                  onFocusConcern={focusConcern}
                />
                <ReviewerComments
                  threads={threads() ?? []}
                  pending={comments.isPending}
                  onFocusNote={focusNote}
                />
              </aside>
            </div>
          </div>
        </Show>

        <Show when={props.tab === "review"}>
          <ReviewPage state={review} onFocusConcern={focusConcern} />
        </Show>
      </div>

      <Show when={modal.isOpen("tmux")}>
        <TmuxModal
          picks={picks.items}
          live={live()}
          busy={tmuxBusy()}
          error={tmuxError()}
          onRemove={removePick}
          onConfirm={confirmTmux}
          onClose={modal.close}
        />
      </Show>
    </div>
  )
}
