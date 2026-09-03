import { For, Show } from "solid-js"
import { agentLabel, type ReviewState } from "../review"

export function AgentPicker(props: { state: ReviewState }) {
  const s = () => props.state

  return (
    <Show when={s().available().length > 1}>
      <div class="agent-picker">
        <For each={s().available()}>
          {(name) => (
            <button
              class={`agent-chip ${s().isSelected(name) ? "on" : "off"}`}
              disabled={s().busy()}
              onClick={() => s().toggleAgent(name)}
              title={name}
            >
              {agentLabel(name)}
            </button>
          )}
        </For>
      </div>
    </Show>
  )
}
