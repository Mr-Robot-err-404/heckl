import { For, Show } from "solid-js"
import { usePRDetail } from "../queries"
import type { PR, Repo } from "../types"
import { DiffView } from "./DiffView"

type Props = {
  repo: Repo
  pr: PR
  onBack: () => void
}

export function PRDetail(props: Props) {
  const detail = usePRDetail(
    () => props.repo.Owner,
    () => props.repo.Name,
    () => props.pr.Number,
  )

  return (
    <div class="pr-detail">
      <div class="pr-detail-header">
        <button class="back-btn" onClick={props.onBack}>← back</button>
        <Show when={detail.data} keyed>
          {(d) => {
            const additions = d.files.reduce((n, f) => n + f.Additions, 0)
            const deletions = d.files.reduce((n, f) => n + f.Deletions, 0)
            return (
              <>
                <h1 class="pr-detail-title">{d.pr.Title}</h1>
                <div class="pr-detail-meta">
                  <span class="pr-number">#{d.pr.Number}</span>
                  <span>{d.pr.Author}</span>
                  <Show when={d.pr.Draft}>
                    <span class="badge draft">draft</span>
                  </Show>
                  <span class="additions">+{additions}</span>
                  <span class="deletions">-{deletions}</span>
                  <span class="muted">{d.files.length} files changed</span>
                </div>
                <Show when={d.pr.Body}>
                  <p class="pr-body">{d.pr.Body}</p>
                </Show>
              </>
            )
          }}
        </Show>
        <Show when={detail.isLoading}>
          <div class="muted">loading...</div>
        </Show>
      </div>
      <Show when={detail.data} keyed>
        {(d) => (
          <div class="diff-list">
            <For each={d.files}>
              {(file) => (
                <Show when={file.Patch}>
                  <DiffView file={file} />
                </Show>
              )}
            </For>
          </div>
        )}
      </Show>
    </div>
  )
}
