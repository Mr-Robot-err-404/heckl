import type { PR, PRDetail, Repo } from "./types"

const BASE = "/api"

async function get<T>(path: string): Promise<T> {
  const res = await fetch(BASE + path)
  if (!res.ok) throw new Error(`${res.status} ${path}`)
  return res.json()
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`${res.status} ${path}`)
  return res.json()
}

async function del(path: string): Promise<void> {
  const res = await fetch(BASE + path, { method: "DELETE" })
  if (!res.ok) throw new Error(`${res.status} ${path}`)
}

export const api = {
  orgs: {
    list: () => get<string[]>("/orgs"),
  },
  repos: {
    list: () => get<Repo[]>("/repos"),
    listByOwner: (owner: string) => get<Repo[]>(`/repos/${owner}`),
    add: (owner: string, name: string) => post<Repo>("/repos", { owner, name }),
    remove: (owner: string, name: string) => del(`/repos/${owner}/${name}`),
  },
  prs: {
    list: (owner: string, repo: string) => get<PR[]>(`/prs/${owner}/${repo}`),
    get: (owner: string, repo: string, number: number) =>
      get<PRDetail>(`/prs/${owner}/${repo}/${number}`),
  },
}
