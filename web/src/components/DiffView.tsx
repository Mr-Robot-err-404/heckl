import { onCleanup, onMount } from "solid-js"
import { FileDiff, getSingularPatch } from "@pierre/diffs"
import type { PRFile } from "../types"

type Props = {
  file: PRFile
}

function buildPatch(file: PRFile): string {
  const a = `a/${file.Filename}`
  const b = `b/${file.Filename}`
  return `diff --git ${a} ${b}\n--- ${a}\n+++ ${b}\n${file.Patch}`
}

export function DiffView(props: Props) {
  let container!: HTMLDivElement
  let instance: InstanceType<typeof FileDiff> | null = null

  onMount(() => {
    const fileDiff = getSingularPatch(buildPatch(props.file))
    instance = new FileDiff()
    instance.hydrate({ fileDiff, fileContainer: container })
  })

  onCleanup(() => {
    instance?.cleanUp()
    instance = null
  })

  return <div ref={container} />
}
