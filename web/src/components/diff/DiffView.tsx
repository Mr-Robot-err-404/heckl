import { createEffect, createMemo, createSignal, onCleanup, onMount, Show, untrack } from "solid-js"
import {
  CodeView,
  parsePatchFiles,
  type CodeViewItem,
  type FileDiffMetadata,
} from "@pierre/diffs"
import { blobOptions, useDiff, resolved } from "../../queries"
import { useQueryClient } from "@tanstack/solid-query"
import { SkeletonDiff } from "../Skeleton"
import { buildCollapseToggle, buildFileCopyButtons, type DiffItemContext } from "./diffHeader"
import {
  annotationsByFile,
  annotationsFingerprint,
  buildAnnotationNode,
  type DiffAnnotation,
  type NoteMetadata,
} from "./annotations"
import { diffUnsafeCSS } from "./diffCss"
import { prefetchBody, toLoadedFiles } from "./loadFiles"
import { api } from "../../api"
import type { ConcernTarget, DiffSides, FileTarget, ReviewerThread } from "../../types"
import { theme } from "../../theme"
import { highlightVersion } from "../../highlight"
import { ChevronLeftIcon, ChevronRightIcon, CloseIcon } from "../icons"
import { searchDiff } from "./search"

type Props = {
  owner: string
  repo: string
  prNumber: number
  sides?: DiffSides
  focus: ConcernTarget | null
  fileFocus?: FileTarget | null
  threads?: ReviewerThread[]
  onPickLine?: (file: string, line: number, side: "additions" | "deletions") => void
  onUnpickLine?: (file: string) => void
  active: boolean
}

function sameAnnotations(a: DiffAnnotation[] | undefined, b: DiffAnnotation[]): boolean {
  if ((a?.length ?? 0) !== b.length) return false
  return (a ?? []).every((item, i) => item.metadata.key === b[i].metadata.key)
}

const SPINNER_DELAY_MS = 120

function fileHost(root: HTMLElement, id: string): HTMLElement | null {
  const slot = root.querySelector(`.diff-collapse-slot[data-file="${CSS.escape(id)}"]`)
  let el = slot?.parentElement ?? null
  while (el && !el.shadowRoot) el = el.parentElement
  return el
}

function hoveredExpandFile(event: Event): string | undefined {
  const path = event.composedPath()
  const onButton = path.some(
    (node) => node instanceof HTMLElement && node.hasAttribute("data-expand-button"),
  )
  if (!onButton) return undefined

  const container = path.find(
    (node): node is HTMLElement => node instanceof HTMLElement && node.shadowRoot != null,
  )
  return container?.querySelector<HTMLElement>(".diff-collapse-slot[data-file]")?.dataset.file
}

