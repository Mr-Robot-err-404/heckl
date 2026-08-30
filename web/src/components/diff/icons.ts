const SVG_NS = "http://www.w3.org/2000/svg"

function svg(paths: string[], viewBox = "0 0 16 16") {
  const el = document.createElementNS(SVG_NS, "svg")
  el.setAttribute("viewBox", viewBox)
  el.setAttribute("width", "14")
  el.setAttribute("height", "14")
  el.setAttribute("fill", "none")
  for (const d of paths) {
    const path = document.createElementNS(SVG_NS, "path")
    path.setAttribute("d", d)
    path.setAttribute("stroke", "currentColor")
    path.setAttribute("stroke-width", "1.5")
    path.setAttribute("stroke-linecap", "round")
    path.setAttribute("stroke-linejoin", "round")
    el.appendChild(path)
  }
  return el
}

export function chevronIcon() {
  return svg(["M4 6l4 4 4-4"])
}

export function clipboardIcon() {
  return svg([
    "M5.5 3.5h5a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-5a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1Z",
    "M6.5 3.5V3a1.5 1.5 0 0 1 1.5-1.5h0A1.5 1.5 0 0 1 9.5 3v.5",
  ])
}

export function checkIcon() {
  return svg(["M3.5 8.5l3 3 6-7"])
}
