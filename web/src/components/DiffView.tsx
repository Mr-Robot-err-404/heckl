import { createEffect, createSignal, onCleanup } from "solid-js"
import { CodeView, parsePatchFiles, type CodeViewItem } from "@pierre/diffs"

type Props = {
  owner: string
  repo: string
  prNumber: number
}

export function DiffView(props: Props) {
  let host!: HTMLDivElement
  let view: InstanceType<typeof CodeView> | null = null
  const [error, setError] = createSignal<string | null>(null)

  createEffect(() => {
    const owner = props.owner
    const repo = props.repo
    const number = props.prNumber

    const controller = new AbortController()

    async function load() {
      try {
        const res = await fetch(`/api/diff/${owner}/${repo}/${number}`, {
          signal: controller.signal,
        })
        if (!res.ok) throw new Error(`${res.status}`)
        const patch = await res.text()
        const patches = parsePatchFiles(patch, `${owner}/${repo}/${number}`)
        const items: CodeViewItem[] = patches.flatMap((p) =>
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
      } catch (e: any) {
        if (e?.name !== "AbortError") setError(String(e))
      }
    }

    load()

    return () => {
      controller.abort()
      view?.cleanUp()
      view = null
    }
  })

  onCleanup(() => {
    view?.cleanUp()
    view = null
  })

  return (
    <div class="diffview-wrap">
      {error() && <div class="muted">{error()}</div>}
      <div ref={host} class="diffview-host" />
    </div>
  )
}
