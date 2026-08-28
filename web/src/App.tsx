import { createSignal, Show } from "solid-js"
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query"
import { PRDetail } from "./components/PRDetail"
import { PRList } from "./components/PRList"
import { TopBar } from "./components/TopBar"
import type { PR, Repo } from "./types"

const client = new QueryClient()

export default function App() {
  const [selectedRepo, setSelectedRepo] = createSignal<Repo | null>(null)
  const [selectedPR, setSelectedPR] = createSignal<PR | null>(null)

  function handleSelectRepo(repo: Repo | null) {
    setSelectedRepo(repo)
    setSelectedPR(null)
  }

  return (
    <QueryClientProvider client={client}>
      <div class="layout">
        <TopBar selected={selectedRepo()} onSelect={handleSelectRepo} />
        <div class="content">
          <Show when={selectedRepo()} keyed>
            {(repo) => (
              <Show
                when={selectedPR()}
                keyed
                fallback={
                  <PRList repo={repo} onSelect={setSelectedPR} />
                }
              >
                {(pr) => (
                  <PRDetail
                    repo={repo}
                    pr={pr}
                    onBack={() => setSelectedPR(null)}
                  />
                )}
              </Show>
            )}
          </Show>
          <Show when={!selectedRepo()}>
            <div class="empty">select a repo to begin</div>
          </Show>
        </div>
      </div>
    </QueryClientProvider>
  )
}
