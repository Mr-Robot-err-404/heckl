import type { PR, PRDetail, Repo, Review, ReviewHistoryPage } from "./types"

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
  reviews: {
    history: (opts: { limit: number; offset: number; owner?: string; repo?: string }) => {
      const params = new URLSearchParams({
        limit: String(opts.limit),
        offset: String(opts.offset),
      })
      if (opts.owner && opts.repo) {
        params.set("owner", opts.owner)
        params.set("repo", opts.repo)
      }
      return get<ReviewHistoryPage>(`/reviews/history?${params}`)
    },
    stream: (on: {
      snapshot: (active: Review[]) => void
      review: (review: Review) => void
      connected: (connected: boolean) => void
    }) => {
      const source = new EventSource(`${BASE}/reviews/stream`)
      source.addEventListener("snapshot", (e) => {
        on.connected(true)
        on.snapshot((JSON.parse(e.data) as Review[] | null) ?? [])
      })
      source.addEventListener("review", (e) => {
        on.connected(true)
        on.review(JSON.parse(e.data) as Review)
      })
      source.addEventListener("error", () => on.connected(false))
      return () => source.close()
    },
  },
  review: {
    start: (owner: string, repo: string, number: number) =>
      post<Review>(`/review/${owner}/${repo}/${number}`, {}),
    stream: (
      owner: string,
      repo: string,
      number: number,
      on: {
        state: (review: Review | null) => void
        connected: (connected: boolean) => void
      },
    ) => {
      const source = new EventSource(`${BASE}/review/${owner}/${repo}/${number}/stream`)
      const handle = (e: MessageEvent) => {
        on.connected(true)
        on.state(JSON.parse(e.data) as Review | null)
      }
      source.addEventListener("snapshot", handle)
      source.addEventListener("review", handle)
      source.addEventListener("error", () => on.connected(false))
      return () => source.close()
    },
  },
}
