import { createEffect, createSignal, For, Show } from "solid-js"
import { useNavigate, useParams } from "@tanstack/solid-router"
import { useAddRepo, useOrgs, useRepos, usePRDetail } from "../queries"

export function TopBar() {
  const orgs = useOrgs()
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

  let selectRef: HTMLSelectElement | undefined

  const reposByOrg = () => {
    const orgList = orgs.data ?? []
    const repoList = repos.data ?? []
    return orgList.map((org) => ({
      org,
      repos: repoList.filter((r) => r.Owner === org),
    }))
  }

  const orphan = () => {
    const key = selectedKey()
    if (!key) return null
    const known = (repos.data ?? []).some((r) => `${r.Owner}/${r.Name}` === key)
    return known ? null : key
  }

  createEffect(() => {
    const key = selectedKey()
    reposByOrg()
    orphan()
    if (selectRef) selectRef.value = key
  })

  function handleRepoChange(e: Event) {
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
          <select ref={selectRef} class="repo-select" value={selectedKey()} onChange={handleRepoChange}>
            <option value="" disabled>select repo</option>
            <Show when={orphan()}>
              {(key) => <option value={key()}>{key()}</option>}
            </Show>
            <For each={reposByOrg()}>
              {(group) => (
                <optgroup label={group.org}>
                  <For each={group.repos}>
                    {(repo) => (
                      <option value={`${repo.Owner}/${repo.Name}`}>{repo.Name}</option>
                    )}
                  </For>
                </optgroup>
              )}
            </For>
            <option value="__add__">+ add repo</option>
          </select>
        </Show>
      </div>
      <Show when={prDetail.data} keyed>
        {(d) => <span class="topbar-pr-title">{d.pr.Title}</span>}
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
