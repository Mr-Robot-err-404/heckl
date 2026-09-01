import { useParams, useNavigate, useSearch } from "@tanstack/solid-router"
import { PRDetail } from "../components/PRDetail"
import type { Tab } from "../types"

export function PRDetailPage() {
  const params = useParams({ from: "/$owner/$repo/$pr" })
  const search = useSearch({ from: "/$owner/$repo/$pr" })
  const navigate = useNavigate()

  const setTab = (tab: Tab) =>
    navigate({
      to: "/$owner/$repo/$pr",
      params: params(),
      search: { tab },
      replace: true,
    })

  return (
    <PRDetail
      owner={params().owner}
      repo={params().repo}
      prNumber={parseInt(params().pr, 10)}
      tab={search().tab}
      onTabChange={setTab}
      onBack={() =>
        navigate({
          to: "/$owner/$repo",
          params: { owner: params().owner, repo: params().repo },
        })
      }
    />
  )
}
