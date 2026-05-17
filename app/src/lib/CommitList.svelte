<script lang="ts">
  import type { Branch, Commit } from "./rpc-types";

  interface Props {
    branches: Branch[];
    sourceBranch: string;
    commits: Commit[];
    selected: Set<string>;
    selectedSha: string;
    loading: boolean;
    refreshing: boolean;
    onsourcebranch: (branch: string) => void;
    ontoggle: (sha: string) => void;
    onselect: (commit: Commit) => void;
    onfetch: () => void;
    onpull: () => void;
  }

  let { branches, sourceBranch, commits, selected, selectedSha, loading, refreshing, onsourcebranch, ontoggle, onselect, onfetch, onpull }: Props = $props();

  let refreshDropdownOpen = $state(false);
  let refreshMode = $state<"fetch" | "pull">("fetch");

  const refreshLabel = $derived(refreshMode === "pull" ? "Pull" : "Fetch");

  function selectRefreshMode(m: "fetch" | "pull") {
    refreshMode = m;
    refreshDropdownOpen = false;
  }

  function executeRefresh() {
    if (refreshMode === "pull") onpull();
    else onfetch();
  }

  function toggleRefreshDropdown() {
    if (!refreshing && !loading) refreshDropdownOpen = !refreshDropdownOpen;
  }

  function onClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest(".refresh-wrap")) refreshDropdownOpen = false;
  }

  function fmt(ts: number): string {
    const d = new Date(ts * 1000);
    return d.toLocaleString(undefined, {
      month: "short", day: "numeric", year: "numeric",
      hour: "2-digit", minute: "2-digit", second: "2-digit",
      hour12: false,
    });
  }

  function shortSha(sha: string) { return sha.slice(0, 7); }
</script>

<svelte:window onclick={onClickOutside} />

