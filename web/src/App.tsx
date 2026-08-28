import { QueryClient, QueryClientProvider } from "@tanstack/solid-query"
import { RouterProvider } from "@tanstack/solid-router"
import { router } from "./router"

const client = new QueryClient()

export default function App() {
  return (
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}
