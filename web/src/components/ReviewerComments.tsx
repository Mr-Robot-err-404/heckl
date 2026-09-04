import { createSignal, For, Show } from "solid-js"
import { fileName, relativeTime } from "../review"
import { Markdown } from "./Markdown"
import type { ReviewerNote, ReviewerThread } from "../types"

type Props = {
  threads: ReviewerThread[]
  pending: boolean
  onFocusNote: (note: ReviewerNote) => void
}

const stateLabels: Record<string, string> = {
  APPROVED: "approved",
  CHANGES_REQUESTED: "changes requested",
  COMMENTED: "commented",
  DISMISSED: "dismissed",
}

export function ReviewerComments(props: Props) {
  return (
    <section class="reviewer-comments">
      <div class="reviewer-comments-head">
        <span class="review-panel-title">other reviewers</span>
        <Show when={props.threads.length > 0}>
          <span class="review-stage-time">{props.threads.length}</span>
        </Show>
      </div>

      <Show when={props.pending}>
        <p class="review-idle">loading comments...</p>
      </Show>

      <Show when={!props.pending && props.threads.length === 0}>
        <p class="review-idle">no comments from other reviewers</p>
      </Show>

      <For each={props.threads}>
        {(thread) => <Thread thread={thread} onFocusNote={props.onFocusNote} />}
      </For>
    </section>
  )
}

function Thread(props: { thread: ReviewerThread; onFocusNote: (note: ReviewerNote) => void }) {
  const [open, setOpen] = createSignal(false)

  return (
    <div class={`reviewer-thread ${open() ? "open" : ""}`}>
      <button class="reviewer-thread-head" onClick={() => setOpen(!open())}>
        <span class="reviewer-caret">{open() ? "▾" : "▸"}</span>
        <Show when={props.thread.user.avatar}>
          <img class="reviewer-avatar" src={props.thread.user.avatar} alt="" />
        </Show>
        <span class="reviewer-login">{props.thread.user.login}</span>
        <span class="reviewer-count">{props.thread.notes.length}</span>
      </button>

      <Show when={open()}>
        <ul class="reviewer-notes">
          <For each={props.thread.notes}>
            {(note) => <Note note={note} onFocus={() => props.onFocusNote(note)} />}
          </For>
        </ul>
      </Show>
    </div>
  )
}

function Note(props: { note: ReviewerNote; onFocus: () => void }) {
  const locatable = () =>
    !!props.note.file && props.note.line != null && props.note.side != null

  return (
    <li class="reviewer-note">
      <div class="reviewer-note-head">
        <Show when={props.note.state}>
          <span class={`reviewer-state is-${props.note.state?.toLowerCase()}`}>
            {stateLabels[props.note.state ?? ""] ?? props.note.state}
          </span>
        </Show>
        <Show when={props.note.file}>
          <Show
            when={locatable()}
            fallback={
              <span class="reviewer-note-loc">
                {fileName(props.note.file!)}
                <Show when={props.note.outdated}>
                  <span class="concern-unpinned"> · outdated</span>
                </Show>
              </span>
            }
          >
            <button class="reviewer-note-loc link" onClick={props.onFocus}>
              {fileName(props.note.file!)}:{props.note.line} →
            </button>
          </Show>
        </Show>
        <Show when={props.note.reply}>
          <span class="reviewer-reply">reply</span>
        </Show>
        <span class="reviewer-note-time">{relativeTime(props.note.createdAt)}</span>
      </div>
      <div class="reviewer-note-body">
        <Markdown content={props.note.body} />
      </div>
    </li>
  )
}
