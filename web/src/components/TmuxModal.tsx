import { For, Show } from "solid-js"
import type { TmuxPick, TmuxSession } from "../types"
import { CloseIcon } from "./icons"

type Props = {
  picks: TmuxPick[]
  busy: boolean
  error: string
  result: TmuxSession | null
  onRemove: (file: string) => void
  onConfirm: () => void
  onClose: () => void
}

function attachCommand(session: string) {
  const attach = `tmux attach -t '${session}'`
  const host = location.hostname
  if (host === "localhost" || host === "127.0.0.1") return attach
  return `ssh -t ${host} "${attach}"`
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

        <Show
          when={!props.result}
          fallback={
            <div class="modal-body">
              <p class="muted">{props.result!.opened.length} window(s) open</p>
              <code class="tmux-attach">{attachCommand(props.result!.session)}</code>
              <Show when={props.result!.skipped?.length}>
                <p class="tmux-skipped">
                  not in this commit: {props.result!.skipped!.join(", ")}
                </p>
              </Show>
            </div>
          }
        >
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
        </Show>
      </div>
    </div>
  )
}
