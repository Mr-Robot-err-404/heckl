import { createMemo, createSignal, For, Show } from "solid-js"
import { useNavigate } from "@tanstack/solid-router"
import { resolved, useNotifications, usePrefetch, useReadNotifications } from "../queries"
import { relativeTime } from "../review"
import { BellIcon, CloseIcon } from "./icons"
import type { Notification } from "../types"

const reasonLabels: Record<string, string> = {
  assign: "assigned",
  author: "author",
  comment: "commented",
  mention: "mentioned",
  review_requested: "review requested",
  team_mention: "team mentioned",
}

const reasonLabel = (reason: string) => reasonLabels[reason] ?? reason

export function NotificationsModal(props: { onClose: () => void }) {
  const notifications = useNotifications()
  const data = resolved(notifications)
  const read = useReadNotifications()
  const navigate = useNavigate()
  const prefetch = usePrefetch()

  const [picked, setPicked] = createSignal<string[]>([])

  const rows = () => data() ?? []
  const isPicked = (id: string) => picked().includes(id)
  const allPicked = createMemo(
    () => rows().length > 0 && rows().every((row) => isPicked(row.id)),
  )

  const toggle = (id: string) =>
    setPicked((prev) => (prev.includes(id) ? prev.filter((k) => k !== id) : [...prev, id]))

  const toggleAll = () => setPicked(allPicked() ? [] : rows().map((row) => row.id))

  const markPicked = () => {
    const ids = picked()
    if (ids.length === 0) return
    setPicked([])
    read.mutate(ids)
  }

  const open = (row: Notification) => {
    read.mutate([row.id])
    props.onClose()
    navigate({
      to: "/$owner/$repo/$pr",
      params: { owner: row.owner, repo: row.repo, pr: String(row.prNumber) },
      search: { tab: "files" },
    })
  }

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal modal-notifications" onClick={(e) => e.stopPropagation()}>
        <div class="modal-head">
          <span class="review-panel-title">notifications</span>
          <button class="modal-close" title="close" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>

        <div class="modal-body">
          <Show when={notifications.error}>
            <div class="empty error">{String(notifications.error)}</div>
          </Show>

          <Show
            when={rows().length > 0}
            fallback={
              <Show
                when={!notifications.isPending}
                fallback={<div class="empty">checking github...</div>}
              >
                <Show when={notifications.isSuccess}>
                  <div class="empty deliberate-empty">
                    <BellIcon />
                    <span>Notifications up to date</span>
                  </div>
                </Show>
              </Show>
            }
          >
            <ul class="review-rows notification-rows">
              <li class="notification-row is-head">
                <input
                  type="checkbox"
                  checked={allPicked()}
                  aria-label="select every notification"
                  onChange={toggleAll}
                />
                <div class="tmux-actions">
                  <button
                    class="topbar-btn"
                    classList={{ danger: picked().length > 0 }}
                    disabled={picked().length === 0}
                    onClick={markPicked}
                  >
                    seen
                  </button>
                </div>
              </li>

              <For each={rows()}>
                {(row) => (
                  <li
                    class="review-row notification-row"
                    classList={{ "is-picked": isPicked(row.id) }}
                    onMouseEnter={() => prefetch.pr(row.owner, row.repo, row.prNumber)}
                    onClick={() => open(row)}
                  >
                    <input
                      type="checkbox"
                      checked={isPicked(row.id)}
                      aria-label={`select ${row.title}`}
                      onClick={(e) => e.stopPropagation()}
                      onChange={() => toggle(row.id)}
                    />
                    <span class="notification-reason">{reasonLabel(row.reason)}</span>
                    <div class="notification-content">
                      <div class="notification-title" title={row.title}>{row.title}</div>
                      <Show when={row.activityActor || row.activityBody}>
                        <div class="notification-activity">
                          <Show when={row.activityActor}>
                            <span class="notification-actor">{row.activityActor}</span>
                          </Show>
                          <Show when={row.activityBody}>
                            <span>{row.activityBody}</span>
                          </Show>
                        </div>
                      </Show>
                    </div>
                    <span class="notification-repo muted">
                      {row.repo} #{row.prNumber}
                    </span>
                    <span
                      class="notification-state"
                      classList={{ [`is-${row.state}`]: row.state !== undefined }}
                    >
                      {row.state ?? ""}
                    </span>
                    <span class="tmux-age muted">{relativeTime(row.updatedAt)}</span>
                  </li>
                )}
              </For>
            </ul>
          </Show>
        </div>
      </div>
    </div>
  )
}
