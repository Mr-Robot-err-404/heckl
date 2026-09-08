import type { FileDiffLoadedFiles, FileDiffMetadata } from "@pierre/diffs"
import { api } from "../../api"
import type { DiffPrefetch, DiffSides, PR } from "../../types"

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

export async function loadDiffFiles(
  owner: string,
  repo: string,
  prNumber: number,
  fileDiff: FileDiffMetadata,
  sides?: DiffSides,
): Promise<FileDiffLoadedFiles> {
  const blob = await api.diff.blob(
    owner,
    repo,
    prNumber,
    fileDiff.name,
    fileDiff.prevName,
    sides,
  )

  const oldFile = blob.oldFile
    ? { name: blob.oldFile.name, contents: blob.oldFile.contents }
    : null
  const newFile = blob.newFile
    ? { name: blob.newFile.name, contents: blob.newFile.contents }
    : null

  if (!oldFile || !newFile) {
    throw new Error(`cannot expand ${fileDiff.name} - missing base or head contents`)
  }
  return { oldFile, newFile }
}
