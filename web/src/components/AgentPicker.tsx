import { For, Show, type JSX } from "solid-js"
import { agentLabel, type ReviewState } from "../review"
import { CheckIcon } from "./icons"

export function AgentPicker(props: { state: ReviewState; meta?: JSX.Element }) {
  const s = () => props.state
  const showList = () => s().available().length > 1

  return (
    <Show when={showList() || props.meta}>
      <div class="agent-picker">
        <div class="agent-picker-head">
          <Show when={showList()}>
            <span class="agent-picker-label">agents</span>
          </Show>
          <Show when={props.meta}>
            <span class="agent-picker-meta">{props.meta}</span>
          </Show>
        </div>
        <Show when={showList()}>
          <div class="agent-picker-list">
            <For each={s().available()}>
              {(name) => (
                <button
                  class={`agent-option ${s().isSelected(name) ? "on" : "off"}`}
                  aria-pressed={s().isSelected(name)}
                  disabled={s().busy()}
                  onClick={() => s().toggleAgent(name)}
                  title={name}
                >
                  <span class="agent-option-mark">
                    <Show when={s().isSelected(name)}>
                      <CheckIcon />
                    </Show>
                  </span>
                  <span class="agent-option-name">{agentLabel(name)}</span>
                </button>
              )}
            </For>
          </div>
        </Show>
      </div>
    </Show>
  )
}
