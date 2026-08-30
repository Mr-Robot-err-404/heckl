const SVG_NS = "http://www.w3.org/2000/svg"

type ShapeSpec = { tag: "path"; d: string } | { tag: "rect"; x: number; y: number; width: number; height: number; rx: number }

function icon(shapes: ShapeSpec[]) {
  const el = document.createElementNS(SVG_NS, "svg")
  el.setAttribute("viewBox", "0 0 24 24")
  el.setAttribute("width", "14")
  el.setAttribute("height", "14")
  el.setAttribute("fill", "none")
  el.setAttribute("stroke", "currentColor")
  el.setAttribute("stroke-width", "2")
  el.setAttribute("stroke-linecap", "round")
  el.setAttribute("stroke-linejoin", "round")
  for (const shape of shapes) {
    if (shape.tag === "path") {
      const path = document.createElementNS(SVG_NS, "path")
      path.setAttribute("d", shape.d)
      el.appendChild(path)
    } else {
      const rect = document.createElementNS(SVG_NS, "rect")
      rect.setAttribute("x", String(shape.x))
      rect.setAttribute("y", String(shape.y))
      rect.setAttribute("width", String(shape.width))
      rect.setAttribute("height", String(shape.height))
      rect.setAttribute("rx", String(shape.rx))
      el.appendChild(rect)
    }
  }
  return el
}

export function chevronIcon() {
  return icon([{ tag: "path", d: "m6 9 6 6 6-6" }])
}

export function clipboardIcon() {
  return icon([
    { tag: "rect", x: 8, y: 8, width: 14, height: 14, rx: 2 },
    { tag: "path", d: "M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" },
  ])
}

export function checkIcon() {
  return icon([{ tag: "path", d: "M20 6 9 17l-5-5" }])
}
