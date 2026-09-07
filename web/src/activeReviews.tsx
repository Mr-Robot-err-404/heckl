import { createContext, createSignal, onCleanup, useContext, type JSX } from "solid-js"
import { useQueryClient } from "@tanstack/solid-query"
import { api } from "./api"
import { historyQueryKey } from "./queries"
import type { Review } from "./types"

export function reviewKey(owner: string, repo: string, prNumber: number) {
  return `${owner}/${repo}#${prNumber}`
}

function isFinished(review: Review) {
  return review.status === "done" || review.status === "error" || review.status === "cancelled"
}

type ActiveReviews = {
  active: () => Review[]
  count: () => number
  connected: () => boolean
}

const ActiveReviewsContext = createContext<ActiveReviews>()

export function ActiveReviewsProvider(props: { children: JSX.Element }) {
  const client = useQueryClient()
  const [byKey, setByKey] = createSignal<Record<string, Review>>({})
  const [connected, setConnected] = createSignal(false)

  const close = api.reviews.stream({
    connected: setConnected,
    snapshot: (reviews) => {
      const next: Record<string, Review> = {}
      for (const review of reviews) {
        next[reviewKey(review.owner, review.repo, review.prNumber)] = review
      }
      setByKey(next)
    },
    review: (review) => {
      const key = reviewKey(review.owner, review.repo, review.prNumber)
      setByKey((prev) => {
        const next = { ...prev }
        if (isFinished(review)) delete next[key]
        else next[key] = review
        return next
      })
      if (isFinished(review)) client.invalidateQueries({ queryKey: historyQueryKey })
    },
  })
  onCleanup(close)

  const active = () =>
    Object.values(byKey()).sort((a, b) => Date.parse(b.startedAt) - Date.parse(a.startedAt))

  const value: ActiveReviews = {
    active,
    count: () => active().length,
    connected,
  }

  return (
    <ActiveReviewsContext.Provider value={value}>
      {props.children}
    </ActiveReviewsContext.Provider>
  )
}

export function useActiveReviews(): ActiveReviews {
  const ctx = useContext(ActiveReviewsContext)
  if (!ctx) throw new Error("useActiveReviews outside ActiveReviewsProvider")
  return ctx
}
