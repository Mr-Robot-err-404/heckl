import { createSignal } from "solid-js"
import { api } from "./api"

export type ThemeId =
  | "gruvbox"
  | "github-dark"
  | "catppuccin-mocha"
  | "kanagawa-wave"
  | "tokyo-night"
  | "everforest"
  | "night-owl"

export type Theme = {
  id: ThemeId
  label: string
  shiki: string
  dark: boolean
}

export const themes: Theme[] = [
  { id: "gruvbox", label: "gruvbox", shiki: "gruvbox-dark-medium", dark: true },
  { id: "github-dark", label: "github dark", shiki: "github-dark-default", dark: true },
  { id: "catppuccin-mocha", label: "catppuccin mocha", shiki: "catppuccin-mocha", dark: true },
  { id: "kanagawa-wave", label: "kanagawa wave", shiki: "kanagawa-wave", dark: true },
  { id: "tokyo-night", label: "tokyo night", shiki: "tokyo-night", dark: true },
  { id: "everforest", label: "everforest", shiki: "everforest-light", dark: false },
  { id: "night-owl", label: "night owl", shiki: "night-owl-light", dark: false },
]

const defaultTheme = themes[0]

export function themeById(id: string): Theme {
  return themes.find((t) => t.id === id) ?? defaultTheme
}

const [theme, setThemeSignal] = createSignal<Theme>(defaultTheme)

export { theme }

function apply(t: Theme) {
  document.documentElement.dataset.theme = t.id
  document.documentElement.style.colorScheme = t.dark ? "dark" : "light"
  setThemeSignal(t)
}

export function setTheme(id: string) {
  apply(themeById(id))
  api.theme.set(id).catch(() => {})
}

export function loadTheme() {
  api.theme
    .get()
    .then((res) => apply(themeById(res.theme)))
    .catch(() => {})
}
