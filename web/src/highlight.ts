import { createSignal } from "solid-js"
import { getHighlighterIfLoaded, getSharedHighlighter } from "@pierre/diffs"

const [highlightVersion, setHighlightVersion] = createSignal(0)

export { highlightVersion }

const inFlight = new Set<string>()
const unavailable = new Set<string>()

function canHighlight(lang: string, themeName: string): boolean {
  const h = getHighlighterIfLoaded()
  if (!h) return false
  try {
    h.codeToHtml("", { lang, theme: themeName })
    return true
  } catch {
    return false
  }
}

function request(lang: string, themeName: string) {
  const key = `${lang}\u0000${themeName}`
  if (inFlight.has(key) || unavailable.has(key)) return
  inFlight.add(key)

  getSharedHighlighter({ themes: [themeName], langs: [lang] })
    .then(() => {
      if (!canHighlight(lang, themeName)) {
        unavailable.add(key)
        return
      }
      setHighlightVersion((v) => v + 1)
    })
    .catch(() => {
      unavailable.add(key)
    })
    .finally(() => {
      inFlight.delete(key)
    })
}

export function highlightCode(code: string, lang: string, themeName: string): string | null {
  const h = getHighlighterIfLoaded()
  if (!h) {
    request(lang, themeName)
    return null
  }
  try {
    return h.codeToHtml(code, { lang, theme: themeName })
  } catch {
    request(lang, themeName)
    return null
  }
}
