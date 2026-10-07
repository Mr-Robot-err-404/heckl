import { onCleanup, onMount } from "solid-js";
import { agentLabel } from "../review";
import type { RankedConcern } from "../types";
import { CloseIcon } from "./icons";

type Props = {
  concern: RankedConcern;
  onFocus: () => void;
  onClose: () => void;
};

export function ConcernModal(props: Props) {
  let dialog: HTMLDivElement | undefined;
  let closeButton: HTMLButtonElement | undefined;
  let previousFocus: HTMLElement | null;

  onMount(() => {
    previousFocus = document.activeElement as HTMLElement | null;
    closeButton?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        props.onClose();
      }
      if (event.key === "Tab" && dialog) {
        const focusable = Array.from(dialog.querySelectorAll<HTMLElement>("button, a[href]"));
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last?.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first?.focus();
        }
      }
    };
    window.addEventListener("keydown", onKeyDown);
    onCleanup(() => {
      window.removeEventListener("keydown", onKeyDown);
      previousFocus?.focus();
    });
  });

  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div
        ref={dialog}
        class="modal concern-modal"
        role="dialog"
        aria-modal="true"
        aria-label={props.concern.title}
        tabIndex={-1}
        onClick={(event) => event.stopPropagation()}
      >
        <div class="modal-head">
          <span class="review-panel-title">{agentLabel(props.concern.agent)} · {props.concern.severity}</span>
          <button ref={closeButton} class="modal-close" aria-label="close concern" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>
        <div class="modal-body">
          <h2 class="concern-modal-title">{props.concern.title}</h2>
          <button class="concern-modal-location" onClick={props.onFocus}>
            {props.concern.file}{props.concern.line != null ? `:${props.concern.line}` : ""} →
          </button>
          <p class="concern-modal-body">{props.concern.body}</p>
        </div>
      </div>
    </div>
  );
}
