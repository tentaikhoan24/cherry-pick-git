<script lang="ts">
  import type { Branch, Commit, CherryPickProgress, DryRunItem } from "./rpc-types";

  interface Props {
    queue: Commit[];
    branches: Branch[];
    targetBranch: string;
    sourceBranch: string;
    busy: boolean;
    progress: CherryPickProgress | null;
    dryRunMap: Map<string, DryRunItem>;
    ontargetbranch: (branch: string) => void;
    onremove: (sha: string) => void;
    onapply: () => void;
    onapplypush: () => void;
    oncancel: () => void;
    oncreate: (name: string, base: string) => void;
  }

  let { queue, branches, targetBranch, sourceBranch, busy, progress, dryRunMap, ontargetbranch, onremove, onapply, onapplypush, oncancel, oncreate }: Props = $props();

  const canApply = $derived(queue.length > 0 && targetBranch !== sourceBranch && !busy);

  let dropdownOpen = $state(false);
  let mode = $state<"apply" | "apply-push">("apply");

  const btnLabel = $derived(
    mode === "apply-push"
      ? `Apply & Push ${queue.length > 0 ? queue.length : ""} commit${queue.length === 1 ? "" : "s"} → ${targetBranch}`
      : `Apply ${queue.length > 0 ? queue.length : ""} commit${queue.length === 1 ? "" : "s"} → ${targetBranch}`
  );

  // ── create branch inline form ─────────────────────────────
  let creatingBranch = $state(false);
  let newBranchName = $state("");
  let newBranchInput: HTMLInputElement | undefined = $state();

  function openCreateForm() {
    newBranchName = "";
    creatingBranch = true;
    // focus after DOM update
    setTimeout(() => newBranchInput?.focus(), 0);
  }

  function cancelCreate() {
    creatingBranch = false;
    newBranchName = "";
  }

  function confirmCreate() {
    const name = newBranchName.trim();
    if (!name) return;
    oncreate(name, targetBranch);
    creatingBranch = false;
    newBranchName = "";
  }

  function onCreateKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") confirmCreate();
    else if (e.key === "Escape") cancelCreate();
  }

  function shortSha(sha: string) { return sha.slice(0, 7); }

  function selectMode(m: "apply" | "apply-push") {
    mode = m;
    dropdownOpen = false;
  }

  function execute() {
    if (mode === "apply-push") onapplypush();
    else onapply();
  }

  function toggleDropdown() {
    if (canApply) dropdownOpen = !dropdownOpen;
  }

  function onClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest(".dropdown-wrap")) dropdownOpen = false;
  }
</script>

<svelte:window onclick={onClickOutside} />

