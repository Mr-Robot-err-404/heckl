import { onCleanup, onMount } from "solid-js"
import { FileDiff } from "@pierre/diffs"
import type { PRFile } from "../types"

type Props = {
  file: PRFile
}

export function DiffView(props: Props) {
  let container!: HTMLDivElement

  onMount(() => {
    const instance = new FileDiff() as unknown as HTMLElement & {
      patch: string
      filename: string
    }
    instance.patch = props.file.Patch
    instance.filename = props.file.Filename
    container.appendChild(instance)
  })

  onCleanup(() => {
    container.innerHTML = ""
  })

  return <div ref={container} />
}
