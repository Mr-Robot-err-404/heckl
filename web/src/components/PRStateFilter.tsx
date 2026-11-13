type Props = {
  state: "open" | "closed"
  onChange: (state: "open" | "closed") => void
}

export function PRStateFilter(props: Props) {
  return (
    <div class="pr-state-filter" role="group" aria-label="Pull request state">
      <button classList={{ active: props.state === "open" }} aria-pressed={props.state === "open"} onClick={() => props.onChange("open")}>OPEN</button>
      <button classList={{ active: props.state === "closed" }} aria-pressed={props.state === "closed"} onClick={() => props.onChange("closed")} title="20 most recently updated closed pull requests">CLOSED</button>
    </div>
  )
}
