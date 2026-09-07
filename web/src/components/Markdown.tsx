import DOMPurify from "dompurify"
import { marked } from "marked"
import { createEffect, createMemo } from "solid-js"
import { highlightCode, highlightVersion } from "../highlight"
import { theme } from "../theme"
import { buildCopyButton } from "./copyButton"

type Props = {
  content: string
}

const GITHUB_ASSET_PROXY_HOSTS = [
  "github.com",
  "user-images.githubusercontent.com",
  "private-user-images.githubusercontent.com",
]

function needsAssetProxy(src: string): boolean {
  try {
    return GITHUB_ASSET_PROXY_HOSTS.includes(new URL(src).host)
  } catch {
    return false
  }
}

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "IMG") {
    const src = node.getAttribute("src")
    if (src && needsAssetProxy(src)) {
      node.setAttribute("src", `/api/asset?url=${encodeURIComponent(src)}`)
    }
  }
})

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
  })

  return <div ref={root} class="markdown" />
}
