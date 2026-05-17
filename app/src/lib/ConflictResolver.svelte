<script lang="ts">
  import type { ConflictFileInfo, Commit, DryRunItem } from "./rpc-types";

  interface Props {
    files: ConflictFileInfo[];
    conflictSha: string;
    queue: Commit[];
    dryRunMap: Map<string, DryRunItem>;
    remainingCommitFiles: Map<string, string[]>;
    busy: boolean;
    resolvedSet: Set<string>;
    onresolve: (file: string, strategy: "ours" | "theirs") => void;
    oncontinue: () => void;
    onabort: () => void;
    onviewfile?: (file: string) => void;
    onviewdiff?: (file: string) => void;
  }

  let { files, conflictSha, queue, dryRunMap, remainingCommitFiles, busy, resolvedSet, onresolve, oncontinue, onabort, onviewfile, onviewdiff }: Props = $props();

  const resolvedCount = $derived([...resolvedSet].filter(p => files.some(f => f.path === p)).length);
  const allResolved = $derived(files.length > 0 && resolvedCount === files.length);

  // Find current conflicting commit in queue (match full or prefix SHA)
  const currentCommit = $derived(
    queue.find(c => c.sha === conflictSha || c.sha.startsWith(conflictSha) || conflictSha.startsWith(c.sha))
  );
  const shortSha = $derived(conflictSha ? conflictSha.slice(0, 7) : "");

  // All remaining commits in queue after the current conflicting one
  const currentIdx = $derived(
    currentCommit ? queue.findIndex(c => c.sha === currentCommit.sha) : -1
  );
  const remainingCommits = $derived(
    currentIdx >= 0 ? queue.slice(currentIdx + 1) : []
  );

  const statusLabel: Record<string, string> = {
    UU: "modified", AA: "added", DD: "deleted",
    AU: "added/modified", UA: "modified/added",
    DU: "deleted/modified", UD: "modified/deleted",
  };
</script>

