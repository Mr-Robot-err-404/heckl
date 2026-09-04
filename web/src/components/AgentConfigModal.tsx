import { createEffect, createSignal, For, Show } from "solid-js"
import { createStore, unwrap } from "solid-js/store"
import { agentLabel } from "../review"
import { resolved, useAgentConfig, useSaveAgentConfig } from "../queries"
import type { AgentConfig, ModelOption } from "../types"
import { CloseIcon } from "./icons"

export function AgentConfigModal(props: { onClose: () => void }) {
  const config = useAgentConfig()
  const save = useSaveAgentConfig()
  const page = resolved(config)

  const [draft, setDraft] = createStore<{ agents: AgentConfig[] }>({ agents: [] })
  const [expanded, setExpanded] = createSignal<string | null>(null)
  const [error, setError] = createSignal("")

  createEffect(() => {
    const loaded = page()
    if (loaded) setDraft("agents", loaded.agents.map((a) => ({ ...a })))
  })

  const models = (): ModelOption[] => page()?.models ?? []

  const edit = (index: number, patch: Partial<AgentConfig>) =>
    setDraft("agents", index, patch)

  const toggle = (name: string) => setExpanded(expanded() === name ? null : name)

  function submit() {
    setError("")
    save.mutate(unwrap(draft.agents), {
      onSuccess: () => props.onClose(),
      onError: (e) => setError(String(e)),
    })
  }

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="review-panel-title">agent config</span>
          <button class="modal-close" title="close" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>

        <Show when={config.isError}>
          <div class="review-error">could not reach opencode — {String(config.error)}</div>
        </Show>

        <Show when={draft.agents.length > 0} fallback={<p class="review-idle">loading...</p>}>
          <div class="modal-body">
            <For each={draft.agents}>
              {(agent, index) => (
                <section class="agent-config">
                  <div class="agent-config-head">
                    <span class="agent-config-name">{agentLabel(agent.name)}</span>
                    <select
                      class="repo-select"
                      value={agent.model}
                      onChange={(e) => edit(index(), { model: e.currentTarget.value })}
                    >
                      <option value="">default — {agent.defaultModel || "unset"}</option>
                      <For each={models()}>
                        {(model) => (
                          <option value={model.ref}>
                            {model.provider} · {model.name}
                          </option>
                        )}
                      </For>
                    </select>
                  </div>
                  <button class="agent-config-toggle" onClick={() => toggle(agent.name)}>
                    <span class="reviewer-caret">{expanded() === agent.name ? "▾" : "▸"}</span>
                    extra prompt
                    <Show when={agent.prompt.trim()}>
                      <span class="muted">set</span>
                    </Show>
                  </button>
                  <Show when={expanded() === agent.name}>
                    <textarea
                      class="agent-config-prompt"
                      rows="8"
                      placeholder="appended as the system prompt — blank for none"
                      value={agent.prompt}
                      onInput={(e) => edit(index(), { prompt: e.currentTarget.value })}
                    />
                  </Show>
                </section>
              )}
            </For>
          </div>
        </Show>

        <Show when={error()}>
          <div class="review-error">{error()}</div>
        </Show>

        <div class="modal-foot">
          <button class="topbar-btn" disabled={save.isPending} onClick={submit}>
            {save.isPending ? "saving..." : "save"}
          </button>
        </div>
      </div>
    </div>
  )
}
