import type { FileDiffLoadedFiles, FileDiffMetadata } from "@pierre/diffs"
import { api } from "../../api"

export async function loadDiffFiles(
  owner: string,
  repo: string,
  prNumber: number,
  fileDiff: FileDiffMetadata,
): Promise<FileDiffLoadedFiles> {
  const blob = await api.diff.blob(
    owner,
    repo,
    prNumber,
    fileDiff.name,
    fileDiff.prevName,
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
