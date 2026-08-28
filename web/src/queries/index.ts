import { createQuery, createMutation, useQueryClient } from "@tanstack/solid-query"
import { api } from "../api"

export function useRepos() {
  return createQuery(() => ({
    queryKey: ["repos"],
    queryFn: api.repos.list,
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

export function usePRs(owner: () => string, repo: () => string) {
  return createQuery(() => ({
    queryKey: ["prs", owner(), repo()],
    queryFn: () => api.prs.list(owner(), repo()),
    enabled: !!owner() && !!repo(),
  }))
}

export function usePRDetail(
  owner: () => string,
  repo: () => string,
  number: () => number | null,
) {
  return createQuery(() => ({
    queryKey: ["pr", owner(), repo(), number()],
    queryFn: () => api.prs.get(owner(), repo(), number()!),
    enabled: !!owner() && !!repo() && number() != null,
  }))
}
