import { createSignal, For, Show } from "solid-js"
import { useNavigate, useParams } from "@tanstack/solid-router"
import { useAddRepo, useRepos, usePRDetail } from "../queries"

export function TopBar() {
  const repos = useRepos()
  const addRepo = useAddRepo()
  const navigate = useNavigate()
  const [input, setInput] = createSignal("")
  const [error, setError] = createSignal<string | null>(null)
  const [adding, setAdding] = createSignal(false)

  const params = useParams({ strict: false })
  const selectedKey = () => {
    const p = params()
    return p.owner && p.repo ? `${p.owner}/${p.repo}` : ""
  }
  const prNumber = () => {
    const n = parseInt(params().pr ?? "", 10)
    return isNaN(n) ? null : n
  }

  const prDetail = usePRDetail(
    () => params().owner ?? "",
    () => params().repo ?? "",
    () => prNumber(),
  )

  function handleChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value
    if (val === "__add__") { setAdding(true); return }
    const [owner, repo] = val.split("/")
    navigate({ to: "/$owner/$repo", params: { owner, repo } })
  }

  function handleAdd() {
    const parts = input().trim().split("/")
    if (parts.length !== 2 || !parts[0] || !parts[1]) {
      setError("owner/repo")
      return
    }
    addRepo.mutate({ owner: parts[0], name: parts[1] }, {
      onSuccess: (repo) => {
        setInput(""); setError(null); setAdding(false)
        navigate({ to: "/$owner/$repo", params: { owner: repo.Owner, repo: repo.Name } })
      },
      onError: (e) => setError(String(e)),
    })
  }

  return (
    <header class="topbar">
      <div class="topbar-left">
        <span class="topbar-brand">pr review</span>
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
              <button class="topbar-btn" onClick={() => { setAdding(false); setInput(""); setError(null) }}>cancel</button>
              <Show when={error()}><span class="input-error">{error()}</span></Show>
            </div>
          }
        >
          <select class="repo-select" value={selectedKey()} onChange={handleChange}>
            <option value="" disabled>select repo</option>
            <For each={repos.data}>
              {(repo) => (
                <option value={`${repo.Owner}/${repo.Name}`}>{repo.Owner}/{repo.Name}</option>
              )}
            </For>
            <option value="__add__">+ add repo</option>
          </select>
        </Show>
      </div>
      <Show when={prDetail.data} keyed>
        {(d) => (
          <span class="topbar-pr-title">{d.pr.Title}</span>
        )}
      </Show>
      <Show when={prDetail.data} keyed>
        {(d) => {
          const additions = d.files.reduce((n, f) => n + f.Additions, 0)
          const deletions = d.files.reduce((n, f) => n + f.Deletions, 0)
          return (
            <div class="topbar-right">
              <span class="additions">+{additions}</span>
              <span class="deletions">-{deletions}</span>
              <span class="muted">{d.files.length} files</span>
              <span class="muted">{d.pr.Author}</span>
            </div>
          )
        }}
      </Show>
    </header>
  )
}
