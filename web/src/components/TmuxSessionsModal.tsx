import { CloseIcon } from "./icons"
import { TmuxPanel } from "./TmuxPanel"

export function TmuxSessionsModal(props: { onClose: () => void }) {
  return (
    <div class="modal-backdrop" onClick={props.onClose}>
      <div class="modal modal-tmux-sessions" onClick={(event) => event.stopPropagation()}>
        <div class="modal-head">
          <span class="review-panel-title">tmux sessions</span>
          <button class="modal-close" title="close" onClick={props.onClose}>
            <CloseIcon />
          </button>
        </div>
        <div class="modal-body">
          <TmuxPanel hideHeading />
        </div>
      </div>
    </div>
  )
}
