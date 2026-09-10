import { createEffect, createSignal, onCleanup, type Accessor } from "solid-js"

export type DiffAnchor = {
  file: string
  line: number
  side: "additions" | "deletions"
}

export function prUrl(owner: string, repo: string, number: number) {
  return `https://github.com/${owner}/${repo}/pull/${number}`
}

const digests = new Map<string, string>()

async function fileDigest(file: string): Promise<string> {
  const cached = digests.get(file)
  if (cached != null) return cached
  if (!crypto?.subtle) return ""

  const bytes = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(file))
  const hash = Array.from(new Uint8Array(bytes))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")

  digests.set(file, hash)
  return hash
}

function anchorFor(hash: string, target: DiffAnchor) {
  if (!hash) return ""
  return `#diff-${hash}${target.side === "deletions" ? "L" : "R"}${target.line}`
}

export function createDiffAnchor(target: Accessor<DiffAnchor | null>): Accessor<string> {
  const [anchor, setAnchor] = createSignal("")

  createEffect(() => {
    const t = target()
    if (!t) {
      setAnchor("")
      return
    }

    const cached = digests.get(t.file)
    if (cached != null) {
      setAnchor(anchorFor(cached, t))
      return
    }

    let stale = false
    onCleanup(() => (stale = true))
    void fileDigest(t.file).then((hash) => {
      if (!stale) setAnchor(anchorFor(hash, t))
    })
  })

  return anchor
}
