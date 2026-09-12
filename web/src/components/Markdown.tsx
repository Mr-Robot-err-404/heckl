import DOMPurify from "dompurify"
import { marked } from "marked"
import { createEffect, createMemo } from "solid-js"
import { highlightCode, highlightVersion } from "../highlight"
import { assetSrc, preloadImage, readyImage } from "../images"
import { theme } from "../theme"
import { buildCopyButton } from "./copyButton"

type Props = {
  content: string
}

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "IMG") {
    const src = node.getAttribute("src")
    if (src) node.setAttribute("src", assetSrc(src))
  }
})

function adopt(node: HTMLImageElement, from: HTMLImageElement): HTMLImageElement {
  for (const attr of from.attributes) {
    if (attr.name === "src") continue
    node.setAttribute(attr.name, attr.value)
  }
  return node
}

export function hydrateImages(root: HTMLElement) {
  for (const img of Array.from(root.querySelectorAll("img"))) {
    const src = img.getAttribute("src")
    if (!src) continue

    const done = readyImage(src)
    if (done) {
      img.replaceWith(adopt(done, img))
      continue
    }

    const slot = document.createElement("span")
    slot.className = "skeleton markdown-img-skeleton"
    img.replaceWith(slot)

    preloadImage(src)
      .then(() => {
        const node = readyImage(src)
        if (slot.isConnected && node) slot.replaceWith(adopt(node, img))
      })
      .catch(() => {
        if (slot.isConnected) slot.replaceWith(img)
      })
  }
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
}

function plainCodeBlock(text: string, lang: string): string {
  const cls = lang ? ` class="language-${escapeHtml(lang)}"` : ""
  return `<pre><code${cls}>${escapeHtml(text)}</code></pre>`
}

marked.use({
  renderer: {
    code({ text, lang }) {
      const name = (lang ?? "").trim().split(/\s+/)[0].toLowerCase()
      if (!name) return plainCodeBlock(text, "")
      return highlightCode(text, name, theme().shiki) ?? plainCodeBlock(text, name)
    },
  },
})

export function renderMarkdown(content: string): string {
  const raw = marked.parse(content, { async: false }) as string
  return DOMPurify.sanitize(raw, {
    USE_PROFILES: { html: true },
    ADD_TAGS: ["img"],
    ADD_ATTR: ["src", "alt", "width", "height", "style", "class"],
  })
}

export function attachCodeCopyButtons(root: HTMLElement) {
  for (const pre of root.querySelectorAll("pre")) {
    if (pre.parentElement?.classList.contains("code-block")) continue
    const code = pre.querySelector("code")
    if (!code) continue

    const wrap = document.createElement("div")
    wrap.className = "code-block"
    pre.replaceWith(wrap)
    wrap.appendChild(pre)
    wrap.appendChild(buildCopyButton(() => code.textContent ?? "", "copy code"))
  }
}

export function Markdown(props: Props) {
  let root!: HTMLDivElement

  const html = createMemo(() => {
    highlightVersion()
    return renderMarkdown(props.content)
  })

  createEffect(() => {
    root.innerHTML = html()
    attachCodeCopyButtons(root)
    hydrateImages(root)
  })

  return <div ref={root} class="markdown" />
}
