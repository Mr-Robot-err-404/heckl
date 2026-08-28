import { createSignal, Show } from "solid-js"
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query"
import { PRDetail } from "./components/PRDetail"
import { PRList } from "./components/PRList"
import { RepoSidebar } from "./components/RepoSidebar"
import type { PR, Repo } from "./types"

const client = new QueryClient()

export default function App() {
  const [selectedRepo, setSelectedRepo] = createSignal<Repo | null>(null)
  const [selectedPR, setSelectedPR] = createSignal<PR | null>(null)

  function handleSelectRepo(repo: Repo) {
    setSelectedRepo(repo)
    setSelectedPR(null)
  }

  return (
    <QueryClientProvider client={client}>
      <div class="layout">
        <RepoSidebar selected={selectedRepo()} onSelect={handleSelectRepo} />
        <Show when={selectedRepo()} keyed>
          {(repo) => (
            <PRList repo={repo} selected={selectedPR()} onSelect={setSelectedPR} />
          )}
        </Show>
        <main class="main">
          <Show
            when={selectedRepo() && selectedPR()}
            fallback={
              <div class="status">
                {selectedRepo() ? "select a PR" : "add or select a repo"}
              </div>
            }
          >
            <PRDetail repo={selectedRepo()!} pr={selectedPR()!} />
          </Show>
        </main>
      </div>
    </QueryClientProvider>
  )
}
