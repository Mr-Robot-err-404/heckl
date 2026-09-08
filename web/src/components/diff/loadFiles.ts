import type { FileDiffLoadedFiles, FileDiffMetadata } from "@pierre/diffs"
import type { DiffBlob, DiffPrefetch, DiffSides, PR } from "../../types"

export function diffSides(pr: PR | undefined): DiffSides | undefined {
  if (!pr?.baseSha || !pr.headSha) return undefined
  return {
    base: { sha: pr.baseSha, ref: pr.baseRef ?? "" },
    head: { sha: pr.headSha, ref: pr.headRef ?? "" },
  }
}

export function prefetchBody(sides: DiffSides, files: FileDiffMetadata[]): DiffPrefetch {
  return {
    base: { ...sides.base, paths: files.map((f) => f.prevName ?? f.name) },
    head: { ...sides.head, paths: files.map((f) => f.name) },
  }
}

export function toLoadedFiles(blob: DiffBlob, name: string): FileDiffLoadedFiles {
  const oldFile = blob.oldFile
    ? { name: blob.oldFile.name, contents: blob.oldFile.contents }
    : null
  const newFile = blob.newFile
    ? { name: blob.newFile.name, contents: blob.newFile.contents }
    : null

  if (!oldFile || !newFile) {
    throw new Error(`cannot expand ${name} - missing base or head contents`)
  }
  return { oldFile, newFile }
}
