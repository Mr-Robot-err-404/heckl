import { createEffect, onCleanup } from "solid-js"
import { CodeView, parsePatchFiles, type CodeViewItem } from "@pierre/diffs"
import { useDiff } from "../queries"

type Props = {
  owner: string
  repo: string
  prNumber: number
}

export function DiffView(props: Props) {
  let host!: HTMLDivElement
  let view: InstanceType<typeof CodeView> | null = null

  const diff = useDiff(
    () => props.owner,
    () => props.repo,
    () => props.prNumber,
  )

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
      theme: { dark: "pierre-dark", light: "pierre-light" },
      hunkSeparators: "line-info",
      diffStyle: "unified",
      diffIndicators: "bars",
      lineDiffType: "word-alt",
      stickyHeaders: true,
      layout: { paddingTop: 8, paddingBottom: 8, gap: 8 },
    })
    view.setup(host)
    view.setItems(items)
    view.render()
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
