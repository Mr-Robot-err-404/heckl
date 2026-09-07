import { relativeTime } from "../../review"
import { renderMarkdown } from "../Markdown"
import type { ReviewerNote, ReviewerThread } from "../../types"

export type NoteMetadata = {
  key: string
  entries: { login: string; avatar: string; note: ReviewerNote }[]
}

export type DiffAnnotation = {
  side: "additions" | "deletions"
  lineNumber: number
  metadata: NoteMetadata
}

function anchored(note: ReviewerNote): boolean {
  return !!note.file && note.line != null && !!note.side && !note.outdated
}

export function annotationsByFile(threads: ReviewerThread[]): Map<string, DiffAnnotation[]> {
  const byKey = new Map<string, DiffAnnotation>()

  for (const thread of threads) {
    if (thread.bot) continue

    for (const note of thread.notes) {
      if (!anchored(note)) continue

      const key = `${note.file}:${note.side}:${note.line}`
      let annotation = byKey.get(key)
      if (!annotation) {
        annotation = {
          side: note.side!,
          lineNumber: note.line!,
          metadata: { key, entries: [] },
        }
        byKey.set(key, annotation)
      }
      annotation.metadata.entries.push({
        login: thread.user.login,
        avatar: thread.user.avatar,
        note,
      })
    }
  }

  const byFile = new Map<string, DiffAnnotation[]>()
  for (const annotation of byKey.values()) {
    annotation.metadata.entries.sort((a, b) =>
      a.note.createdAt < b.note.createdAt ? -1 : 1,
    )
    const file = annotation.metadata.key.split(":")[0]
    const list = byFile.get(file) ?? []
    list.push(annotation)
    byFile.set(file, list)
  }
  return byFile
}

export function annotationsFingerprint(byFile: Map<string, DiffAnnotation[]>): string {
  return [...byFile.entries()]
    .map(([file, list]) => `${file}#${list.map((a) => a.metadata.key).sort().join(",")}`)
    .sort()
    .join("|")
}

export function buildAnnotationNode(annotation: { metadata?: NoteMetadata }): HTMLElement {
  const box = document.createElement("div")
  box.className = "diff-note"

  for (const entry of annotation.metadata?.entries ?? []) {
    box.appendChild(buildNote(entry))
  }
  return box
}

function buildNote(entry: { login: string; avatar: string; note: ReviewerNote }): HTMLElement {
  const wrap = document.createElement("div")
  wrap.className = "diff-note-item"

  const head = document.createElement("div")
  head.className = "diff-note-head"

  if (entry.avatar) {
    const avatar = document.createElement("img")
    avatar.className = "diff-note-avatar"
    avatar.src = entry.avatar
    avatar.alt = ""
    head.appendChild(avatar)
  }

  const login = document.createElement("span")
  login.className = "diff-note-login"
  login.textContent = entry.login
  head.appendChild(login)

  const when = document.createElement("span")
  when.className = "diff-note-time"
  when.textContent = relativeTime(entry.note.createdAt)
  head.appendChild(when)

  if (entry.note.url) {
    const link = document.createElement("a")
    link.className = "diff-note-link"
    link.href = entry.note.url
    link.target = "_blank"
    link.rel = "noreferrer"
    link.textContent = "↗"
    head.appendChild(link)
  }

  const body = document.createElement("div")
  body.className = "markdown diff-note-body"
  body.innerHTML = renderMarkdown(entry.note.body)

  wrap.appendChild(head)
  wrap.appendChild(body)
  return wrap
}
