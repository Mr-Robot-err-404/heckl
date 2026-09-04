import { For, Show } from "solid-js"
import type { GitHubUser, PR } from "../types"
import { BotIcon } from "./icons"

const maxAvatars = 4

type ReviewerState = "approved" | "changes" | "pending"

type Reviewer = {
  login: string
  avatar: string
  state: ReviewerState
  isViewer: boolean
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
