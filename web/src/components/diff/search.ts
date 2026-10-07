import type { FileDiffMetadata } from "@pierre/diffs"

export type DiffMatch = {
  file: string
  line: number
  side: "additions" | "deletions"
  column: number
}

export function searchDiff(files: FileDiffMetadata[], query: string): DiffMatch[] {
  if (!query) return []
  const needle = query.toLocaleLowerCase()
  const matches: DiffMatch[] = []

  for (const file of files) {
    for (const hunk of file.hunks) {
      let additionLine = hunk.additionStart
      let deletionLine = hunk.deletionStart

      const addMatches = (text: string, line: number, side: DiffMatch["side"]) => {
        const lower = text.toLocaleLowerCase()
        let from = 0
        while (true) {
          const column = lower.indexOf(needle, from)
          if (column === -1) break
          matches.push({ file: file.name, line, side, column })
          from = column + needle.length
        }
      }

      for (const block of hunk.hunkContent) {
        if (block.type === "context") {
          for (let i = 0; i < block.lines; i++) {
            addMatches(file.additionLines[block.additionLineIndex + i] ?? "", additionLine++, "additions")
            deletionLine++
          }
        } else {
          for (let i = 0; i < block.deletions; i++) {
            addMatches(file.deletionLines[block.deletionLineIndex + i] ?? "", deletionLine++, "deletions")
          }
          for (let i = 0; i < block.additions; i++) {
            addMatches(file.additionLines[block.additionLineIndex + i] ?? "", additionLine++, "additions")
          }
        }
      }
    }
  }

  return matches
}
