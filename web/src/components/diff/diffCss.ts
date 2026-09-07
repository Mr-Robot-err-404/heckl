const SUBTLE_BLUE = "color-mix(in srgb, var(--blue) 14%, transparent)"
const STRONG_BLUE = "color-mix(in srgb, var(--blue) 34%, transparent)"
const HOVER_BLUE = "color-mix(in srgb, var(--blue) 52%, transparent)"

export const diffUnsafeCSS = `
  [data-change-icon] { display: none; }

  [data-code] { scrollbar-gutter: auto; }

  [data-separator="line-info"] {
    height: 1lh;
    margin-block: 0;
  }

  [data-separator="line-info"] [data-separator-wrapper],
  [data-separator="line-info"] [data-separator-wrapper][data-separator-multi-button] {
    grid-template-columns: 68px auto;
    padding-left: 0;
    background-color: ${SUBTLE_BLUE};
  }

  [data-separator="line-info"] [data-expand-button] {
    justify-content: center;
    color: var(--text);
    background-color: ${STRONG_BLUE};
    border-radius: 0 !important;
  }

  [data-separator="line-info"] [data-expand-button]:hover {
    background-color: ${HOVER_BLUE};
  }

  [data-separator="line-info"] [data-separator-content] {
    background-color: transparent;
    border-radius: 0 !important;
  }
`
