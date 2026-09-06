import { For, Show } from "solid-js"
import { agentLabel, type ReviewState } from "../review"
import { CheckIcon } from "./icons"

export function AgentPicker(props: { state: ReviewState }) {
  const s = () => props.state

  return (
    <Show when={s().available().length > 1}>
      <div class="agent-picker">
        <span class="agent-picker-label">agents</span>
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
      </div>
    </Show>
  )
}
