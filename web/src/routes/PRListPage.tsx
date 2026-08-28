import { useParams } from "@tanstack/solid-router"
import { PRList } from "../components/PRList"

export function PRListPage() {
  const params = useParams({ from: "/$owner/$repo" })
  return <PRList owner={params().owner} repo={params().repo} />
}
