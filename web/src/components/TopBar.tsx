import { createEffect, createSignal, For, onMount, Show } from "solid-js";
import { useNavigate, useParams } from "@tanstack/solid-router";
import { errText } from "../api";
import {
  useAddRepo,
  useOrgs,
  useRepos,
  usePRDetail,
  usePrefetch,
  resolved,
  historyPageSize,
} from "../queries";
import { useActiveReviews } from "../activeReviews";
import { AgentConfigModal } from "./AgentConfigModal";
import { BotIcon, PaletteIcon } from "./icons";
import { ThemeModal } from "./ThemeModal";
import { useModal } from "../modal";

export function TopBar() {
  const activeReviews = useActiveReviews();
  const orgs = useOrgs();
  const repos = useRepos();
  const addRepo = useAddRepo();
  const prefetch = usePrefetch();
  const navigate = useNavigate();
  const [input, setInput] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);
  const [adding, setAdding] = createSignal(false);
  const modal = useModal();

  onMount(() => prefetch.agentConfig());

  const params = useParams({ strict: false });
  const selectedKey = () => {
    const p = params();
    return p.owner && p.repo ? `${p.owner}/${p.repo}` : "";
  };
  const prNumber = () => {
    const n = parseInt(params().pr ?? "", 10);
    return isNaN(n) ? null : n;
  };

  const prDetail = usePRDetail(
    () => params().owner ?? "",
    () => params().repo ?? "",
    () => prNumber(),
  );

  let selectRef: HTMLSelectElement | undefined;

  const orgData = resolved(orgs);
  const repoData = resolved(repos);
  const detailData = resolved(prDetail);

  const reposByOrg = () => {
    const orgList = orgData() ?? [];
    const repoList = repoData() ?? [];
    return orgList.map((org) => ({
      org,
      repos: repoList.filter((r) => r.Owner === org),
    }));
  };

  const orphan = () => {
    const key = selectedKey();
    if (!key) return null;
    const known = (repoData() ?? []).some((r) => `${r.Owner}/${r.Name}` === key);
    return known ? null : key;
  };

  createEffect(() => {
    const key = selectedKey();
    reposByOrg();
    orphan();
    if (selectRef) selectRef.value = key;
  });

  function handleRepoChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value;
    if (val === "__add__") {
      setAdding(true);
      return;
    }
    const [owner, repo] = val.split("/");
    navigate({ to: "/$owner/$repo", params: { owner, repo } });
  }

  function handleAdd() {
    const parts = input().trim().split("/");
    if (parts.length !== 2 || !parts[0] || !parts[1]) {
      setError("owner/repo");
      return;
    }
    addRepo.mutate(
      { owner: parts[0], name: parts[1] },
      {
        onSuccess: (repo) => {
          setInput("");
          setError(null);
          setAdding(false);
          navigate({ to: "/$owner/$repo", params: { owner: repo.Owner, repo: repo.Name } });
        },
        onError: (e) => setError(errText(e)),
      },
    );
  }

  return (
    <header class="topbar">
      <div class="topbar-left">
        <button
          class="topbar-brand"
          onMouseEnter={() => prefetch.historyPage(0)}
          onClick={() => navigate({ to: "/", search: { page: 0 } })}
        >
          heckl
          <Show when={activeReviews.count() > 0}>
            <span class="brand-badge">{activeReviews.count()}</span>
          </Show>
        </button>
        <Show
          when={!adding()}
          fallback={
            <div class="repo-input-wrap">
              <input
                class="repo-input"
                placeholder="owner/repo"
                value={input()}
                autofocus
                onInput={(e) => setInput(e.currentTarget.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleAdd();
                  if (e.key === "Escape") {
                    setAdding(false);
                    setInput("");
                    setError(null);
                  }
                }}
              />
              <button class="topbar-btn" onClick={handleAdd}>
                add
              </button>
              <button
                class="topbar-btn"
                onClick={() => {
                  setAdding(false);
                  setInput("");
                  setError(null);
                }}
              >
                cancel
              </button>
              <Show when={error()}>
                <span class="input-error">{error()}</span>
              </Show>
            </div>
          }
        >
          <select
            ref={selectRef}
            class="repo-select"
            value={selectedKey()}
            onChange={handleRepoChange}
          >
            <option value="" disabled>
              select repo
            </option>
            <Show when={orphan()}>{(key) => <option value={key()}>{key()}</option>}</Show>
            <For each={reposByOrg()}>
              {(group) => (
                <optgroup label={group.org}>
                  <For each={group.repos}>
                    {(repo) => (
                      <option
                        value={`${repo.Owner}/${repo.Name}`}
                        onMouseEnter={() => {
                          prefetch.prs(repo.Owner, repo.Name);
                          prefetch.repoHistory(repo.Owner, repo.Name, historyPageSize);
                        }}
                      >
                        {repo.Name}
                      </option>
                    )}
                  </For>
                </optgroup>
              )}
            </For>
            <option value="__add__">+ add repo</option>
          </select>
        </Show>
      </div>
      <Show when={detailData()} keyed>
        {(d) => <span class="topbar-pr-title">{d.pr.Title}</span>}
      </Show>
      <div class="topbar-right">
        <Show when={detailData()} keyed>
          {(d) => {
            const additions = d.files.reduce((n, f) => n + f.Additions, 0);
            const deletions = d.files.reduce((n, f) => n + f.Deletions, 0);
            return (
              <>
                <span class="additions">+{additions}</span>
                <span class="deletions">-{deletions}</span>
                <span class="muted">{d.files.length} files</span>
                <span class="muted">{d.pr.Author}</span>
              </>
            );
          }}
        </Show>
        <button class="topbar-btn topbar-config" title="theme" onClick={() => modal.open("theme")}>
          <PaletteIcon />
        </button>
        <button
          class="topbar-btn topbar-config"
          title="agent config"
          onClick={() => modal.open("agents")}
        >
          <BotIcon />
        </button>
      </div>
      <Show when={modal.isOpen("agents")}>
        <AgentConfigModal onClose={modal.close} />
      </Show>
      <Show when={modal.isOpen("theme")}>
        <ThemeModal onClose={modal.close} />
      </Show>
    </header>
  );
}
