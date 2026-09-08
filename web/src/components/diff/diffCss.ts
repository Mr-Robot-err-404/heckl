const ROW_BLUE = "color-mix(in srgb, var(--blue) 6%, transparent)"
const BUTTON_BLUE = "color-mix(in srgb, var(--blue) 34%, transparent)"
const BUTTON_HOVER_BLUE = "color-mix(in srgb, var(--blue) 52%, transparent)"

const SPLIT_HEIGHT = "32px"

const NUMBER_COLUMN = "4ch"
const NUMBER_PADDING_LEFT = "2ch"
const NUMBER_PADDING_RIGHT = "1ch"
const NUMBER_BORDER = "2px"
const BUTTON_WIDTH = `calc(${NUMBER_PADDING_LEFT} + ${NUMBER_COLUMN} + ${NUMBER_PADDING_RIGHT} + ${NUMBER_BORDER})`

export const diffUnsafeCSS = `
  :host {
    --diffs-min-number-column-width: ${NUMBER_COLUMN};
    --diff-gutter-width: ${BUTTON_WIDTH};
    --diff-gutter-border: ${NUMBER_BORDER};
  }

  [data-change-icon] { display: none; }

  [data-diffs-header="default"] { padding-left: 0; }

  [data-code] {
    --diffs-scrollbar-gutter-override: 0px;
    scrollbar-gutter: auto;
    scrollbar-width: none;
    padding-bottom: 0;
  }

  [data-separator="line-info"] {
    height: 1lh;
    margin-block: 0;
  }

  [data-separator="line-info"]:has([data-separator-multi-button]) {
    height: ${SPLIT_HEIGHT};
  }

  [data-separator="line-info"] [data-separator-wrapper] {
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
    grid-template-columns: var(--diff-gutter-width) auto;
    padding-inline: 0;
    background-color: ${ROW_BLUE};
  }

  [data-separator="line-info"] [data-expand-button] {
    min-width: 0;
    color: var(--text);
    background-color: ${BUTTON_BLUE};
    border-radius: 0;
  }

  [data-separator="line-info"] [data-expand-button]:hover {
    background-color: ${BUTTON_HOVER_BLUE};
  }

  [data-separator="line-info"] [data-separator-content] {
    font-family: var(--diffs-header-font-family, var(--diffs-header-font-fallback));
    background-color: transparent;
    border-radius: 0;
  }
`