<div class="panel">
  <div class="panel-header">
    <label class="label" for="target-branch">Target branch</label>
    <select
      id="target-branch"
      class="branch-select"
      value={targetBranch}
      onchange={(e) => ontargetbranch((e.target as HTMLSelectElement).value)}
      disabled={branches.length === 0 || busy || creatingBranch}
    >
      {#each branches as b}
        <option value={b.name}>{b.name}{b.isHead ? " (current)" : ""}</option>
      {/each}
    </select>
    <button
      class="new-branch-btn"
      onclick={openCreateForm}
      disabled={busy || creatingBranch}
      title="Create new branch from {targetBranch}"
    >+</button>
  </div>

  {#if creatingBranch}
    <div class="create-branch-form">
      <input
        bind:this={newBranchInput}
        class="branch-name-input"
        type="text"
        placeholder="new-branch-name"
        bind:value={newBranchName}
        onkeydown={onCreateKeydown}
        spellcheck={false}
      />
      <span class="from-label">from {targetBranch}</span>
      <button class="create-confirm-btn" onclick={confirmCreate} disabled={!newBranchName.trim()}>
        Create
      </button>
      <button class="create-cancel-btn" onclick={cancelCreate}>×</button>
    </div>
  {/if}

  <div class="queue-body">
    {#if queue.length === 0}
      <div class="empty">
        <p>No commits selected.</p>
        <p class="hint">Tick commits on the left to add them here.</p>
      </div>
    {:else}
      <div class="queue-header">
        Pick queue — {queue.length} commit{queue.length === 1 ? "" : "s"}
      </div>
      <ul class="queue-list">
        {#each queue as c, i}
          {@const dryRun = dryRunMap.get(c.sha)}
          <li class="queue-item" class:conflict={dryRun?.willConflict}>
            <span class="order">{i + 1}</span>
            <div class="commit-info">
              <span class="subject" title={c.subject}>{c.subject}</span>
              <span class="sha">{shortSha(c.sha)}</span>
            </div>
            {#if dryRun?.willConflict}
              <span class="conflict-icon" title="Predicted conflict: {dryRun.files.length > 0 ? dryRun.files.join(', ') : 'unknown files'}">⚠</span>
            {/if}
            <button class="remove-btn" onclick={() => onremove(c.sha)} title="Remove" disabled={busy}>✕</button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  <div class="panel-footer">
    {#if targetBranch === sourceBranch && branches.length > 0}
      <p class="warn">Source and target branch are the same.</p>
    {/if}

    {#if busy}
      <!-- Progress row -->
      <div class="progress-row">
        <div class="progress-bar-wrap">
          <div
            class="progress-bar-fill"
            style="width: {progress ? Math.round((progress.n / progress.total) * 100) : 0}%"
          ></div>
        </div>
        <span class="progress-label">
          {#if progress}
            {progress.n}/{progress.total} — {shortSha(progress.sha)}
          {:else}
            Preparing…
          {/if}
        </span>
        <button class="cancel-btn" onclick={oncancel}>Cancel</button>
      </div>
    {:else}
      <div class="dropdown-wrap">
        <!-- Main button -->
        <button class="apply-btn" onclick={execute} disabled={!canApply}>
          {btnLabel}
        </button>
        <!-- Arrow toggle -->
        <button class="arrow-btn" onclick={toggleDropdown} disabled={!canApply} aria-label="More options">
          ▾
        </button>
        <!-- Dropdown menu -->
        {#if dropdownOpen}
          <ul class="dropdown-menu">
            <li>
              <button class:active={mode === "apply"} onclick={() => selectMode("apply")}>
                {#if mode === "apply"}<span class="check">✓</span>{:else}<span class="check"></span>{/if}
                Apply
              </button>
            </li>
            <li>
              <button class:active={mode === "apply-push"} onclick={() => selectMode("apply-push")}>
                {#if mode === "apply-push"}<span class="check">✓</span>{:else}<span class="check"></span>{/if}
                Apply &amp; Push
              </button>
            </li>
          </ul>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }
  .panel-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid var(--border, #3a3a3a);
    flex-shrink: 0;
  }
  .new-branch-btn {
    flex-shrink: 0;
    width: 1.7rem;
    height: 1.7rem;
    border-radius: 5px;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #2a2a2a);
    color: var(--text-secondary, #ccc);
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .new-branch-btn:not(:disabled):hover { background: var(--hover, #3a3a3a); color: var(--accent, #4a7ef5); }
  .new-branch-btn:disabled { opacity: 0.35; cursor: not-allowed; }

  /* inline create-branch form */
  .create-branch-form {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.45rem 0.75rem;
    background: var(--hover, #272727);
    border-bottom: 1px solid var(--accent, #4a7ef5);
    flex-shrink: 0;
  }
  .branch-name-input {
    flex: 1;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    border: 1px solid var(--accent, #4a7ef5);
    background: var(--input-bg, #1e1e1e);
    color: var(--text, #f0f0f0);
    font-size: 0.85rem;
    font-family: ui-monospace, monospace;
    outline: none;
    min-width: 0;
  }
  .from-label {
    flex-shrink: 0;
    font-size: 0.73rem;
    font-family: ui-monospace, monospace;
    color: var(--text-muted, #666);
    white-space: nowrap;
  }
  .create-confirm-btn {
    flex-shrink: 0;
    padding: 0.25rem 0.65rem;
    border-radius: 4px;
    border: none;
    background: var(--accent, #4a7ef5);
    color: #fff;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
  }
  .create-confirm-btn:disabled { opacity: 0.4; cursor: not-allowed; }
  .create-confirm-btn:not(:disabled):hover { opacity: 0.85; }
  .create-cancel-btn {
    flex-shrink: 0;
    background: none;
    border: none;
    color: var(--text-muted, #666);
    font-size: 1rem;
    cursor: pointer;
    padding: 0.1rem 0.3rem;
    line-height: 1;
    border-radius: 3px;
  }
  .create-cancel-btn:hover { color: #ef5350; }

  .label {
    font-size: 0.78rem;
    color: var(--text-secondary, #aaa);
    white-space: nowrap;
  }
  .branch-select {
    flex: 1;
    min-width: 0;
    padding: 0.3rem 0.5rem;
    border-radius: 5px;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #2a2a2a);
    color: var(--text, #f0f0f0);
    font-size: 0.85rem;
    font-family: ui-monospace, monospace;
  }
  .queue-body {
    flex: 1;
    overflow-y: auto;
    padding: 0.25rem 0;
  }
  .empty {
    padding: 1.5rem;
    text-align: center;
  }
  .empty p { margin: 0.25rem 0; font-size: 0.9rem; color: var(--text-muted, #666); }
  .empty .hint { font-size: 0.8rem; }
  .queue-header {
    padding: 0.4rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-secondary, #aaa);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    border-bottom: 1px solid var(--border-subtle, #2e2e2e);
  }
  .queue-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .queue-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.45rem 0.75rem;
    border-bottom: 1px solid var(--border-subtle, #2e2e2e);
  }
  .order {
    flex-shrink: 0;
    width: 1.5rem;
    text-align: right;
    font-size: 0.75rem;
    color: var(--text-muted, #666);
    font-family: ui-monospace, monospace;
  }
  .commit-info {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }
  .subject {
    font-size: 0.85rem;
    color: var(--text, #f0f0f0);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .sha {
    font-size: 0.73rem;
    font-family: ui-monospace, monospace;
    color: var(--accent, #6e9fff);
  }
  .queue-item.conflict {
    background: rgba(255, 152, 0, 0.06);
    border-left: 2px solid #ffa726;
  }
  .conflict-icon {
    flex-shrink: 0;
    font-size: 0.82rem;
    color: #ffa726;
    cursor: default;
  }
  .remove-btn {
    flex-shrink: 0;
    background: none;
    border: none;
    color: var(--text-muted, #666);
    cursor: pointer;
    font-size: 0.8rem;
    padding: 0.2rem 0.3rem;
    border-radius: 4px;
    line-height: 1;
  }
  .remove-btn:hover:not(:disabled) { color: #ef5350; background: rgba(239,83,80,0.1); }
  .remove-btn:disabled { cursor: default; opacity: 0.3; }

  /* footer */
  .panel-footer {
    padding: 0.75rem;
    border-top: 1px solid var(--border, #3a3a3a);
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .warn {
    margin: 0;
    font-size: 0.78rem;
    color: #f4a261;
    text-align: center;
  }

  /* progress */
  .progress-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .progress-bar-wrap {
    flex: 1;
    height: 6px;
    border-radius: 3px;
    background: rgba(255,255,255,0.1);
    overflow: hidden;
  }
  .progress-bar-fill {
    height: 100%;
    border-radius: 3px;
    background: var(--accent, #4a7ef5);
    transition: width 0.2s ease;
  }
  .progress-label {
    flex-shrink: 0;
    font-size: 0.75rem;
    font-family: ui-monospace, monospace;
    color: var(--text-secondary, #aaa);
    white-space: nowrap;
  }
  .cancel-btn {
    flex-shrink: 0;
    padding: 0.3rem 0.65rem;
    border-radius: 5px;
    border: 1px solid #ef5350;
    background: transparent;
    color: #ef5350;
    font-size: 0.8rem;
    cursor: pointer;
  }
  .cancel-btn:hover { background: rgba(239,83,80,0.15); }

  /* dropdown button */
  .dropdown-wrap {
    position: relative;
    display: flex;
    gap: 2px;
  }
  .apply-btn {
    flex: 1;
    padding: 0.6rem 0.75rem;
    border-radius: 7px 0 0 7px;
    border: none;
    background: var(--accent, #396cd8);
    color: #fff;
    font-size: 0.88rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
    text-align: left;
  }
  .arrow-btn {
    flex-shrink: 0;
    padding: 0.6rem 0.7rem;
    border-radius: 0 7px 7px 0;
    border: none;
    border-left: 1px solid rgba(255,255,255,0.2);
    background: var(--accent, #396cd8);
    color: #fff;
    font-size: 0.8rem;
    cursor: pointer;
    transition: opacity 0.15s;
    line-height: 1;
  }
  .apply-btn:disabled,
  .arrow-btn:disabled { opacity: 0.35; cursor: not-allowed; }
  .apply-btn:not(:disabled):hover,
  .arrow-btn:not(:disabled):hover { opacity: 0.85; }

  .dropdown-menu {
    position: absolute;
    bottom: calc(100% + 4px);
    right: 0;
    min-width: 160px;
    background: #2c2c2c;
    border: 1px solid var(--border, #3a3a3a);
    border-radius: 7px;
    list-style: none;
    margin: 0;
    padding: 0.3rem 0;
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
    z-index: 100;
  }
  .dropdown-menu li button {
    width: 100%;
    padding: 0.5rem 1rem;
    background: none;
    border: none;
    color: var(--text, #f0f0f0);
    font-size: 0.88rem;
    text-align: left;
    cursor: pointer;
    border-radius: 4px;
  }
  .dropdown-menu li button:hover {
    background: var(--hover, #3a3a3a);
  }
  .dropdown-menu li button.active {
    color: var(--accent, #4a7ef5);
  }
  .check {
    display: inline-block;
    width: 1rem;
    font-size: 0.8rem;
  }
</style>
