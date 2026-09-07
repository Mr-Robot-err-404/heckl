const ROW_BLUE = "color-mix(in srgb, var(--blue) 6%, transparent)"
const BUTTON_BLUE = "color-mix(in srgb, var(--blue) 34%, transparent)"
const BUTTON_HOVER_BLUE = "color-mix(in srgb, var(--blue) 52%, transparent)"

const BUTTON_WIDTH = 68

export const diffUnsafeCSS = `
  [data-change-icon] { display: none; }

  [data-code] { scrollbar-gutter: auto; }

  [data-separator="line-info"] {
    height: 1lh;
    margin-block: 0;
  }

  [data-separator="line-info"] [data-separator-wrapper] {
    grid-template-columns: ${BUTTON_WIDTH}px auto;
    padding-left: 0;
    background-color: ${ROW_BLUE};
  }

  [data-separator="line-info"] [data-separator-wrapper][data-separator-multi-button] {
    grid-template-columns: ${BUTTON_WIDTH / 2}px ${BUTTON_WIDTH / 2}px auto;
  }

  [data-separator="line-info"] [data-separator-multi-button] [data-expand-up] {
    grid-column: 1;
  }

  [data-separator="line-info"] [data-separator-multi-button] [data-expand-down] {
    grid-column: 2;
  }

  [data-separator="line-info"] [data-expand-button] {
    justify-content: center;
    color: var(--text);
    background-color: ${BUTTON_BLUE};
    border-radius: 0 !important;
  }

  [data-separator="line-info"] [data-expand-button]:hover {
    background-color: ${BUTTON_HOVER_BLUE};
  }

  [data-separator="line-info"] [data-separator-content] {
    background-color: transparent;
    border-radius: 0 !important;
  }
`
