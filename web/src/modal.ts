import { useNavigate, useSearch } from "@tanstack/solid-router"

export type ModalId = "theme" | "agents" | "tmux"

const ids: ModalId[] = ["theme", "agents", "tmux"]

export function validateModal(search: Record<string, unknown>): { modal?: ModalId } {
  const modal = ids.find((id) => id === search.modal)
  return modal ? { modal } : {}
}

export function useModal() {
  const search = useSearch({ strict: false })
  const navigate = useNavigate()

  const isOpen = (id: ModalId) => search().modal === id

  const open = (id: ModalId) =>
    navigate({ to: ".", search: (prev: Record<string, unknown>) => ({ ...prev, modal: id }) })

  const close = () =>
    navigate({
      to: ".",
      replace: true,
      search: (prev: Record<string, unknown>) => ({ ...prev, modal: undefined }),
    })

  return { isOpen, open, close }
}
