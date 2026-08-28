import { createSignal, For } from "solid-js"
import { useAddRepo, useRemoveRepo, useRepos } from "../queries"
import type { Repo } from "../types"

type Props = {
  selected: Repo | null
  onSelect: (repo: Repo) => void
}

export function RepoSidebar(props: Props) {
  const repos = useRepos()
  const addRepo = useAddRepo()
  const removeRepo = useRemoveRepo()
  const [input, setInput] = createSignal("")
  const [error, setError] = createSignal<string | null>(null)

  function handleAdd() {
    const parts = input().trim().split("/")
    if (parts.length !== 2 || !parts[0] || !parts[1]) {
      setError("format: owner/repo")
      return
    }
    addRepo.mutate({ owner: parts[0], name: parts[1] }, {
      onSuccess: () => { setInput(""); setError(null) },
      onError: (e) => setError(String(e)),
    })
  }

  function handleRemove(e: MouseEvent, repo: Repo) {
    e.stopPropagation()
    removeRepo.mutate({ owner: repo.Owner, name: repo.Name })
  }

  return (
    <aside class="sidebar">
      <div class="sidebar-header">repos</div>
      <ul class="repo-list">
        <For each={repos.data}>
          {(repo) => (
            <li
              class={`repo-item ${props.selected?.ID === repo.ID ? "active" : ""}`}
              onClick={() => props.onSelect(repo)}
            >
              <span class="repo-name">{repo.Owner}/{repo.Name}</span>
              <button class="remove-btn" onClick={(e) => handleRemove(e, repo)}>×</button>
            </li>
          )}
        </For>
      </ul>
      <div class="add-repo">
        <input
          placeholder="owner/repo"
          value={input()}
          onInput={(e) => setInput(e.currentTarget.value)}
          onKeyDown={(e) => e.key === "Enter" && handleAdd()}
        />
        {error() && <span class="error">{error()}</span>}
      </div>
    </aside>
  )
}
