import type { FileDiffMetadata } from "@pierre/diffs"
import { checkIcon, chevronIcon, clipboardIcon } from "./icons"

type DiffItemContext = {
  item: { id: string; collapsed?: boolean }
}

export function buildCollapseToggle(
  fileDiff: FileDiffMetadata,
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
  return button
}

export function buildCopyPathButton(fileDiff: FileDiffMetadata) {
  const button = document.createElement("button")
  button.type = "button"
  button.className = "diff-header-btn diff-copy-btn"
  button.setAttribute("aria-label", "copy file path")
  const icon = clipboardIcon()
  button.appendChild(icon)

  button.addEventListener("click", (e) => {
    e.stopPropagation()
    navigator.clipboard.writeText(fileDiff.name).then(() => {
      button.replaceChild(checkIcon(), button.firstChild!)
      button.classList.add("copied")
      setTimeout(() => {
        if (!button.isConnected) return
        button.replaceChild(clipboardIcon(), button.firstChild!)
        button.classList.remove("copied")
      }, 1200)
    })
  })

  return button
}
