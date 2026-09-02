import { Suspense } from "solid-js"
import { createRouter, createRoute, createRootRoute, Outlet } from "@tanstack/solid-router"
import { PRListPage } from "./routes/PRListPage"
import { PRDetailPage } from "./routes/PRDetailPage"
import { DashboardPage } from "./routes/DashboardPage"
import { TopBar } from "./components/TopBar"
import { ActiveReviewsProvider } from "./activeReviews"
import type { Tab } from "./types"

const rootRoute = createRootRoute({
  component: () => (
    <ActiveReviewsProvider>
      <div class="layout">
        <TopBar />
        <div class="content">
          <Suspense>
            <Outlet />
          </Suspense>
        </div>
      </div>
    </ActiveReviewsProvider>
  ),
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  validateSearch: (search: Record<string, unknown>): { page: number } => {
    const page = Number(search.page)
    return { page: Number.isInteger(page) && page > 0 ? page : 0 }
  },
  component: DashboardPage,
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
    tab: search.tab === "files" || search.tab === "review" ? search.tab : "description",
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
