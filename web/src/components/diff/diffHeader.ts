import { buildCopyButton } from "../copyButton"
import { chevronIcon } from "./icons"

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
  return button
}

export function buildCopyPathButton(fileDiff: NamedFile) {
  return buildCopyButton(() => fileDiff.name, "copy file path")
}
