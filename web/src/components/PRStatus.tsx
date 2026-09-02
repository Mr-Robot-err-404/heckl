import { For, Show } from "solid-js"
import type { GitHubUser, PR } from "../types"

const maxAvatars = 4

type ReviewerState = "approved" | "changes" | "pending"

type Reviewer = {
  login: string
  avatar: string
  state: ReviewerState
  isViewer: boolean
}

function BotIcon() {
  return (
    <svg class="bot-icon" viewBox="0 0 16 16" aria-hidden="true">
      <path
        fill="currentColor"
        d="M8 1a.75.75 0 0 1 .75.75V3h2.5A2.75 2.75 0 0 1 14 5.75v4.5A2.75 2.75 0 0 1 11.25 13h-6.5A2.75 2.75 0 0 1 2 10.25v-4.5A2.75 2.75 0 0 1 4.75 3h2.5V1.75A.75.75 0 0 1 8 1Zm-3.25 3.5c-.69 0-1.25.56-1.25 1.25v4.5c0 .69.56 1.25 1.25 1.25h6.5c.69 0 1.25-.56 1.25-1.25v-4.5c0-.69-.56-1.25-1.25-1.25h-6.5Z"
      />
      <circle cx="6" cy="8" r="1.15" fill="currentColor" />
      <circle cx="10" cy="8" r="1.15" fill="currentColor" />
      <path fill="currentColor" d="M0 7.25a.75.75 0 0 1 1.5 0v1.5a.75.75 0 0 1-1.5 0v-1.5Zm14.5 0a.75.75 0 0 1 1.5 0v1.5a.75.75 0 0 1-1.5 0v-1.5Z" />
    </svg>
  )
}

function reviewers(pr: PR): Reviewer[] {
  const seen = new Set<string>()
  const out: Reviewer[] = []

  const add = (users: GitHubUser[] | undefined, state: ReviewerState) => {
    for (const user of users ?? []) {
      if (seen.has(user.login)) continue
      seen.add(user.login)
      out.push({
        login: user.login,
        avatar: user.avatar,
        state,
        isViewer: state === "approved" && pr.viewerApproved && users === pr.approvals,
      })
    }
  }

  add(pr.changesRequested, "changes")
  add(pr.approvals, "approved")
  add(pr.requestedReviewers, "pending")

  return out
}

export function PRStatus(props: { pr: PR; reviewing: boolean }) {
  const all = () => reviewers(props.pr)
  const shown = () => all().slice(0, maxAvatars)
  const overflow = () => all().length - shown().length

  const summary = () =>
    all()
      .map((r) => `${r.login}${r.state === "approved" ? " ✓" : r.state === "changes" ? " ✗" : ""}`)
      .join(", ")

  const agent = () => props.pr.review

  return (
    <div class="pr-status">
      <Show when={all().length > 0}>
        <span class="reviewers" title={summary()}>
          <For each={shown()}>
            {(reviewer) => (
              <span class={`reviewer is-${reviewer.state}`}>
                <img class="avatar" src={reviewer.avatar} alt={reviewer.login} loading="lazy" />
                <Show when={reviewer.state !== "pending"}>
                  <span class="reviewer-mark" classList={{ "is-viewer": reviewer.isViewer }}>
                    {reviewer.state === "approved" ? "✓" : "✗"}
                  </span>
                </Show>
              </span>
            )}
          </For>
          <Show when={overflow() > 0}>
            <span class="reviewer">
              <span class="avatar avatar-more">+{overflow()}</span>
            </span>
          </Show>
        </span>
      </Show>

      <Show when={props.reviewing}>
        <span class="agent-mark is-running" title="agent review in progress">
          <BotIcon />
        </span>
      </Show>

      <Show when={!props.reviewing && agent()} keyed>
        {(review) => (
          <span
            class="agent-mark"
            classList={{
              "is-error": review.status === "error",
              "has-high": review.status === "done" && review.highCount > 0,
            }}
            title={
              review.status === "error"
                ? "last agent review failed"
                : `agent reviewed — ${review.concernCount} concerns, ${review.highCount} high`
            }
          >
            <BotIcon />
            <Show when={review.status === "done" && review.concernCount > 0}>
              <span class="agent-count">{review.concernCount}</span>
            </Show>
          </span>
        )}
      </Show>
    </div>
  )
}