<div class="resolver">
  <div class="resolver-header">
    <span class="icon">⚠</span>
    <span class="title">Cherry-pick paused — conflicts</span>
    <span class="hint">Ours = target branch &nbsp;·&nbsp; Theirs = cherry-picked commit</span>
  </div>

  <div class="commit-sections">

    <!-- ── Current conflicting commit (actionable) ── -->
    <div class="commit-section">
      <div class="commit-row">
        <span class="cdot cdot-conflict">●</span>
        {#if currentCommit}
          <span class="commit-subject">{currentCommit.subject}</span>
        {/if}
        <code class="sha-badge">{shortSha}</code>
        <span class="cbadge cbadge-conflict">conflict</span>
      </div>

      {#each files as f}
        {@const resolved = resolvedSet.has(f.path)}
        <div class="file-row" class:resolved>
          <span class="file-tree-line"></span>
          <span class="status-label" class:resolved-label={resolved}>{statusLabel[f.status] ?? f.status}</span>
          {#if resolved}
            {#if onviewdiff}
              <button class="file-path-btn file-path-btn-resolved" title="View staged diff: {f.path}" onclick={() => onviewdiff!(f.path)}>{f.path}</button>
            {:else}
              <span class="file-path">{f.path}</span>
            {/if}
          {:else if onviewfile}
            <button class="file-path-btn" title="Open merge editor: {f.path}" onclick={() => onviewfile!(f.path)}>{f.path}</button>
          {:else}
            <span class="file-path">{f.path}</span>
          {/if}
          {#if resolved}
            <span class="resolved-badge">✓ resolved</span>
          {:else}
            <button class="resolve-btn ours"   onclick={() => onresolve(f.path, "ours")}   disabled={busy}>Keep Ours</button>
            <button class="resolve-btn theirs" onclick={() => onresolve(f.path, "theirs")} disabled={busy}>Use Theirs</button>
          {/if}
        </div>
      {/each}
    </div>

    <!-- ── Remaining commits in queue (informational) ── -->
    {#each remainingCommits as c}
      {@const dryRun = dryRunMap.get(c.sha)}
      {@const willConflict = dryRun?.willConflict ?? false}
      {@const touchedFiles = remainingCommitFiles.get(c.sha) ?? []}
      <div class="commit-section upcoming">
        <div class="commit-row">
          <span class="cdot" class:cdot-upcoming-conflict={willConflict} class:cdot-upcoming={!willConflict}>
            {willConflict ? "⚠" : "○"}
          </span>
          <span class="commit-subject upcoming-subject">{c.subject}</span>
          <code class="sha-badge sha-upcoming">{c.sha.slice(0, 7)}</code>
          {#if willConflict}
            <span class="cbadge cbadge-upcoming">predicted conflict</span>
          {:else}
            <span class="cbadge cbadge-queued">queued</span>
          {/if}
        </div>
        <!-- Show files this commit touches — may conflict when git gets to it -->
        {#each touchedFiles as f}
          <div class="file-row upcoming-file-row">
            <span class="file-tree-line"></span>
            <span class="status-label upcoming-status-label">
              {willConflict ? "conflict" : "modifies"}
            </span>
            <span class="file-path upcoming-file-path">{f}</span>
          </div>
        {/each}
      </div>
    {/each}

  </div>

  <div class="resolver-footer">
    <span class="progress-text">
      {resolvedCount}/{files.length} file{files.length === 1 ? "" : "s"} resolved
      {#if remainingCommits.length > 0}
        <span class="upcoming-hint">· còn {remainingCommits.length} commit tiếp theo</span>
      {/if}
    </span>
    <div class="actions">
      <button class="abort-btn" onclick={onabort} disabled={busy}>Abort</button>
      <button
        class="continue-btn"
        onclick={oncontinue}
        disabled={!allResolved || busy}
        title={allResolved ? "Continue cherry-pick" : "Resolve all conflicts first"}
      >
        {busy ? "Working…" : "Continue →"}
      </button>
    </div>
  </div>
</div>

<style>
  .resolver {
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    border-top: 2px solid #ffa726;
    background: #1e1a14;
    max-height: 300px;
  }

  /* ── Header ── */
  .resolver-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem;
    background: rgba(255, 167, 38, 0.08);
    border-bottom: 1px solid rgba(255, 167, 38, 0.2);
    flex-shrink: 0;
  }
  .icon  { color: #ffa726; font-size: 0.9rem; flex-shrink: 0; }
  .title { font-size: 0.82rem; font-weight: 600; color: #ffa726; flex-shrink: 0; }
  .hint  { font-size: 0.72rem; color: #666; margin-left: auto; white-space: nowrap; }

  /* ── Commit sections ── */
  .commit-sections {
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }

  .commit-section {
    border-bottom: 1px solid #2a2418;
    padding-bottom: 0.15rem;
  }
  .commit-section.upcoming {
    opacity: 0.55;
  }

  /* Commit header row */
  .commit-row {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.75rem 0.2rem;
  }
  .cdot {
    font-size: 0.65rem;
    flex-shrink: 0;
    width: 10px;
    text-align: center;
  }
  .cdot-conflict          { color: #ef5350; }
  .cdot-upcoming-conflict { color: #ffa726; font-size: 0.6rem; }
  .cdot-upcoming          { color: #555; }

  .commit-subject {
    font-size: 0.8rem;
    font-weight: 600;
    color: #e0e0e0;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .upcoming-subject { color: #888; font-weight: 400; }

  .sha-badge {
    font-family: ui-monospace, monospace;
    font-size: 0.72rem;
    color: #4a7ef5;
    background: rgba(74, 126, 245, 0.12);
    padding: 0.05rem 0.4rem;
    border-radius: 4px;
    flex-shrink: 0;
  }
  .sha-upcoming { color: #666; background: rgba(100,100,100,0.1); }

  .cbadge {
    font-size: 0.65rem;
    padding: 0.05rem 0.35rem;
    border-radius: 3px;
    flex-shrink: 0;
    font-family: ui-monospace, monospace;
  }
  .cbadge-conflict { background: rgba(239,83,80,0.15);  color: #ef5350; }
  .cbadge-upcoming { background: rgba(255,167,38,0.1);  color: #ffa726; }
  .cbadge-queued   { background: rgba(100,100,100,0.1); color: #555; }

  /* ── File rows ── */
  .file-row {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.22rem 0.75rem 0.22rem 0;
    border-bottom: 1px solid rgba(255,255,255,0.03);
    transition: background 0.1s;
  }
  .file-row.resolved { opacity: 0.45; }
  .file-row:last-child { border-bottom: none; }
  .upcoming-file-row { opacity: 1; } /* opacity inherited from .upcoming already */

  /* Tree indent line */
  .file-tree-line {
    flex-shrink: 0;
    width: 22px;
    height: 100%;
    border-left: 1px solid #333;
    margin-left: 12px;
    margin-right: 4px;
  }

  .status-label {
    flex-shrink: 0;
    font-size: 0.65rem;
    font-family: ui-monospace, monospace;
    color: #ffa726;
    background: rgba(255,167,38,0.12);
    padding: 0.04rem 0.3rem;
    border-radius: 3px;
    min-width: 4rem;
    text-align: center;
  }
  .resolved-label  { color: #66bb6a; background: rgba(102,187,106,0.12); }
  .upcoming-status-label { color: #666; background: rgba(100,100,100,0.1); }

  .file-path {
    flex: 1;
    font-family: ui-monospace, monospace;
    font-size: 0.78rem;
    color: #d0d0d0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .upcoming-file-path { color: #666; }

  .file-path-btn {
    flex: 1;
    font-family: ui-monospace, monospace;
    font-size: 0.78rem;
    color: #4a7ef5;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .file-path-btn:hover { color: #7aaeff; }
  .file-path-btn-resolved { color: #888; }
  .file-path-btn-resolved:hover { color: #aaa; }

  .resolved-badge {
    flex-shrink: 0;
    font-size: 0.72rem;
    color: #66bb6a;
    font-family: ui-monospace, monospace;
  }

  .resolve-btn {
    flex-shrink: 0;
    padding: 0.18rem 0.5rem;
    border-radius: 4px;
    border: 1px solid;
    font-size: 0.72rem;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .resolve-btn:disabled { opacity: 0.35; cursor: not-allowed; }
  .resolve-btn.ours {
    border-color: #444;
    background: #2a2a2a;
    color: #ccc;
  }
  .resolve-btn.ours:not(:disabled):hover { background: #3a3a3a; }
  .resolve-btn.theirs {
    border-color: rgba(74,126,245,0.5);
    background: rgba(74,126,245,0.1);
    color: #4a7ef5;
  }
  .resolve-btn.theirs:not(:disabled):hover { background: rgba(74,126,245,0.2); }

  /* ── Footer ── */
  .resolver-footer {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.45rem 0.75rem;
    border-top: 1px solid #2e2818;
    flex-shrink: 0;
  }
  .progress-text {
    font-size: 0.75rem;
    color: #888;
    font-family: ui-monospace, monospace;
  }
  .upcoming-hint { color: #665530; }

  .actions { display: flex; gap: 0.5rem; margin-left: auto; }

  .abort-btn {
    padding: 0.3rem 0.8rem;
    border-radius: 5px;
    border: 1px solid #ef5350;
    background: transparent;
    color: #ef5350;
    font-size: 0.8rem;
    cursor: pointer;
  }
  .abort-btn:not(:disabled):hover { background: rgba(239,83,80,0.12); }
  .abort-btn:disabled { opacity: 0.4; cursor: not-allowed; }

  .continue-btn {
    padding: 0.3rem 0.9rem;
    border-radius: 5px;
    border: none;
    background: #396cd8;
    color: #fff;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .continue-btn:disabled { opacity: 0.35; cursor: not-allowed; }
  .continue-btn:not(:disabled):hover { opacity: 0.85; }
</style>