<div class="panel">
  <div class="panel-header">
    <label class="label" for="source-branch">Source branch</label>
    <select
      id="source-branch"
      class="branch-select"
      value={sourceBranch}
      onchange={(e) => onsourcebranch((e.target as HTMLSelectElement).value)}
      disabled={branches.length === 0 || refreshing}
    >
      {#each branches as b}
        <option value={b.name}>{b.name}{b.isHead ? " (current)" : ""}</option>
      {/each}
    </select>

    <!-- Fetch / Pull dropdown -->
    <div class="refresh-wrap">
      <button
        class="refresh-btn"
        onclick={executeRefresh}
        disabled={refreshing || loading || branches.length === 0}
        title={refreshMode === "pull" ? "Pull source branch from remote" : "Fetch from remote"}
      >
        {refreshing ? "…" : refreshLabel}
      </button>
      <button
        class="refresh-arrow"
        onclick={toggleRefreshDropdown}
        disabled={refreshing || loading || branches.length === 0}
        aria-label="More refresh options"
      >▾</button>
      {#if refreshDropdownOpen}
        <ul class="refresh-menu">
          <li>
            <button class:active={refreshMode === "fetch"} onclick={() => selectRefreshMode("fetch")}>
              <span class="check">{refreshMode === "fetch" ? "✓" : ""}</span>
              Fetch
            </button>
          </li>
          <li>
            <button class:active={refreshMode === "pull"} onclick={() => selectRefreshMode("pull")}>
              <span class="check">{refreshMode === "pull" ? "✓" : ""}</span>
              Pull
            </button>
          </li>
        </ul>
      {/if}
    </div>
  </div>

  <div class="commit-list">
    {#if loading}
      <div class="empty">Loading commits…</div>
    {:else if commits.length === 0}
      <div class="empty">No commits found.</div>
    {:else}
      {#each commits as c}
        <div
          class="commit-row"
          class:checked={selected.has(c.sha)}
          class:active={selectedSha === c.sha}
          role="row"
          tabindex="0"
          onclick={() => onselect(c)}
          onkeydown={(e) => e.key === "Enter" && onselect(c)}
        >
          <label
            class="cb-wrap"
            onclick={(e) => { e.stopPropagation(); }}
            title={selected.has(c.sha) ? "Bỏ khỏi queue" : "Thêm vào queue"}
          >
            <input
              type="checkbox"
              checked={selected.has(c.sha)}
              onchange={() => ontoggle(c.sha)}
            />
          </label>
          <div class="commit-info">
            <span class="subject" title={c.subject}>{c.subject}</span>
            <span class="meta">
              <span class="sha">{shortSha(c.sha)}</span>
              <span class="author">{c.author}</span>
              <span class="date">{fmt(c.time)}</span>
            </span>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    border-right: 1px solid var(--border, #3a3a3a);
  }
  .panel-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid var(--border, #3a3a3a);
    flex-shrink: 0;
  }
  .label {
    font-size: 0.78rem;
    color: var(--text-secondary, #aaa);
    white-space: nowrap;
  }
  .branch-select {
    flex: 1;
    padding: 0.3rem 0.5rem;
    border-radius: 5px;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #2a2a2a);
    color: var(--text, #f0f0f0);
    font-size: 0.85rem;
    font-family: ui-monospace, monospace;
    min-width: 0;
  }

  /* refresh dropdown */
  .refresh-wrap {
    position: relative;
    display: flex;
    gap: 1px;
    flex-shrink: 0;
  }
  .refresh-btn {
    padding: 0.3rem 0.55rem;
    border-radius: 5px 0 0 5px;
    border: 1px solid var(--border, #555);
    border-right: none;
    background: var(--input-bg, #2a2a2a);
    color: var(--text-secondary, #ccc);
    font-size: 0.8rem;
    cursor: pointer;
    white-space: nowrap;
  }
  .refresh-btn:not(:disabled):hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
  .refresh-arrow {
    padding: 0.3rem 0.35rem;
    border-radius: 0 5px 5px 0;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #2a2a2a);
    color: var(--text-secondary, #ccc);
    font-size: 0.7rem;
    cursor: pointer;
    line-height: 1;
  }
  .refresh-arrow:not(:disabled):hover { background: var(--hover, #3a3a3a); }
  .refresh-btn:disabled, .refresh-arrow:disabled { opacity: 0.4; cursor: not-allowed; }
  .refresh-menu {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    min-width: 120px;
    background: #2c2c2c;
    border: 1px solid var(--border, #3a3a3a);
    border-radius: 7px;
    list-style: none;
    margin: 0;
    padding: 0.3rem 0;
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
    z-index: 100;
  }
  .refresh-menu li button {
    width: 100%;
    padding: 0.45rem 0.75rem;
    background: none;
    border: none;
    color: var(--text, #f0f0f0);
    font-size: 0.85rem;
    text-align: left;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 0.25rem;
    border-radius: 4px;
  }
  .refresh-menu li button:hover { background: var(--hover, #3a3a3a); }
  .refresh-menu li button.active { color: var(--accent, #4a7ef5); }
  .check { width: 1rem; font-size: 0.8rem; flex-shrink: 0; }
  .commit-list {
    flex: 1;
    overflow-y: auto;
    padding: 0.25rem 0;
  }
  .empty {
    padding: 1.5rem;
    text-align: center;
    font-size: 0.85rem;
    color: var(--text-muted, #666);
  }
  .commit-row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    border-bottom: 1px solid var(--border-subtle, #2e2e2e);
    transition: background 0.1s;
  }
  .commit-row:hover { background: var(--hover, #2e2e2e); }
  .commit-row.checked { background: var(--selected, #1a2a4a); }
  .commit-row.active { outline: 1px solid var(--accent, #4a7ef5); outline-offset: -1px; }
  .cb-wrap {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    padding: 6px 10px 6px 2px;
    margin: -6px -10px -6px -2px;
    cursor: pointer;
    border-radius: 4px;
  }
  .cb-wrap:hover { background: rgba(255,255,255,0.06); }
  .cb-wrap input[type="checkbox"] { cursor: pointer; pointer-events: none; }
  .commit-info {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    overflow: hidden;
  }
  .subject {
    font-size: 0.88rem;
    color: var(--text, #f0f0f0);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .meta {
    display: flex;
    gap: 0.5rem;
    font-size: 0.75rem;
    color: var(--text-muted, #888);
    font-family: ui-monospace, monospace;
  }
  .sha { color: var(--accent, #6e9fff); }
</style>
