import { createEffect, createSignal, onCleanup, onMount, Show, untrack } from "solid-js"
import {
  CodeView,
  parsePatchFiles,
  type CodeViewItem,
  type FileDiffMetadata,
} from "@pierre/diffs"
import { blobOptions, useDiff, resolved } from "../../queries"
import { useQueryClient } from "@tanstack/solid-query"
import { SkeletonDiff } from "../Skeleton"
import { buildCollapseToggle, buildCopyPathButton, type DiffItemContext } from "./diffHeader"
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
import type { ConcernTarget, DiffSides, ReviewerThread } from "../../types"
import { theme } from "../../theme"
import { highlightVersion } from "../../highlight"

type Props = {
  owner: string
  repo: string
  prNumber: number
  sides?: DiffSides
  focus: ConcernTarget | null
  threads?: ReviewerThread[]
  onPickLine?: (file: string, line: number, side: "additions" | "deletions") => void
  onUnpickLine?: (file: string) => void
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
  let view: CodeView<NoteMetadata> | null = null
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
    setRendered(false)
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
      renderHeaderFilenameSuffix: (fileDiff) => buildCopyPathButton(fileDiff),
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

  onMount(() => host.addEventListener("mouseover", warmOnHover))

  onCleanup(() => {
    host.removeEventListener("mouseover", warmOnHover)
    view?.cleanUp()
    view = null
  })

  return (
    <>
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
