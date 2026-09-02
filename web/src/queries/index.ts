import { createQuery, createMutation, useQueryClient } from "@tanstack/solid-query"
import { api } from "../api"

const keepPrevious = <T,>(prev: T | undefined) => prev

type Settling<T> = { isSuccess: boolean; data: T | undefined }

export function resolved<T>(query: Settling<T>): () => T | undefined {
  return () => (query.isSuccess ? query.data : undefined)
}

export function useOrgs() {
  return createQuery(() => ({
    queryKey: ["orgs"],
    queryFn: api.orgs.list,
    staleTime: Infinity,
  }))
}

export function useRepos() {
  return createQuery(() => ({
    queryKey: ["repos"],
    queryFn: api.repos.list,
    staleTime: Infinity,
  }))
}

export function useReposByOwner(owner: () => string) {
  return createQuery(() => ({
    queryKey: ["repos", owner()],
    queryFn: () => api.repos.listByOwner(owner()),
    enabled: !!owner(),
    staleTime: Infinity,
  }))
}

export function useAddRepo() {
  const client = useQueryClient()
  return createMutation(() => ({
    mutationFn: ({ owner, name }: { owner: string; name: string }) =>
      api.repos.add(owner, name),
    onSuccess: () => client.invalidateQueries({ queryKey: ["repos"] }),
  }))
}

export function useRemoveRepo() {
  const client = useQueryClient()
  return createMutation(() => ({
    mutationFn: ({ owner, name }: { owner: string; name: string }) =>
      api.repos.remove(owner, name),
    onSuccess: () => client.invalidateQueries({ queryKey: ["repos"] }),
  }))
}

export const historyQueryKey = ["reviews", "history"]

export const historyPageSize = 20

export function historyOptions(page: number) {
  return {
    queryKey: [...historyQueryKey, page],
    queryFn: () => api.reviews.history({ limit: historyPageSize, offset: page * historyPageSize }),
  }
}

export function repoHistoryOptions(owner: string, repo: string, limit: number) {
  return {
    queryKey: [...historyQueryKey, owner, repo, limit],
    queryFn: () => api.reviews.history({ owner, repo, limit, offset: 0 }),
  }
}

export function prsOptions(owner: string, repo: string) {
  return {
    queryKey: ["prs", owner, repo],
    queryFn: () => api.prs.list(owner, repo),
  }
}

export function prDetailOptions(owner: string, repo: string, number: number) {
  return {
    queryKey: ["pr", owner, repo, number],
    queryFn: () => api.prs.get(owner, repo, number),
  }
}

export function diffOptions(owner: string, repo: string, number: number) {
  return {
    queryKey: ["diff", owner, repo, number],
    queryFn: () => api.diff.get(owner, repo, number),
    staleTime: Infinity,
  }
}

export function useReviewHistory(page: () => number) {
  return createQuery(() => ({
    ...historyOptions(page()),
    placeholderData: keepPrevious,
  }))
}

export function useRepoReviewHistory(
  owner: () => string,
  repo: () => string,
  limit: () => number,
) {
  return createQuery(() => ({
    ...repoHistoryOptions(owner(), repo(), limit()),
    enabled: !!owner() && !!repo(),
    placeholderData: keepPrevious,
  }))
}

export function usePRs(owner: () => string, repo: () => string) {
  return createQuery(() => ({
    ...prsOptions(owner(), repo()),
    enabled: !!owner() && !!repo(),
  }))
}

export function usePRDetail(
  owner: () => string,
  repo: () => string,
  number: () => number | null,
) {
  return createQuery(() => ({
    ...prDetailOptions(owner(), repo(), number() ?? 0),
    enabled: !!owner() && !!repo() && number() != null,
  }))
}

export function useDiff(
  owner: () => string,
  repo: () => string,
  number: () => number | null,
) {
  return createQuery(() => ({
    ...diffOptions(owner(), repo(), number() ?? 0),
    enabled: !!owner() && !!repo() && number() != null,
  }))
}

export function usePrefetch() {
  const client = useQueryClient()
  const run = (options: Parameters<typeof client.prefetchQuery>[0]) => {
    void client.prefetchQuery(options)
  }
  return {
    prs: (owner: string, repo: string) => run(prsOptions(owner, repo)),
    prDetail: (owner: string, repo: string, number: number) =>
      run(prDetailOptions(owner, repo, number)),
    diff: (owner: string, repo: string, number: number) =>
      run(diffOptions(owner, repo, number)),
    historyPage: (page: number) => run(historyOptions(page)),
    repoHistory: (owner: string, repo: string, limit: number) =>
      run(repoHistoryOptions(owner, repo, limit)),
    pr: (owner: string, repo: string, number: number) => {
      run(prDetailOptions(owner, repo, number))
      run(diffOptions(owner, repo, number))
    },
  }
}
