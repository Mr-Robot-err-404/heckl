import { For, Show } from "solid-js"
import type { TmuxPick } from "../types"
import { CloseIcon } from "./icons"

type Props = {
  picks: TmuxPick[]
  busy: boolean
  error: string
  onRemove: (file: string) => void
  onConfirm: () => void
  onClose: () => void
}

export function TmuxModal(props: Props) {
  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal tmux-modal" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="review-panel-title">open in nvim</span>
          <button class="modal-close" title="close" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>

        <div class="modal-body">
            <Show when={props.error}>
              <div class="review-error">{props.error}</div>
            </Show>

            <Show
              when={props.picks.length > 0}
              fallback={<p class="muted">no lines selected — click a line in the diff</p>}
            >
              <ul class="tmux-pick-list">
                <For each={props.picks}>
                  {(pick) => (
                    <li class="tmux-pick-row">
                      <span class="tmux-pick-file">{pick.file}</span>
                      <Show when={pick.line}>
                        <span class="tmux-pick-line">:{pick.line}</span>
                      </Show>
                      <button
                        class="tmux-pick-remove"
                        title="remove"
                        aria-label={`remove ${pick.file}`}
                        onClick={() => props.onRemove(pick.file)}
                      >
                        <CloseIcon />
                      </button>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </div>

          <div class="modal-foot">
            <button
              class="review-run"
              disabled={props.picks.length === 0 || props.busy}
              onClick={props.onConfirm}
            >
              {props.busy ? "creating..." : "create session"}
            </button>
          </div>
      </div>
    </div>
  )
}
