import { createSignal, For, Show } from "solid-js";
import { agentLabel, fileName, formatMs, opencodeUrl, type ReviewState } from "../review";
import type { RankedConcern } from "../types";
import { AgentPicker } from "./AgentPicker";
import { ChevronIcon, RerunIcon, StopIcon } from "./icons";
import { AgentButton } from "./ReviewPage";

type Props = {
  state: ReviewState;
  activeRank?: number;
  onFocusConcern: (concern: RankedConcern) => void;
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
  const [collapsed, setCollapsed] = createSignal(false);
  const activeStage = () =>
    s()
      .review()
      ?.stages.find((stage) => stage.status === "running");

  const isRerun = () => s().synced() && !s().busy() && !!s().review();

  const runLabel = () =>
    !s().synced() ? "connecting..." : s().review() ? "re-run" : "run";

  return (
    <aside class="review-panel" classList={{ collapsed: collapsed() }}>
      <div class="review-panel-head" onClick={() => setCollapsed(!collapsed())}>
        <button
          class="panel-collapse"
          aria-expanded={!collapsed()}
          aria-label={collapsed() ? "expand agent review" : "collapse agent review"}
        >
          <ChevronIcon />
        </button>
        <span class="review-panel-title">agent review</span>
        <Show when={s().review()?.opencodeSessionPath}>
          <a
            class="review-session-link"
            href={opencodeUrl(s().review()?.opencodeSessionPath)}
            target="_blank"
            rel="noreferrer"
            onClick={(e) => e.stopPropagation()}
          >
            continue in opencode ↗
          </a>
        </Show>
        <Show
          when={s().busy()}
          fallback={
            <button
              class="review-run"
              title={runLabel()}
              aria-label={runLabel()}
              onClick={(e) => {
                e.stopPropagation()
                s().start()
              }}
              disabled={!s().canRun()}
            >
              <Show when={isRerun()} fallback={runLabel()}>
                <RerunIcon />
              </Show>
            </button>
          }
        >
          <button
            class="review-run is-stop"
            title="stop this review"
            aria-label="stop this review"
            onClick={(e) => {
              e.stopPropagation()
              s().cancel()
            }}
            disabled={s().stopping()}
          >
            <StopIcon />
          </button>
        </Show>
      </div>

      <AgentPicker state={s()} />

      <Show when={s().error()}>
        <div class="review-error">{s().error()}</div>
      </Show>

      <Show when={s().synced() && !s().connected()}>
        <div class="review-stale">connection lost - reconnecting</div>
      </Show>

      <Show when={!s().synced()}>
        <p class="review-idle">syncing with server...</p>
      </Show>

      <Show when={s().busy()}>
        <div class="review-progress">
          <span class="review-stage-dot" />
          <span class="review-stage-name">
            {stageLabels[activeStage()?.name ?? ""] ?? "starting"}
          </span>
          <span class="review-stage-time">{formatMs(s().elapsed())}</span>
        </div>
      </Show>

      <Show when={(s().review()?.agents ?? []).length > 0}>
        <ul class="agent-progress">
          <For each={s().review()?.agents ?? []}>
            {(agent) => (
              <li class={`agent-progress-row is-${agent.status}`}>
                <span class="review-stage-dot" />
                <span class="agent-progress-name">{agentLabel(agent.name)}</span>
                <span class="review-stage-detail">
                  {agent.status === "error"
                    ? "failed"
                    : agent.status === "cancelled"
                      ? "stopped"
                      : (stageLabels[
                          agent.stages.find((st) => st.status === "running")?.name ?? ""
                        ] ?? "")}
                </span>
                <AgentButton state={s()} agent={agent} />
              </li>
            )}
          </For>
        </ul>
      </Show>

      <Show when={s().synced() && !s().busy() && !s().review()}>
        <p class="review-idle">no review yet</p>
      </Show>

      <Show when={s().review()?.status === "error"}>
        <div class="review-error">{s().review()?.error}</div>
      </Show>

      <Show when={s().review()?.status === "cancelled"}>
        <div class="review-stopped">stopped</div>
      </Show>

      <Show when={s().review()?.status === "done"}>
        <Show
          when={s().concerns().length > 0}
          fallback={<div class="review-clear">no concerns</div>}
        >
          <ul class="concern-index">
            <For each={s().concerns()}>
              {(concern) => (
                <ConcernRow
                  concern={concern}
                  active={props.activeRank === concern.rank}
                  onFocus={() => props.onFocusConcern(concern)}
                />
              )}
            </For>
          </ul>
        </Show>
      </Show>
    </aside>
  );
}

function ConcernRow(props: { concern: RankedConcern; active: boolean; onFocus: () => void }) {
  const locatable = () => props.concern.line != null && props.concern.side != null;

  return (
    <li
      class={`concern-row sev-${props.concern.severity} ${props.active ? "active" : ""} ${locatable() ? "locatable" : ""}`}
      onClick={() => locatable() && props.onFocus()}
    >
      <span class="concern-row-title">{props.concern.title}</span>
      <span class="concern-row-file">
        {fileName(props.concern.file)}
        <Show when={props.concern.line} fallback={<span class="concern-unpinned"> · file</span>}>
          :{props.concern.line}
        </Show>
      </span>
    </li>
  );
}
