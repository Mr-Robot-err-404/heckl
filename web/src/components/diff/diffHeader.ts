import { buildCopyButton } from "../copyButton"
import { checkIcon, chevronIcon, fileContentsIcon } from "./icons"
import type { FileDiffMetadata } from "@pierre/diffs"
import { copyText, COPIED_MS } from "../../clipboard"

type NamedFile = { name: string }

export type DiffItemContext = {
  item: { id: string; collapsed?: boolean }
}

export function buildCollapseToggle(
  fileDiff: NamedFile,
  context: DiffItemContext | undefined,
  onToggle: (id: string) => void,
) {
  const id = context?.item.id ?? fileDiff.name
  const collapsed = context?.item.collapsed ?? false

  const button = document.createElement("button")
  button.type = "button"
  button.className = "diff-header-btn diff-collapse-btn"
  button.setAttribute("aria-label", collapsed ? "expand file" : "collapse file")
  button.style.transform = collapsed ? "rotate(-90deg)" : "rotate(0deg)"
  button.appendChild(chevronIcon())
  button.addEventListener("click", (e) => {
    e.stopPropagation()
    onToggle(id)
  })

  const slot = document.createElement("span")
  slot.className = "diff-collapse-slot"
  slot.dataset.file = id
  slot.appendChild(button)
  return slot
}

export function buildCopyPathButton(fileDiff: NamedFile) {
  return buildCopyButton(() => fileDiff.name, "copy file path")
}

export function buildFileCopyButtons(
  fileDiff: FileDiffMetadata,
  contents: () => Promise<string | null>,
) {
  const buttons = document.createElement("span")
  buttons.className = "diff-file-copy-actions"
  buttons.appendChild(buildCopyPathButton(fileDiff))
  if (fileDiff.type !== "deleted" && fileDiff.hunks.length > 0) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "diff-header-btn diff-copy-btn"
    button.title = "copy file contents"
    button.setAttribute("aria-label", "copy file contents")
    button.appendChild(fileContentsIcon())
    button.addEventListener("click", async (event) => {
      event.stopPropagation()
      if (button.disabled) return
      button.disabled = true
      try {
        const text = await contents()
        if (text == null || !(await copyText(text))) return
        button.replaceChild(checkIcon(), button.firstChild!)
        button.classList.add("copied")
        setTimeout(() => {
          if (!button.isConnected) return
          button.replaceChild(fileContentsIcon(), button.firstChild!)
          button.classList.remove("copied")
        }, COPIED_MS)
      } catch {
        // No readable text for this file.
      } finally {
        button.disabled = false
      }
    })
    buttons.appendChild(button)
  }
  return buttons
}
