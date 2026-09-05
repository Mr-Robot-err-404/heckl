import { createSignal, onCleanup, Show } from "solid-js"

export function createToast(durationMs = 1400) {
  const [message, setMessage] = createSignal("")

  let timer: ReturnType<typeof setTimeout> | undefined
  const flash = (text: string) => {
    clearTimeout(timer)
    setMessage(text)
    timer = setTimeout(() => setMessage(""), durationMs)
  }

  onCleanup(() => clearTimeout(timer))
  return { message, flash }
}

export function Toast(props: { message: string }) {
  return (
    <Show when={props.message}>
      <div class="toast">{props.message}</div>
    </Show>
  )
}
