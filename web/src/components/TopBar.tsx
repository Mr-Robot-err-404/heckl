import { createSignal, For, Show } from "solid-js"
import { useAddRepo, useRepos } from "../queries"
import type { Repo } from "../types"

type Props = {
  selected: Repo | null
  onSelect: (repo: Repo | null) => void
}

export function TopBar(props: Props) {
  const repos = useRepos()
  const addRepo = useAddRepo()
  const [input, setInput] = createSignal("")
  const [error, setError] = createSignal<string | null>(null)
  const [adding, setAdding] = createSignal(false)

  function handleAdd() {
    const parts = input().trim().split("/")
    if (parts.length !== 2 || !parts[0] || !parts[1]) {
      setError("owner/repo")
      return
    }
    addRepo.mutate({ owner: parts[0], name: parts[1] }, {
      onSuccess: (repo) => {
        setInput("")
        setError(null)
        setAdding(false)
        props.onSelect(repo)
      },
      onError: (e) => setError(String(e)),
    })
  }

  function handleChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value
    if (val === "__add__") {
      setAdding(true)
      return
    }
    const repo = repos.data?.find((r) => String(r.ID) === val) ?? null
    props.onSelect(repo)
  }

  return (
    <header class="topbar">
      <span class="topbar-brand">pr review</span>
      <div class="topbar-controls">
        <Show
          when={!adding()}
          fallback={
            <div class="repo-input-wrap">
              <input
                class="repo-input"
                placeholder="owner/repo"
                value={input()}
                autofocus
                onInput={(e) => setInput(e.currentTarget.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleAdd()
                  if (e.key === "Escape") { setAdding(false); setInput(""); setError(null) }
                }}
              />
              <button class="topbar-btn" onClick={handleAdd}>add</button>
              <button class="topbar-btn muted" onClick={() => { setAdding(false); setInput(""); setError(null) }}>cancel</button>
              <Show when={error()}>
                <span class="input-error">{error()}</span>
              </Show>
            </div>
          }
        >
          <select class="repo-select" onChange={handleChange} value={props.selected ? String(props.selected.ID) : ""}>
            <option value="" disabled>select repo</option>
            <For each={repos.data}>
              {(repo) => (
                <option value={String(repo.ID)}>{repo.Owner}/{repo.Name}</option>
              )}
            </For>
            <option value="__add__">+ add repo</option>
          </select>

        </Show>
      </div>
    </header>
  )
}
