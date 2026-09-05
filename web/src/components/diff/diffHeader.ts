import { copyText, COPIED_MS } from "../../clipboard"
import { checkIcon, chevronIcon, clipboardIcon } from "./icons"

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
  const button = document.createElement("button")
  button.type = "button"
  button.className = "diff-header-btn diff-copy-btn"
  button.setAttribute("aria-label", "copy file path")
  button.appendChild(clipboardIcon())

  button.addEventListener("click", async (e) => {
    e.stopPropagation()
    if (!(await copyText(fileDiff.name))) return
    showCopied(button)
  })

  return button
}

function showCopied(button: HTMLButtonElement) {
  button.replaceChild(checkIcon(), button.firstChild!)
  button.classList.add("copied")
  setTimeout(() => {
    if (!button.isConnected) return
    button.replaceChild(clipboardIcon(), button.firstChild!)
    button.classList.remove("copied")
  }, COPIED_MS)
}
