import { createEffect, createSignal, onCleanup, Show, untrack } from "solid-js"
import { CodeView, parsePatchFiles, type CodeViewItem } from "@pierre/diffs"
import { useDiff, resolved } from "../../queries"
import { SkeletonDiff } from "../Skeleton"
import { buildCollapseToggle, buildCopyPathButton, type DiffItemContext } from "./diffHeader"
import {
  annotationsByFile,
  annotationsFingerprint,
  buildAnnotationNode,
  type DiffAnnotation,
  type NoteMetadata,
} from "./annotations"
import type { ConcernTarget, ReviewerThread } from "../../types"
import { theme } from "../../theme"

type Props = {
  owner: string
  repo: string
  prNumber: number
  focus: ConcernTarget | null
  threads?: ReviewerThread[]
  onPickLine?: (file: string, line: number, side: "additions" | "deletions") => void
  onUnpickLine?: (file: string) => void
}

function sameAnnotations(a: DiffAnnotation[] | undefined, b: DiffAnnotation[]): boolean {
  if ((a?.length ?? 0) !== b.length) return false
  return (a ?? []).every((item, i) => item.metadata.key === b[i].metadata.key)
}

export function DiffView(props: Props) {
  let host!: HTMLDivElement
  let view: CodeView<NoteMetadata> | null = null
  let selectedFile: string | null = null
  let fileIds: string[] = []

  const diff = useDiff(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )
  const patchData = resolved(diff)
  const [rendered, setRendered] = createSignal(false)

  const notes = () => annotationsByFile(props.threads ?? [])

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
    const patch = patchData()
    const shiki = theme().shiki
    const annotations = untrack(notes)
    if (!patch) return

    const items: CodeViewItem<NoteMetadata>[] = parsePatchFiles(
      patch,
      `${props.owner}/${props.repo}/${props.prNumber}`,
    ).flatMap((p) =>
      p.files.map((fileDiff) => ({
        id: `${fileDiff.name}`,
        type: "diff" as const,
        fileDiff,
        annotations: annotations.get(fileDiff.name) ?? [],
      }))
    )

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
      layout: { paddingTop: 8, paddingBottom: 8, gap: 8 },
      unsafeCSS: "[data-change-icon] { display: none; }",
      renderHeaderPrefix: (fileDiff, context: unknown) =>
        buildCollapseToggle(fileDiff, context as DiffItemContext, toggleCollapsed),
      renderHeaderFilenameSuffix: (fileDiff) => buildCopyPathButton(fileDiff),
      renderAnnotation: (annotation: { metadata?: NoteMetadata }) =>
        buildAnnotationNode(annotation),
    })
    fileIds = items.map((item) => item.id)
    view.setup(host)
    view.setItems(items)
    view.render()
    setRendered(true)
  })

  createEffect(() => {
    const annotations = notes()
    annotationsFingerprint(annotations)
    const v = view
    if (!v || !rendered()) return

    untrack(() => {
      for (const id of fileIds) {
        const item = v.getItem(id)
        if (!item || item.type !== "diff") continue

        const next: DiffAnnotation[] = annotations.get(id) ?? []
        if (sameAnnotations(item.annotations, next)) continue

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

  onCleanup(() => {
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
