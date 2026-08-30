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

const BARE_GITHUB_ASSET_URL = /^(https:\/\/(?:github\.com\/user-attachments\/assets|user-images\.githubusercontent\.com)\/\S+)$/gm

function embedBareVideoUrls(markdown: string): string {
  return markdown.replace(BARE_GITHUB_ASSET_URL, (url) => `<video controls preload="metadata" src="${url}"></video>`)
}

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "IMG" || node.tagName === "VIDEO") {
    const src = node.getAttribute("src")
    if (src && needsAssetProxy(src)) {
      node.setAttribute("src", `/api/asset?url=${encodeURIComponent(src)}`)
    }
  }
})

export function Markdown(props: Props) {
  const html = () => {
    const raw = marked.parse(embedBareVideoUrls(props.content), { async: false }) as string
    return DOMPurify.sanitize(raw, {
      USE_PROFILES: { html: true },
      ADD_TAGS: ["img", "video"],
      ADD_ATTR: ["src", "alt", "width", "height", "controls", "preload"],
    })
  }

  return <div class="markdown" innerHTML={html()} />
}
