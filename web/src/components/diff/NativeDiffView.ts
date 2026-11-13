import { FileDiff, type DiffLineAnnotation, type FileDiffMetadata, type FileDiffOptions, type SelectedLineRange, type SelectionWriteOptions } from "@pierre/diffs"

export type DiffItem<T> = {
  id: string
  fileDiff: FileDiffMetadata
  annotations?: DiffLineAnnotation<T>[]
  collapsed?: boolean
}

type Selection = { id: string; range: SelectedLineRange } | null
type ScrollTarget = { id: string; align: "start" | "center" } & (
  | { type: "item" }
  | { type: "line"; lineNumber: number; side: "additions" | "deletions" }
)

type Options<T> = FileDiffOptions<T> & {
  onSelectedLinesChange?: (selection: Selection) => void
}

export class NativeDiffView<T> {
  private host!: HTMLElement
  private files = new Map<string, { item: DiffItem<T>; instance: FileDiff<T>; element: HTMLElement }>()
  private selection: Selection = null

  constructor(private options: Options<T>) {}

  setup(host: HTMLElement) {
    this.host = host
  }

  setItems(items: DiffItem<T>[]) {
    for (const item of items) {
      const element = document.createElement("diffs-container")
      element.style.display = "block"
      this.host.append(element)
      const instance = new FileDiff<T>({
        ...this.options,
        onLineSelected: (range) => {
          this.setSelectedLines(range ? { id: item.id, range } : null, { notify: false })
          this.options.onSelectedLinesChange?.(this.selection)
        },
      })
      this.files.set(item.id, { item, instance, element })
      instance.render({ fileContainer: element, fileDiff: item.fileDiff, lineAnnotations: item.annotations })
    }
  }

  getItem(id: string) {
    return this.files.get(id)?.item
  }

  setTheme(theme: FileDiffOptions<T>["theme"]) {
    if (this.options.theme === theme) return
    this.options.theme = theme
    for (const file of this.files.values()) {
      file.instance.setOptions({ ...file.instance.options, theme })
      file.instance.rerender()
    }
  }

  updateItem(item: DiffItem<T>) {
    const file = this.files.get(item.id)
    if (!file) return
    const collapsed = file.item.collapsed !== item.collapsed
    file.item = item
    file.instance.setLineAnnotations(item.annotations ?? [])
    if (collapsed) {
      file.instance.setOptions({ ...file.instance.options, collapsed: item.collapsed })
    }
    file.instance.rerender()
  }

  getSelectedLines() {
    return this.selection
  }

  setSelectedLines(selection: Selection, options?: SelectionWriteOptions) {
    if (this.selection && this.selection.id !== selection?.id) {
      this.files.get(this.selection.id)?.instance.setSelectedLines(null, { notify: false })
    }
    this.selection = selection
    if (selection) this.files.get(selection.id)?.instance.setSelectedLines(selection.range, { notify: false })
    if (options?.notify !== false) this.options.onSelectedLinesChange?.(selection)
  }

  scrollTo(target: ScrollTarget) {
    const file = this.files.get(target.id)
    if (!file) return
    let element = file.element
    if (target.type === "line") {
      const index = file.instance.getLineIndex(target.lineNumber, target.side)
      if (index) element = file.element.shadowRoot?.querySelector<HTMLElement>(`[data-line-index="${index.join(",")}"]`) ?? element
    }
    const top = element.getBoundingClientRect().top - this.host.getBoundingClientRect().top + this.host.scrollTop
    this.host.scrollTo({ top: target.align === "center" ? top - this.host.clientHeight / 2 + element.clientHeight / 2 : top, behavior: "instant" })
  }

  cleanUp() {
    for (const file of this.files.values()) {
      file.instance.cleanUp()
      file.element.remove()
    }
    this.files.clear()
  }
}
