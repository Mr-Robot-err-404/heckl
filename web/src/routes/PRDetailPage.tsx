import { useParams, useNavigate } from "@tanstack/solid-router"
import { PRDetail } from "../components/PRDetail"

export function PRDetailPage() {
  const params = useParams({ from: "/$owner/$repo/$pr" })
  const navigate = useNavigate()

  return (
    <PRDetail
      owner={params().owner}
      repo={params().repo}
      prNumber={parseInt(params().pr, 10)}
      onBack={() => navigate({ to: "/$owner/$repo", params: { owner: params().owner, repo: params().repo } })}
    />
  )
}
