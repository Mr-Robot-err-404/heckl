type StreamHandlers = Record<string, (data: string) => void>

const retryDelays = [500, 1_000, 2_000, 4_000, 8_000, 10_000, 10_000, 10_000]

export function reconnectingEventStream(
  url: string,
  handlers: StreamHandlers,
  connected: (connected: boolean) => void,
) {
  let source: EventSource | undefined
  let timer: ReturnType<typeof setTimeout> | undefined
  let retry = 0
  let cancelled = false

  const connect = () => {
    if (cancelled) return

    const next = new EventSource(url)
    source = next

    for (const [event, handle] of Object.entries(handlers)) {
      next.addEventListener(event, (message) => {
        if (cancelled || source !== next) return
        handle((message as MessageEvent<string>).data)
        retry = 0
        connected(true)
      })
    }

    next.addEventListener("error", () => {
      if (cancelled || source !== next) return

      source = undefined
      next.close()
      connected(false)

      const delay = retryDelays[retry]
      if (delay === undefined) return
      retry++
      timer = setTimeout(() => {
        timer = undefined
        connect()
      }, delay)
    })
  }

  connect()

  return () => {
    cancelled = true
    if (timer !== undefined) clearTimeout(timer)
    source?.close()
    source = undefined
  }
}
