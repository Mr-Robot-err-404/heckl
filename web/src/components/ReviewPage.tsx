import { For, Show } from "solid-js";
import { agentLabel, formatMs, opencodeUrl, type ReviewState } from "../review";
import { AgentPicker } from "./AgentPicker";
import { RerunIcon } from "./icons";
import type { RankedConcern, ReviewAgent, ReviewStage } from "../types";

type Props = {
  state: ReviewState;
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

export function ReviewPage(props: Props) {
  const s = () => props.state;

  const agents = (): ReviewAgent[] => s().review()?.agents ?? [];

  const isRerun = () => s().synced() && !s().busy() && !!s().review();

  const runLabel = () =>
    !s().synced() ? "connecting..." : s().busy() ? "reviewing..." : s().review() ? "re-run" : "run";

  const grouped = () => {
    const known = agents();
    const orphaned = s()
      .concerns()
      .map((c) => c.agent)
      .filter((name, i, all) => all.indexOf(name) === i)
      .filter((name) => !known.some((a) => a.name === name))
      .map((name): ReviewAgent => ({ name, status: "done", stages: [], durationMs: 0 }));

    return [...known, ...orphaned].map((agent) => ({
      agent,
      concerns: s()
        .concerns()
        .filter((c) => c.agent === agent.name),
    }));
  };

  return (
    <div class="review-page">
      <div class="review-page-head">
        <div class="review-head-start">
          <Show when={s().review()?.opencodeSessionPath}>
            <a
              class="review-session-link"
              href={opencodeUrl(s().review()?.opencodeSessionPath)}
              target="_blank"
              rel="noreferrer"
            >
              continue in opencode ↗
            </a>
          </Show>
        </div>
        <div class="review-head-end">
          <button
            class="review-run"
            title={runLabel()}
            aria-label={runLabel()}
            onClick={() => s().start()}
            disabled={!s().canRun()}
          >
            <Show when={isRerun()} fallback={runLabel()}>
              <RerunIcon />
            </Show>
          </button>
        </div>
      </div>

      <AgentPicker
        state={s()}
        meta={
          <Show when={s().review()?.status === "done"}>
            <span class="review-meta">
              {s().concerns().length === 0
                ? "no concerns"
                : `${s().concerns().length} concern${s().concerns().length === 1 ? "" : "s"}`}
              <Show when={s().elapsed() > 0}>
                {" · "}
                {formatMs(s().elapsed())}
              </Show>
            </span>
          </Show>
        }
      />

      <Show when={s().error()}>
        <div class="review-error">{s().error()}</div>
      </Show>

      <Show when={s().synced() && !s().connected()}>
        <div class="review-stale">connection lost - reconnecting</div>
      </Show>

      <Show when={!s().synced()}>
        <p class="review-idle">syncing with server...</p>
      </Show>

      <Show when={s().synced() && !s().busy() && !s().review()}>
        <p class="review-idle">no review yet - run one to see the agent's read on this PR</p>
      </Show>

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
                  <button
                    class="agent-rerun"
                    title="re-run this agent"
                    aria-label="re-run this agent"
                    disabled={s().busy() || !s().review()?.sessionId}
                    onClick={() => s().rerun(agent.name)}
                  >
                    <RerunIcon />
                  </button>
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

      <Show when={s().review()?.status === "error"}>
        <div class="review-error">{s().review()?.error}</div>
      </Show>

      <Show when={s().review()?.status === "done"}>
        <Show when={s().review()?.summary}>
          <section class="review-synopsis">
            <h3 class="review-section-title">synopsis</h3>
            <p>{s().review()?.summary}</p>
          </section>
        </Show>

        <For each={grouped()}>
          {(group) => (
            <section class="review-findings">
              <h3 class="review-section-title">{agentLabel(group.agent.name)}</h3>
              <Show
                when={group.agent.status !== "error"}
                fallback={<p class="review-failed">this agent failed</p>}
              >
                <Show
                  when={group.concerns.length > 0}
                  fallback={<p class="review-clear">nothing worth flagging</p>}
                >
                  <For each={group.concerns}>
                    {(concern) => (
                      <ConcernCard
                        concern={concern}
                        onFocus={() => props.onFocusConcern(concern)}
                      />
                    )}
                  </For>
                </Show>
              </Show>
            </section>
          )}
        </For>
      </Show>
    </div>
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
      <Show when={props.stage.status === "done" && props.stage.durationMs > 0}>
        <span class="review-stage-time">{formatMs(props.stage.durationMs)}</span>
      </Show>
      <Show when={live() > 0}>
        <span class="review-stage-time">{formatMs(live())}</span>
      </Show>
    </div>
  );
}

function ConcernCard(props: { concern: RankedConcern; onFocus: () => void }) {
  const locatable = () => props.concern.line != null && props.concern.side != null;

  return (
    <article class={`review-concern sev-${props.concern.severity}`}>
      <div class="review-concern-head">
        <span class="review-concern-title">{props.concern.title}</span>
        <span class="review-concern-sev">{props.concern.severity}</span>
      </div>
      <Show
        when={locatable()}
        fallback={
          <div class="review-concern-loc">
            {props.concern.file} <span class="concern-unpinned">· not pinned to a line</span>
          </div>
        }
      >
        <button class="review-concern-loc link" onClick={props.onFocus}>
          {props.concern.file}:{props.concern.line} →
        </button>
      </Show>
      <p class="review-concern-body">{props.concern.body}</p>
    </article>
  );
}
