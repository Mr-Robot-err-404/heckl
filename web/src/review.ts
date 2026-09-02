import { createEffect, createSignal, onCleanup, type Accessor } from "solid-js"
import { api } from "./api"
import type { RankedConcern, Review } from "./types"

export type ReviewState = {
  review: Accessor<Review | null>
  synced: Accessor<boolean>
  connected: Accessor<boolean>
  busy: Accessor<boolean>
  canRun: Accessor<boolean>
  error: Accessor<string>
  now: Accessor<number>
  elapsed: Accessor<number>
  concerns: Accessor<RankedConcern[]>
  start: (agents?: string[]) => Promise<void>
}

const severityRank: Record<string, number> = { high: 0, medium: 1, low: 2 }

export const agentOrder = ["pr-reviewer", "pr-skeptic"]

export const agentLabels: Record<string, string> = {
  "pr-reviewer": "core review",
  "pr-skeptic": "is this necessary?",
}

export function agentLabel(name: string) {
  return agentLabels[name] ?? name
}

function agentRank(name: string) {
  const i = agentOrder.indexOf(name)
  return i === -1 ? agentOrder.length : i
}

export function createReview(
  owner: Accessor<string>,
  repo: Accessor<string>,
  prNumber: Accessor<number>,
): ReviewState {
  const [review, setReview] = createSignal<Review | null>(null)
  const [synced, setSynced] = createSignal(false)
  const [connected, setConnected] = createSignal(false)
  const [starting, setStarting] = createSignal(false)
  const [error, setError] = createSignal("")
  const [now, setNow] = createSignal(Date.now())

  createEffect(() => {
    const o = owner()
    const r = repo()
    const n = prNumber()
    setReview(null)
    setSynced(false)
    setConnected(false)
    setStarting(false)
    setError("")
    const close = api.review.stream(o, r, n, {
      state: (next) => {
        if (next) setStarting(false)
        setReview(next)
        setSynced(true)
      },
      connected: setConnected,
    })
    onCleanup(close)
  })

  const inFlight = () => {
    const status = review()?.status
    return status === "pending" || status === "running"
  }
  const busy = () => starting() || inFlight()
  const canRun = () => synced() && connected() && !busy()

  createEffect(() => {
    if (!busy()) return
    const timer = setInterval(() => setNow(Date.now()), 200)
    onCleanup(() => clearInterval(timer))
  })

  const elapsed = () => {
    const r = review()
    if (!r) return 0
    const end = r.endedAt ? Date.parse(r.endedAt) : now()
    return end - Date.parse(r.startedAt)
  }

  const concerns = (): RankedConcern[] =>
    [...(review()?.concerns ?? [])]
      .sort(
        (a, b) =>
          agentRank(a.agent) - agentRank(b.agent) ||
          (severityRank[a.severity] ?? 3) - (severityRank[b.severity] ?? 3),
      )
      .map((c, i) => ({ ...c, rank: i + 1 }))

  const start = async (agents: string[] = [...agentOrder]) => {
    setError("")
    setStarting(true)
    try {
      await api.review.start(owner(), repo(), prNumber(), agents)
    } catch (e) {
      setStarting(false)
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return { review, synced, connected, busy, canRun, error, now, elapsed, concerns, start }
}

const opencodePort = "4420"

export function opencodeUrl(path: string | undefined) {
  if (!path) return ""
  return `${window.location.protocol}//${window.location.hostname}:${opencodePort}${path}`
}

export function formatMs(ms: number) {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export function relativeTime(iso: string) {
  const then = Date.parse(iso)
  if (isNaN(then)) return ""
  const seconds = Math.max(0, (Date.now() - then) / 1000)
  if (seconds < 60) return "just now"
  const units: [number, string][] = [
    [60, "m"],
    [3600, "h"],
    [86400, "d"],
  ]
  for (let i = units.length - 1; i >= 0; i--) {
    const [size, suffix] = units[i]
    if (seconds >= size) return `${Math.floor(seconds / size)}${suffix} ago`
  }
  return "just now"
}

export function fileName(path: string) {
  const i = path.lastIndexOf("/")
  return i === -1 ? path : path.slice(i + 1)
}
