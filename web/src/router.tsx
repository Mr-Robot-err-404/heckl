import { createRouter, createRoute, createRootRoute, Outlet } from "@tanstack/solid-router"
import { PRListPage } from "./routes/PRListPage"
import { PRDetailPage } from "./routes/PRDetailPage"
import { TopBar } from "./components/TopBar"
import type { Tab } from "./types"

const rootRoute = createRootRoute({
  component: () => (
    <>
      <TopBar />
      <div class="content">
        <Outlet />
      </div>
    </>
  ),
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: () => <div class="empty">select a repo to begin</div>,
})

const repoRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/$owner/$repo",
  component: PRListPage,
})

const prRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/$owner/$repo/$pr",
  validateSearch: (search: Record<string, unknown>): { tab: Tab } => ({
    tab: search.tab === "review" ? "review" : "description",
  }),
  component: PRDetailPage,
})

const routeTree = rootRoute.addChildren([indexRoute, repoRoute, prRoute])

export const router = createRouter({ routeTree })

declare module "@tanstack/solid-router" {
  interface Register {
    router: typeof router
  }
}
