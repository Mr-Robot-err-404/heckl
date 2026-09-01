import { createEffect, onCleanup } from "solid-js"
import { CodeView, parsePatchFiles, type CodeViewItem } from "@pierre/diffs"
import { useDiff } from "../../queries"
import { buildCollapseToggle, buildCopyPathButton, type DiffItemContext } from "./diffHeader"
import type { ConcernTarget } from "../../types"

type Props = {
  owner: string
  repo: string
  prNumber: number
  focus: ConcernTarget | null
}

export function DiffView(props: Props) {
  let host!: HTMLDivElement
  let view: InstanceType<typeof CodeView> | null = null

  const diff = useDiff(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )

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
    const patch = diff.data
    if (!patch) return

    const items: CodeViewItem[] = parsePatchFiles(
      patch,
      `${props.owner}/${props.repo}/${props.prNumber}`,
    ).flatMap((p) =>
      p.files.map((fileDiff) => ({
        id: `${fileDiff.name}`,
        type: "diff" as const,
        fileDiff,
      }))
    )

    view?.cleanUp()
    view = new CodeView({
      theme: "gruvbox-dark-medium",
      hunkSeparators: "line-info",
      diffStyle: "unified",
      diffIndicators: "bars",
      lineDiffType: "word-alt",
      stickyHeaders: true,
      enableLineSelection: true,
      layout: { paddingTop: 8, paddingBottom: 8, gap: 8 },
      unsafeCSS: "[data-change-icon] { display: none; }",
      renderHeaderPrefix: (fileDiff, context: unknown) =>
        buildCollapseToggle(fileDiff, context as DiffItemContext, toggleCollapsed),
      renderHeaderFilenameSuffix: (fileDiff) => buildCopyPathButton(fileDiff),
    })
    view.setup(host)
    view.setItems(items)
    view.render()
  })

  createEffect(() => {
    const target = props.focus
    if (!diff.data || !target || !view) return

    const item = view.getItem(target.file)
    if (!item) return
    if (item.collapsed) {
      view.updateItem({ ...item, collapsed: false, version: (item.version ?? 0) + 1 })
    }

    view.setSelectedLines({
      id: target.file,
      range: { start: target.line, end: target.line, side: target.side },
    })
    view.scrollTo({
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
      <div ref={host} class="diffview-host" />
    </>
  )
}
