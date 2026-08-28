import { For, Show } from "solid-js"
import { usePRDetail } from "../queries"
import type { PR, Repo } from "../types"
import { DiffView } from "./DiffView"

type Props = {
  repo: Repo
  pr: PR
}

export function PRDetail(props: Props) {
  const detail = usePRDetail(
    () => props.repo.Owner,
    () => props.repo.Name,
    () => props.pr.Number,
  )

  return (
    <div class="pr-detail">
      <Show when={detail.data} keyed>
        {(d) => {
          const additions = d.files.reduce((n, f) => n + f.Additions, 0)
          const deletions = d.files.reduce((n, f) => n + f.Deletions, 0)
          return (
            <>
              <div class="pr-detail-header">
                <h2>{d.pr.Title}</h2>
                <div class="pr-detail-meta">
                  <span>#{d.pr.Number}</span>
                  <span>{d.pr.Author}</span>
                  <Show when={d.pr.Draft}>
                    <span class="badge draft">draft</span>
                  </Show>
                  <span class="additions">+{additions}</span>
                  <span class="deletions">-{deletions}</span>
                  <span>{d.files.length} files</span>
                </div>
                <Show when={d.pr.Body}>
                  <p class="pr-body">{d.pr.Body}</p>
                </Show>
              </div>
              <div class="diff-list">
                <For each={d.files}>
                  {(file) => (
                    <Show when={file.Patch}>
                      <DiffView file={file} />
                    </Show>
                  )}
                </For>
              </div>
            </>
          )
        }}
      </Show>
      <Show when={detail.isLoading}>
        <div class="status">loading diff...</div>
      </Show>
    </div>
  )
}
