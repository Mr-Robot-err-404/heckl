import DOMPurify from "dompurify"
import { marked } from "marked"

type Props = {
  content: string
}

// GitHub-hosted asset domains that 404 without an authenticated session
// (private repo attachments). These get proxied through /api/asset so the
// server can attach the gh auth token; anything else loads directly.
const AUTHED_ASSET_HOSTS = [
  "github.com",
  "user-images.githubusercontent.com",
  "private-user-images.githubusercontent.com",
]

function needsProxy(src: string): boolean {
  try {
    return AUTHED_ASSET_HOSTS.includes(new URL(src).host)
  } catch {
    return false
  }
}

DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node.tagName === "IMG") {
    const src = node.getAttribute("src")
    if (src && needsProxy(src)) {
      node.setAttribute("src", `/api/asset?url=${encodeURIComponent(src)}`)
    }
  }
})

export function Markdown(props: Props) {
  const html = () => {
    const raw = marked.parse(props.content, { async: false }) as string
    return DOMPurify.sanitize(raw, {
      USE_PROFILES: { html: true },
      ADD_TAGS: ["img"],
      ADD_ATTR: ["src", "alt", "width", "height"],
    })
  }

  return <div class="markdown" innerHTML={html()} />
}
