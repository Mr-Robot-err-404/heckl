import { createQuery, createMutation, useQueryClient } from "@tanstack/solid-query"
import { api } from "../api"

export function useOrgs() {
  return createQuery(() => ({
    queryKey: ["orgs"],
    queryFn: api.orgs.list,
  }))
}

export function useRepos() {
  return createQuery(() => ({
    queryKey: ["repos"],
    queryFn: api.repos.list,
  }))
}

export function useReposByOwner(owner: () => string) {
  return createQuery(() => ({
    queryKey: ["repos", owner()],
    queryFn: () => api.repos.listByOwner(owner()),
    enabled: !!owner(),
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
