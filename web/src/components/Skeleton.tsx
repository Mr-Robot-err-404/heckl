import { For } from "solid-js"

const paragraphWidths = ["96%", "88%", "92%", "64%", "0", "90%", "84%", "70%"]

export function SkeletonLines() {
  return (
    <div class="skeleton-lines">
      <For each={paragraphWidths}>
        {(width) =>
          width === "0" ? (
            <div class="skeleton-gap" />
          ) : (
            <span class="skeleton skeleton-line" style={{ width }} />
          )
        }
      </For>
    </div>
  )
}

const fileHunks = [6, 3, 8]

export function SkeletonDiff() {
  return (
    <div class="skeleton-diff">
      <For each={fileHunks}>
        {(lines) => (
          <div class="skeleton-diff-file">
            <div class="skeleton-diff-head">
              <span class="skeleton skeleton-line" style={{ width: "34%" }} />
            </div>
            <div class="skeleton-diff-body">
              <For each={Array.from({ length: lines })}>
                {(_, i) => (
                  <span
                    class="skeleton skeleton-line"
                    style={{ width: `${45 + ((i() * 37) % 50)}%` }}
                  />
                )}
              </For>
            </div>
          </div>
        )}
      </For>
    </div>
  )
}
