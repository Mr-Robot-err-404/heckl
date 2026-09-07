import type {
  AgentConfig,
  AgentConfigPage,
  DiffBlob,
  PR,
  PRDetail,
  Repo,
  Review,
  ReviewHistoryPage,
  ReviewerThread,
  TmuxLiveSession,
  TmuxPick,
  TmuxSession,
} from "./types"

const BASE = "/api"

export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

async function request(path: string, method: string, body?: unknown): Promise<Response> {
  const res = await fetch(BASE + path, {
    method,
    ...(body === undefined
      ? {}
      : { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
  })
  if (!res.ok) throw new Error(await errorMessage(res, path))
  return res
}

async function errorMessage(res: Response, path: string): Promise<string> {
  const text = await res.text().catch(() => "")
  if (!text) return `${res.status} ${path}`
  try {
    const parsed = JSON.parse(text)
    if (typeof parsed?.error === "string") return parsed.error
  } catch {}
  return text.trim()
}

async function get<T>(path: string): Promise<T> {
  return (await request(path, "GET")).json()
}

async function post<T>(path: string, body: unknown): Promise<T> {
  return (await request(path, "POST", body)).json()
}

async function put<T>(path: string, body: unknown): Promise<T> {
  return (await request(path, "PUT", body)).json()
}

async function del(path: string): Promise<void> {
  await request(path, "DELETE")
}

async function getText(path: string): Promise<string> {
  return (await request(path, "GET")).text()
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
    comments: (owner: string, repo: string, number: number) =>
      get<ReviewerThread[]>(`/prs/${owner}/${repo}/${number}/comments`),
  },
  diff: {
    get: (owner: string, repo: string, number: number) =>
      getText(`/diff/${owner}/${repo}/${number}`),
    blob: (owner: string, repo: string, number: number, path: string, prev?: string) => {
      const params = new URLSearchParams({ path })
      if (prev && prev !== path) params.set("prev", prev)
      return get<DiffBlob>(`/blob/${owner}/${repo}/${number}?${params}`)
    },
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
  theme: {
    get: () => get<{ theme: string; available: string[] }>("/theme"),
    set: (theme: string) => put<{ theme: string }>("/theme", { theme }),
  },
  agents: {
    list: () => get<string[]>("/agents"),
    config: () => get<AgentConfigPage>("/agents/config"),
    save: (configs: AgentConfig[]) => put<AgentConfigPage>("/agents/config", configs),
  },
  tmux: {
    get: (owner: string, repo: string, number: number) =>
      get<TmuxLiveSession | null>(`/tmux/${owner}/${repo}/${number}`),
    open: (owner: string, repo: string, number: number, files: TmuxPick[]) =>
      post<TmuxSession>(`/tmux/${owner}/${repo}/${number}`, { files }),
  },
  review: {
    start: (owner: string, repo: string, number: number, agents: string[]) =>
      post<Review>(`/review/${owner}/${repo}/${number}`, { agents }),
    rerunAgent: (owner: string, repo: string, number: number, agent: string) =>
      post<Review>(`/review/${owner}/${repo}/${number}/agent/${agent}`, {}),
    cancel: (owner: string, repo: string, number: number) =>
      del(`/review/${owner}/${repo}/${number}`),
    cancelAgent: (owner: string, repo: string, number: number, agent: string) =>
      del(`/review/${owner}/${repo}/${number}/agent/${agent}`),
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