export function DiffView(props: Props) {
  const queryClient = useQueryClient()
  let host!: HTMLDivElement
  let searchInput: HTMLInputElement | undefined
  let view: CodeView<NoteMetadata> | null = null
  let selectionBeforeSearch: ReturnType<CodeView<NoteMetadata>["getSelectedLines"]> = null
  let selectedFile: string | null = null
  let fileIds: string[] = []
  let highlightSeen = highlightVersion()

  const diff = useDiff(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const patchData = resolved(diff)
  const [rendered, setRendered] = createSignal(false)
  const [parsedFiles, setParsedFiles] = createSignal<FileDiffMetadata[]>([])
  const [searchOpen, setSearchOpen] = createSignal(false)
  const [query, setQuery] = createSignal("")
  const [matchIndex, setMatchIndex] = createSignal(0)
  const matches = createMemo(() => searchDiff(parsedFiles(), query().trim()))
  const currentIndex = () => Math.min(matchIndex(), Math.max(0, matches().length - 1))
  let searchSelection: { file: string; line: number; side: "additions" | "deletions" } | null = null

  const closeSearch = () => {
    setSearchOpen(false)
    setQuery("")
    if (searchSelection && view?.getSelectedLines()?.id === searchSelection.file &&
      view?.getSelectedLines()?.range.start === searchSelection.line &&
      view?.getSelectedLines()?.range.side === searchSelection.side) {
      view.setSelectedLines(selectionBeforeSearch, { notify: false })
    }
    searchSelection = null
    selectionBeforeSearch = null
    searchInput?.blur()
  }

  const moveMatch = (step: number) => {
    const length = matches().length
    if (length) setMatchIndex((index) => (Math.min(index, length - 1) + step + length) % length)
  }

  createEffect(() => {
    if (!props.active && searchOpen()) closeSearch()
  })

  createEffect(() => {
    if (!searchOpen() || !rendered()) return
    const match = matches()[currentIndex()]
    const v = view
    if (!match || !v) {
      if (searchSelection && v?.getSelectedLines()?.id === searchSelection.file &&
        v?.getSelectedLines()?.range.start === searchSelection.line &&
        v?.getSelectedLines()?.range.side === searchSelection.side) {
        v.setSelectedLines(selectionBeforeSearch, { notify: false })
      }
      searchSelection = null
      return
    }
    const item = v.getItem(match.file)
    if (!item) return
    if (item.collapsed) v.updateItem({ ...item, collapsed: false, version: (item.version ?? 0) + 1 })
    v.setSelectedLines(
      { id: match.file, range: { start: match.line, end: match.line, side: match.side } },
      { notify: false },
    )
    searchSelection = match
    v.scrollTo({ type: "line", id: match.file, lineNumber: match.line, side: match.side, align: "center" })
  })

  const notes = () => annotationsByFile(props.threads ?? [])

  const blobFor = (fileDiff: FileDiffMetadata) =>
    blobOptions(
      props.owner,
      props.repo,
      props.prNumber,
      fileDiff.name,
      fileDiff.prevName,
      props.sides,
    )

  const warmOnHover = (event: Event) => {
    const id = hoveredExpandFile(event)
    if (!id) return
    const fileDiff = parsedFiles().find((f) => f.name === id)
    if (fileDiff) void queryClient.prefetchQuery(blobFor(fileDiff))
  }

  const loadFiles = async (fileDiff: FileDiffMetadata) => {
    const timer = window.setTimeout(() => {
      fileHost(host, fileDiff.name)?.setAttribute("data-expanding", "")
    }, SPINNER_DELAY_MS)

    try {
      const blob = await queryClient.fetchQuery(blobFor(fileDiff))
      return toLoadedFiles(blob, fileDiff.name)
    } finally {
      clearTimeout(timer)
      fileHost(host, fileDiff.name)?.removeAttribute("data-expanding")
    }
  }

  const toggleCollapsed = (id: string) => {
    const item = view?.getItem(id)
    if (!item) return
    view?.updateItem({
      ...item,
      collapsed: !item.collapsed,
      version: (item.version ?? 0) + 1,
    })
  }

  createEffect(() => {
    props.owner
    props.repo
    props.prNumber
    untrack(() => {
      setRendered(false)
      closeSearch()
    })
  })

  createEffect(() => {
    const sides = props.sides
    const files = parsedFiles()
    if (!sides || files.length === 0) return
    api.diff
      .prefetch(props.owner, props.repo, prefetchBody(sides, files))
      .catch(() => {})
  })

  createEffect(() => {
    const patch = patchData()
    const shiki = theme().shiki
    const annotations = untrack(notes)
    if (!patch) return

    const files = parsePatchFiles(
      patch,
      `${props.owner}/${props.repo}/${props.prNumber}`,
    ).flatMap((p) => p.files)

    const items: CodeViewItem<NoteMetadata>[] = files.map((fileDiff) => ({
      id: `${fileDiff.name}`,
      type: "diff" as const,
      fileDiff,
      annotations: annotations.get(fileDiff.name) ?? [],
    }))

    view?.cleanUp()
    selectedFile = null
    searchSelection = null
    selectionBeforeSearch = null
    view = new CodeView<NoteMetadata>({
      theme: shiki,
      hunkSeparators: "line-info",
      diffStyle: "unified",
      diffIndicators: "bars",
      lineDiffType: "word-alt",
      stickyHeaders: true,
      enableLineSelection: true,
      onSelectedLinesChange: (selection) => {
        if (!selection) {
          if (selectedFile) props.onUnpickLine?.(selectedFile)
          selectedFile = null
          return
        }
        selectedFile = selection.id
        props.onPickLine?.(
          selection.id,
          selection.range.start,
          selection.range.side ?? "additions",
        )
      },
      loadDiffFiles: loadFiles,
      expansionLineCount: 20,
      layout: { paddingTop: 8, paddingBottom: 8, gap: 0 },
      unsafeCSS: diffUnsafeCSS,
      renderHeaderPrefix: (fileDiff, context: unknown) =>
        buildCollapseToggle(fileDiff, context as DiffItemContext, toggleCollapsed),
      renderHeaderFilenameSuffix: (file) => file && "hunks" in file
        ? buildFileCopyButtons(
            file,
            async () => (await queryClient.fetchQuery(blobFor(file))).newFile?.contents ?? null,
          )
        : undefined,
      renderAnnotation: (annotation: { metadata?: NoteMetadata }) =>
        buildAnnotationNode(annotation),
    })
    fileIds = items.map((item) => item.id)
    setParsedFiles(files)
    view.setup(host)
    view.setItems(items)
    view.render()
    setRendered(true)
  })

  createEffect(() => {
    const annotations = notes()
    annotationsFingerprint(annotations)
    const version = highlightVersion()
    const v = view
    if (!v || !rendered()) return

    const forced = version !== highlightSeen
    highlightSeen = version

    untrack(() => {
      for (const id of fileIds) {
        const item = v.getItem(id)
        if (!item || item.type !== "diff") continue

        const next: DiffAnnotation[] = annotations.get(id) ?? []
        if (!forced && sameAnnotations(item.annotations, next)) continue

        v.updateItem({ ...item, annotations: next, version: (item.version ?? 0) + 1 })
      }
    })
  })

  createEffect(() => {
    const target = props.focus
    const v = view
    if (!patchData() || !target || !v) return

    const item = v.getItem(target.file)
    if (!item) return
    if (item.collapsed) {
      v.updateItem({ ...item, collapsed: false, version: (item.version ?? 0) + 1 })
    }

    v.setSelectedLines(
      {
        id: target.file,
        range: { start: target.line, end: target.line, side: target.side },
      },
      { notify: false },
    )
    selectedFile = target.file
    v.scrollTo({
      type: "line",
      id: target.file,
      lineNumber: target.line,
      side: target.side,
      align: "center",
    })
  })

  createEffect(() => {
    const target = props.fileFocus
    const v = view
    if (!patchData() || !target || !v) return

    const item = v.getItem(target.file)
    if (!item) return
    if (item.collapsed) {
      v.updateItem({ ...item, collapsed: false, version: (item.version ?? 0) + 1 })
    }

    v.scrollTo({ type: "item", id: target.file, align: "start" })
  })

  onMount(() => {
    host.addEventListener("mouseover", warmOnHover)
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && searchOpen()) {
        event.preventDefault()
        event.stopImmediatePropagation()
        closeSearch()
        return
      }
      if (!props.active || event.defaultPrevented || event.repeat || event.ctrlKey || event.altKey || event.metaKey) return
      if (event.target instanceof Element && event.target.closest("input, textarea, select, button, a, [contenteditable], [role='dialog']")) return
      if (event.key !== "/") return
      event.preventDefault()
      selectionBeforeSearch = view?.getSelectedLines() ?? null
      setSearchOpen(true)
      requestAnimationFrame(() => searchInput?.focus())
    }
    window.addEventListener("keydown", onKeyDown, true)
    onCleanup(() => window.removeEventListener("keydown", onKeyDown, true))
  })

  onCleanup(() => {
    host.removeEventListener("mouseover", warmOnHover)
    view?.cleanUp()
    view = null
  })

  return (
    <>
      <Show when={searchOpen()}>
        <div class="diff-search" role="search" onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault()
            moveMatch(event.shiftKey ? -1 : 1)
          }
        }}>
          <input
            ref={searchInput}
            type="text"
            value={query()}
            placeholder="Search diff"
            aria-label="Search diff"
            onInput={(event) => { setMatchIndex(0); setQuery(event.currentTarget.value) }}
          />
          <span class="diff-search-count" aria-live="polite">
            {query().trim() ? (matches().length ? `${currentIndex() + 1} / ${matches().length}` : "0 / 0") : ""}
          </span>
          <button title="previous match (Shift+Enter)" aria-label="previous match" disabled={!matches().length} onClick={() => moveMatch(-1)}><ChevronLeftIcon /></button>
          <button title="next match (Enter)" aria-label="next match" disabled={!matches().length} onClick={() => moveMatch(1)}><ChevronRightIcon /></button>
          <button title="close search (Esc)" aria-label="close search" onClick={closeSearch}><CloseIcon /></button>
        </div>
      </Show>
      {diff.isError && (
        <div class="muted" style="padding:16px">{String(diff.error)}</div>
      )}
      <Show when={!rendered() && !diff.isError}>
        <SkeletonDiff />
      </Show>
      <div ref={host} class="diffview-host" />
    </>
  )
}
