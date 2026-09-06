export type DiffAnchor = {
  file: string
  line: number
  side: "additions" | "deletions"
}

export function prUrl(owner: string, repo: string, number: number) {
  return `https://github.com/${owner}/${repo}/pull/${number}`
}

export async function diffAnchor(target: DiffAnchor): Promise<string> {
  if (!crypto?.subtle) return ""

  const bytes = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(target.file))
  const hash = Array.from(new Uint8Array(bytes))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")

  return `#diff-${hash}${target.side === "deletions" ? "L" : "R"}${target.line}`
}
