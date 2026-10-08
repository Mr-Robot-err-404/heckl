import { createSignal, For, Show } from "solid-js";
import { agentLabel, fileName, formatMs, type ReviewState } from "../review";
import type { RankedConcern, ReviewAgent } from "../types";
import { AgentPicker } from "./AgentPicker";
import { BotIcon, ChevronIcon, DetailsIcon, PlayIcon, RerunIcon, StopIcon } from "./icons";

type Props = {
  state: ReviewState;
  activeRank?: number;
  onFocusConcern: (concern: RankedConcern) => void;
  onOpenConcern: (concern: RankedConcern) => void;
};

const stageLabels: Record<string, string> = {
  fetch: "fetching pr",
  checkout: "checking out",
  session: "starting agent",
  prompt: "reviewing",
  parse: "reading result",
  store: "saving",
};

export function ReviewPanel(props: Props) {
  const s = () => props.state;
  const activeStage = () => s().review()?.stages.find((stage) => stage.status === "running");
  const agentNames = () => [...new Set([
    ...(s().review()?.agents ?? []).map((agent) => agent.name),
    ...s().concerns().map((concern) => concern.agent),
  ])];

  return (
    <>
      <Show when={!s().review() || s().busy() || s().error() || (s().synced() && !s().connected())}>
        <section class="review-panel">
          <Show when={!s().review()}>
            <div class="review-panel-head">
              <span class="review-panel-title">agent review</span>
              <button class="agent-rerun" title="run review" aria-label="run review"
                onClick={() => s().start()} disabled={!s().canRun()}><PlayIcon /></button>
            </div>
          </Show>

          <Show when={!s().review()}><AgentPicker state={s()} /></Show>
          <Show when={s().error()}><div class="review-error">{s().error()}</div></Show>
          <Show when={s().synced() && !s().connected()}><div class="review-stale">connection lost - reconnecting</div></Show>
          <Show when={!s().synced()}><p class="review-idle">syncing with server...</p></Show>
          <Show when={s().busy()}>
            <div class="review-progress">
              <span class="review-stage-dot" />
              <span class="review-stage-name">{stageLabels[activeStage()?.name ?? ""] ?? "starting"}</span>
              <span class="review-stage-time">{formatMs(s().elapsed())}</span>
              <button class="agent-rerun is-stop" title="stop this review" aria-label="stop this review"
                onClick={() => s().cancel()} disabled={s().stopping()}><StopIcon /></button>
            </div>
          </Show>
        </section>
      </Show>

      <For each={agentNames()}>
        {(name) => (
          <AgentThread
            agent={() => s().review()?.agents.find((item) => item.name === name) ?? { name, status: "done", stages: [], durationMs: 0 }}
            state={s()}
            activeRank={props.activeRank}
            onFocusConcern={props.onFocusConcern}
            onOpenConcern={props.onOpenConcern}
          />
        )}
      </For>

      <Show when={s().review()?.status === "error"}><div class="review-error">{s().review()?.error}</div></Show>
      <Show when={s().review()?.status === "cancelled"}><div class="review-stopped">stopped</div></Show>
    </>
  );
}

function AgentThread(props: {
  agent: () => ReviewAgent;
  state: ReviewState;
  activeRank?: number;
  onFocusConcern: (concern: RankedConcern) => void;
  onOpenConcern: (concern: RankedConcern) => void;
}) {
  const [open, setOpen] = createSignal(true);
  const agent = () => props.agent();
  const working = () => agent().status === "pending" || agent().status === "running";
  const progress = () => stageLabels[agent().stages.find((stage) => stage.status === "running")?.name ?? ""] ?? "reviewing";
  const concerns = () => props.state.concerns().filter((concern) => concern.agent === agent().name);

  return (
    <section class="reviewer-thread" classList={{ open: open() }}>
      <div class="reviewer-thread-head">
        <button class="agent-thread-toggle" aria-expanded={open()} aria-label={`${open() ? "collapse" : "expand"} ${agentLabel(agent().name)}`}
          onClick={() => setOpen(!open())}>
          <span class="reviewer-caret" aria-hidden="true"><ChevronIcon /></span>
          <span class={`agent-avatar is-${agent().name}`} aria-hidden="true"><BotIcon /></span>
          <span class="reviewer-login">{agentLabel(agent().name)}</span>
          <Show when={working()}>
            <span class="agent-thread-progress"><span class="review-stage-dot" />{progress()}</span>
          </Show>
          <Show when={agent().status === "error"}><span class="review-error">failed</span></Show>
          <Show when={agent().status === "cancelled"}><span class="review-stopped">stopped</span></Show>
          <Show when={concerns().length > 0}><span class="reviewer-count">{concerns().length}</span></Show>
        </button>
        <Show when={props.state.agentBusy(agent().name)} fallback={
          <button class="agent-rerun" title={`re-run ${agentLabel(agent().name)}`} aria-label={`re-run ${agentLabel(agent().name)}`}
            disabled={props.state.busy() || !props.state.review()?.sessionId}
            onClick={() => props.state.rerun(agent().name)}><RerunIcon /></button>
        }>
          <button class="agent-rerun is-stop" title={`stop ${agentLabel(agent().name)}`} aria-label={`stop ${agentLabel(agent().name)}`}
            onClick={() => props.state.cancelAgent(agent().name)}><StopIcon /></button>
        </Show>
      </div>

      <Show when={open()}>
        <Show when={agent().error}><div class="review-error agent-thread-error">{agent().error}</div></Show>
        <Show when={agent().status === "done"}>
          <Show when={concerns().length > 0} fallback={<div class="review-clear agent-thread-empty">no concerns</div>}>
            <ul class="concern-index">
              <For each={concerns()}>
                {(concern) => (
                  <ConcernRow
                    concern={concern}
                    active={props.activeRank === concern.rank}
                    onFocus={() => props.onFocusConcern(concern)}
                    onOpen={() => props.onOpenConcern(concern)}
                  />
                )}
              </For>
            </ul>
          </Show>
        </Show>
      </Show>
    </section>
  );
}

function ConcernRow(props: { concern: RankedConcern; active: boolean; onFocus: () => void; onOpen: () => void }) {
  const locatable = () => props.concern.line != null && props.concern.side != null;

  return (
    <li class={`concern-row sev-${props.concern.severity} ${props.active ? "active" : ""} ${locatable() ? "locatable" : ""}`}>
      <button
        class="concern-row-target"
        onClick={props.onFocus}
        onKeyUp={(event) => { if (event.code === "Space") props.onOpen(); }}
        onKeyDown={(event) => { if (event.code === "Space") event.preventDefault(); }}
        title="click to locate in diff; press Space for details"
      >
        <span class="concern-row-title">{props.concern.title}</span>
        <span class="concern-row-file">
          {fileName(props.concern.file)}
          <Show when={props.concern.line} fallback={<span class="concern-unpinned"> · file</span>}>
            :{props.concern.line}
          </Show>
        </span>
      </button>
      <button class="agent-rerun concern-row-details" onClick={props.onOpen} aria-label={`read ${props.concern.title}`} title="read details">
        <DetailsIcon />
      </button>
    </li>
  );
}
