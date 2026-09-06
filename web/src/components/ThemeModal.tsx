import { For } from "solid-js"
import { setTheme, theme, themes } from "../theme"
import { CloseIcon } from "./icons"

const groups = [
  { label: "dark", themes: themes.filter((t) => t.dark) },
  { label: "light", themes: themes.filter((t) => !t.dark) },
]

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
          <For each={groups}>
            {(group) => (
              <div class="theme-group">
                <span class="theme-group-label">{group.label}</span>
                <For each={group.themes}>
                  {(t) => (
                    <button
                      class={`theme-option ${theme().id === t.id ? "active" : ""}`}
                      onClick={() => setTheme(t.id)}
                    >
                      {t.label}
                    </button>
                  )}
                </For>
              </div>
            )}
          </For>
        </div>
      </div>
    </div>
  )
}
