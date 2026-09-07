import DOMPurify from "dompurify"
import { marked } from "marked"

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

export function renderMarkdown(content: string): string {
  const raw = marked.parse(content, { async: false }) as string
  return DOMPurify.sanitize(raw, {
    USE_PROFILES: { html: true },
    ADD_TAGS: ["img"],
    ADD_ATTR: ["src", "alt", "width", "height"],
  })
}

export function Markdown(props: Props) {
  return <div class="markdown" innerHTML={renderMarkdown(props.content)} />
}
