import { createSignal, For, Show } from "solid-js"
import { copyText } from "../clipboard"
import type { TmuxLiveSession, TmuxPick } from "../types"
import { CheckIcon, ClipboardIcon, CloseIcon } from "./icons"

type Props = {
  picks: TmuxPick[]
  live: TmuxLiveSession | null
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
          <span class="review-panel-title">tmux</span>
          <button class="modal-close" title="close" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>

        <div class="modal-body">
          <Show when={props.error}>
            <div class="review-error">{props.error}</div>
          </Show>

          <Show when={props.live} keyed>
            {(live) => (
              <section class="tmux-section">
                <h4 class="tmux-section-title">
                  active session
                  <span class="muted"> · {live.windows} window(s)</span>
                </h4>
                <CommandBlock command={live.attach} />
              </section>
            )}
          </Show>

          <section class="tmux-section">
            <h4 class="tmux-section-title">
              {props.live ? "replace with a new selection" : "selected lines"}
            </h4>
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
          </section>
        </div>

        <div class="modal-foot">
          <button
            class="review-run"
            disabled={props.picks.length === 0 || props.busy}
            onClick={props.onConfirm}
          >
            {props.busy ? "creating..." : props.live ? "replace session" : "create session"}
          </button>
        </div>
      </div>
    </div>
  )
}

function CommandBlock(props: { command: string }) {
  const [copied, setCopied] = createSignal(false)

  let timer: ReturnType<typeof setTimeout> | undefined
  const copy = async () => {
    const ok = await copyText(props.command)
    if (!ok) return
    setCopied(true)
    clearTimeout(timer)
    timer = setTimeout(() => setCopied(false), 1200)
  }

  return (
    <div class="command-block">
      <code>{props.command}</code>
      <button
        class={`command-copy ${copied() ? "copied" : ""}`}
        title="copy"
        aria-label="copy command"
        onClick={copy}
      >
        {copied() ? <CheckIcon /> : <ClipboardIcon />}
      </button>
    </div>
  )
}
