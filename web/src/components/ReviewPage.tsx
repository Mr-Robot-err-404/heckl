import { For, Show } from "solid-js";
import { agentLabel, formatMs, opencodeUrl, type ReviewState } from "../review";
import { RerunIcon, StopIcon } from "./icons";
import type { ReviewAgent, ReviewStage } from "../types";

const stageLabels: Record<string, string> = {
  fetch: "fetching pr",
  checkout: "checking out",
  session: "starting agent",
  prompt: "reviewing",
  parse: "reading result",
  store: "saving",
};

export function ReviewAgents(props: { state: ReviewState }) {
  const s = () => props.state;
  const agents = (): ReviewAgent[] => s().review()?.agents ?? [];

  return (
    <div class="review-agents">
      <div class="review-panel-head">
        <span class="review-panel-title">agents</span>
      </div>

      <Show when={s().busy() && s().review()}>
        <div class="review-stages">
          <For each={s().review()!.stages}>
            {(stage) => <StageRow stage={stage} now={s().now()} />}
          </For>
        </div>
      </Show>

      <Show when={s().review() && agents().length > 0}>
        <div class="agent-lanes">
          <For each={agents()}>
            {(agent) => (
              <div class={`agent-lane is-${agent.status}`}>
                <div class="agent-lane-head">
                  <span class="agent-lane-name">{agentLabel(agent.name)}</span>
                  <Show when={agent.opencodeSessionPath}>
                    <a
                      class="review-session-link"
                      href={opencodeUrl(agent.opencodeSessionPath)}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {agent.status === "running" ? "watch ↗" : "session ↗"}
                    </a>
                  </Show>
                  <Show when={agent.status === "done" && agent.durationMs > 0}>
                    <span class="review-stage-time">{formatMs(agent.durationMs)}</span>
                  </Show>
                  <AgentButton state={s()} agent={agent} />
                </div>
                <For each={agent.stages}>
                  {(stage) => <StageRow stage={stage} now={s().now()} />}
                </For>
                <Show when={agent.error}>
                  <div class="review-error">{agent.error}</div>
                </Show>
              </div>
            )}
          </For>
        </div>
      </Show>

      <Show when={!s().review()}>
        <p class="review-idle">no agents have run yet</p>
      </Show>
    </div>
  );
}

export function AgentButton(props: { state: ReviewState; agent: ReviewAgent }) {
  const s = () => props.state;

  return (
    <Show
      when={s().agentBusy(props.agent.name)}
      fallback={
        <button
          class="agent-rerun"
          title="re-run this agent"
          aria-label="re-run this agent"
          disabled={s().busy() || !s().review()?.sessionId}
          onClick={() => s().rerun(props.agent.name)}
        >
          <RerunIcon />
        </button>
      }
    >
      <button
        class="agent-rerun is-stop"
        title="stop this agent"
        aria-label="stop this agent"
        onClick={() => s().cancelAgent(props.agent.name)}
      >
        <StopIcon />
      </button>
    </Show>
  );
}

function StageRow(props: { stage: ReviewStage; now: number }) {
  const live = () => {
    if (props.stage.status !== "running" || !props.stage.startedAt) return 0;
    return props.now - Date.parse(props.stage.startedAt);
  };

  return (
    <div class={`review-stage ${props.stage.status}`}>
      <span class="review-stage-dot" />
      <span class="review-stage-name">{stageLabels[props.stage.name] ?? props.stage.name}</span>
      <Show when={props.stage.detail}>
        <span class="review-stage-detail">{props.stage.detail}</span>
      </Show>
      <Show when={props.stage.status === "error"}>
        <span class="review-stage-failed">failed</span>
      </Show>
      <Show when={props.stage.status === "cancelled"}>
        <span class="review-stage-stopped">stopped</span>
      </Show>
      <Show when={props.stage.status === "done" && props.stage.durationMs > 0}>
        <span class="review-stage-time">{formatMs(props.stage.durationMs)}</span>
      </Show>
      <Show when={live() > 0}>
        <span class="review-stage-time">{formatMs(live())}</span>
      </Show>
    </div>
  );
}
