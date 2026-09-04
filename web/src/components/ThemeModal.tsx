import { For } from "solid-js"
import { setTheme, theme, themes } from "../theme"
import { CloseIcon } from "./icons"

export function ThemeModal(props: { onClose: () => void }) {
  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal modal-narrow" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="review-panel-title">theme</span>
          <button class="modal-close" title="close" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>

        <div class="modal-body">
          <For each={themes}>
            {(t) => (
              <button
                class={`theme-option ${theme().id === t.id ? "active" : ""}`}
                onClick={() => setTheme(t.id)}
              >
                <span class="theme-option-name">{t.label}</span>
                <span class="muted">{t.dark ? "dark" : "light"}</span>
              </button>
            )}
          </For>
        </div>
      </div>
    </div>
  )
}
