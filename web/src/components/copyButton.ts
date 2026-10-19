import { copyText, COPIED_MS } from "../clipboard"
import { checkIcon, clipboardIcon } from "./diff/icons"

export function buildCopyButton(text: () => string, label: string, icon = clipboardIcon) {
  const button = document.createElement("button")
  button.type = "button"
  button.className = "diff-header-btn diff-copy-btn"
  button.setAttribute("aria-label", label)
  button.title = label
  button.appendChild(icon())

  button.addEventListener("click", async (e) => {
    e.stopPropagation()
    if (!(await copyText(text()))) return
  showCopied(button, icon)
  })

  return button
}

function showCopied(button: HTMLButtonElement, icon: () => SVGSVGElement) {
  button.replaceChild(checkIcon(), button.firstChild!)
  button.classList.add("copied")
  setTimeout(() => {
    if (!button.isConnected) return
    button.replaceChild(icon(), button.firstChild!)
    button.classList.remove("copied")
  }, COPIED_MS)
}
