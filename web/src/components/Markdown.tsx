import DOMPurify from "dompurify"
import { marked } from "marked"

type Props = {
  content: string
}

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
