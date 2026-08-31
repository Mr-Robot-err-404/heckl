import type { PR, PRDetail, Repo, Review } from "./types"

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

async function getText(path: string): Promise<string> {
  const res = await fetch(BASE + path)
  if (!res.ok) throw new Error(`${res.status} ${path}`)
  return res.text()
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
  diff: {
    get: (owner: string, repo: string, number: number) =>
      getText(`/diff/${owner}/${repo}/${number}`),
  },
  review: {
    start: (owner: string, repo: string, number: number) =>
      post<Review>(`/review/${owner}/${repo}/${number}`, {}),
    stream: (id: string, onReview: (review: Review) => void) => {
      const source = new EventSource(`${BASE}/review/live/${id}/stream`)
      const handle = (e: MessageEvent) => {
        const review = JSON.parse(e.data) as Review
        onReview(review)
        if (review.status === "done" || review.status === "error") source.close()
      }
      source.addEventListener("snapshot", handle)
      source.addEventListener("review", handle)
      return () => source.close()
    },
  },
}
