const ROW_BLUE = "color-mix(in srgb, var(--blue) 6%, transparent)";
const BUTTON_BLUE = "color-mix(in srgb, var(--blue) 34%, transparent)";
const BUTTON_HOVER_BLUE = "color-mix(in srgb, var(--blue) 52%, transparent)";

const SPLIT_HEIGHT = "32px";

const NUMBER_COLUMN = "4ch";
const NUMBER_PADDING_LEFT = "2ch";
const NUMBER_PADDING_RIGHT = "1ch";
const NUMBER_BORDER = "2px";
const SPINNER_SIZE = "11px";
const BUTTON_WIDTH = `calc(${NUMBER_PADDING_LEFT} + ${NUMBER_COLUMN} + ${NUMBER_PADDING_RIGHT} + ${NUMBER_BORDER})`;

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
    position: relative;
  }

  [data-separator="line-info"] [data-expand-button]:hover {
    background-color: ${BUTTON_HOVER_BLUE};
  }

  :host([data-expanding]) [data-separator="line-info"] [data-expand-button] {
    pointer-events: none;
    background-color: ${BUTTON_BLUE};
  }

  :host([data-expanding]) [data-separator="line-info"] [data-expand-button] [data-icon] {
    visibility: hidden;
  }

  :host([data-expanding]) [data-separator="line-info"] [data-expand-button]::after {
    content: "";
    position: absolute;
    top: 50%;
    left: 50%;
    width: ${SPINNER_SIZE};
    height: ${SPINNER_SIZE};
    margin-top: calc(${SPINNER_SIZE} / -2);
    margin-left: calc((${SPINNER_SIZE} / -2) - (${NUMBER_BORDER} / 2));
    border: 1.5px solid color-mix(in srgb, var(--text) 22%, transparent);
    border-top-color: var(--text);
    border-radius: 50%;
    animation: diff-expand-spin 0.6s linear infinite;
  }

  @keyframes diff-expand-spin {
    to { transform: rotate(360deg); }
  }

  [data-separator="line-info"] [data-separator-content] {
    font-family: var(--diffs-header-font-family, var(--diffs-header-font-fallback));
    background-color: transparent;
    border-radius: 0;
  }
`;
