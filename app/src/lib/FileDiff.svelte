<script lang="ts">
  import type { CommitFile } from "./rpc-types";

  interface DiffLine {
    type: "hunk" | "add" | "del" | "ctx" | "meta";
    oldNo: number | null;
    newNo: number | null;
    content: string;
  }

  interface Props {
    diff: string;
    file: CommitFile | null;
    loading: boolean;
    onback: () => void;
  }

  let { diff, file, loading, onback }: Props = $props();

  const statusColor: Record<string, string> = {
    A: "#66bb6a", D: "#ef5350", M: "#ffa726",
    R: "#42a5f5", C: "#ab47bc",
  };

  function parseDiff(raw: string): DiffLine[] {
    if (!raw) return [];
    const result: DiffLine[] = [];
    let oldNo = 0;
    let newNo = 0;
    for (const line of raw.split("\n")) {
      if (line.startsWith("@@")) {
        const m = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
        if (m) { oldNo = parseInt(m[1], 10); newNo = parseInt(m[2], 10); }
        result.push({ type: "hunk", oldNo: null, newNo: null, content: line });
      } else if (line.startsWith("+") && !line.startsWith("+++")) {
        result.push({ type: "add", oldNo: null, newNo: newNo++, content: line.slice(1) });
      } else if (line.startsWith("-") && !line.startsWith("---")) {
        result.push({ type: "del", oldNo: oldNo++, newNo: null, content: line.slice(1) });
      } else if (
        line.startsWith("diff ") || line.startsWith("index ") ||
        line.startsWith("--- ") || line.startsWith("+++ ") ||
        line.startsWith("new file") || line.startsWith("deleted file")
      ) {
        result.push({ type: "meta", oldNo: null, newNo: null, content: line });
      } else if (line !== "") {
        const text = line.startsWith(" ") ? line.slice(1) : line;
        result.push({ type: "ctx", oldNo: oldNo++, newNo: newNo++, content: text });
      }
    }
    return result;
  }

  const lines = $derived(parseDiff(diff));

  // Count change groups (consecutive blocks of add/del lines)
  function countGroups(ls: DiffLine[]): number {
    let count = 0;
    let inChange = false;
    for (const l of ls) {
      const isChange = l.type === "add" || l.type === "del";
      if (isChange && !inChange) { count++; inChange = true; }
      else if (!isChange) { inChange = false; }
    }
    return count;
  }

  const totalGroups = $derived(countGroups(lines));

  let currentGroup = $state(-1);
  let diffBody: HTMLElement;

  // Reset navigation when a new diff is loaded
  $effect(() => {
    diff;
    currentGroup = -1;
  });

  // Find the first <tr> of each consecutive change block in the rendered DOM
  function findGroupRows(body: HTMLElement): Element[] {
    const groups: Element[] = [];
    let inChange = false;
    for (const row of body.querySelectorAll("tr")) {
      const isChange = row.classList.contains("add") || row.classList.contains("del");
      if (isChange && !inChange) {
        groups.push(row);
        inChange = true;
      } else if (!isChange) {
        inChange = false;
      }
    }
    return groups;
  }

  function nextChange() {
    if (!diffBody) return;
    const groups = findGroupRows(diffBody);
    if (!groups.length) return;
    currentGroup = currentGroup < 0 ? 0 : (currentGroup + 1) % groups.length;
    groups[currentGroup].scrollIntoView({ behavior: "smooth", block: "center" });
  }

  function prevChange() {
    if (!diffBody) return;
    const groups = findGroupRows(diffBody);
    if (!groups.length) return;
    currentGroup = currentGroup < 0
      ? groups.length - 1
      : (currentGroup - 1 + groups.length) % groups.length;
    groups[currentGroup].scrollIntoView({ behavior: "smooth", block: "center" });
  }
</script>

<div class="diff-view">
  <!-- Header bar -->
  <div class="diff-header">
    <button class="back-btn" onclick={onback} title="Close">◀</button>
    {#if file}
      <span class="status-badge" style="color: {statusColor[file.status] ?? '#aaa'}">{file.status}</span>
      <span class="file-path">{file.path}</span>
      <span class="stat-added">+{file.added}</span>
      <span class="stat-removed">-{file.removed}</span>
    {/if}
    {#if loading}
      <span class="loading-label">Loading…</span>
    {/if}

    <!-- Change navigation -->
    {#if totalGroups > 0}
      <div class="nav-group">
        <button class="nav-btn" onclick={prevChange} title="Previous change">▲</button>
        <span class="nav-count">{currentGroup >= 0 ? currentGroup + 1 : 0}/{totalGroups}</span>
        <button class="nav-btn" onclick={nextChange} title="Next change">▼</button>
      </div>
    {/if}
  </div>

  <!-- Diff body -->
  <div class="diff-body" bind:this={diffBody}>
    {#if loading}
      <!-- spinner handled by header -->
    {:else if lines.length === 0 && !loading}
      <div class="empty">No diff available.</div>
    {:else}
      <table class="diff-table">
        <tbody>
          {#each lines as ln}
            {#if ln.type === "meta"}
              <!-- skip meta lines (diff --git, index, ---, +++) -->
            {:else if ln.type === "hunk"}
              <tr class="hunk-row">
                <td class="ln-cell" colspan="2"></td>
                <td class="hunk-content">{ln.content}</td>
              </tr>
            {:else}
              <tr class="line-row {ln.type}">
                <td class="ln-cell">{ln.oldNo ?? ""}</td>
                <td class="ln-cell">{ln.newNo ?? ""}</td>
                <td class="line-content">
                  <span class="sigil">{ln.type === "add" ? "+" : ln.type === "del" ? "-" : " "}</span>{ln.content}
                </td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>

<style>
  .diff-view {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    background: #1a1a1a;
  }

  /* Header */
  .diff-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.35rem 0.75rem;
    background: var(--toolbar-bg, #252525);
    border-bottom: 1px solid var(--border, #3a3a3a);
    flex-shrink: 0;
    min-height: 32px;
  }
  .back-btn {
    background: none;
    border: 1px solid var(--border, #555);
    color: var(--text-secondary, #ccc);
    border-radius: 4px;
    padding: 0.1rem 0.45rem;
    cursor: pointer;
    font-size: 0.8rem;
    line-height: 1.4;
    flex-shrink: 0;
  }
  .back-btn:hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
  .status-badge {
    font-family: ui-monospace, monospace;
    font-weight: 700;
    font-size: 0.75rem;
    flex-shrink: 0;
  }
  .file-path {
    font-family: ui-monospace, monospace;
    font-size: 0.82rem;
    color: var(--text, #f0f0f0);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .stat-added { font-family: ui-monospace, monospace; font-size: 0.78rem; color: #66bb6a; flex-shrink: 0; }
  .stat-removed { font-family: ui-monospace, monospace; font-size: 0.78rem; color: #ef5350; flex-shrink: 0; }
  .loading-label { font-size: 0.8rem; color: var(--text-muted, #666); }

  /* Navigation buttons */
  .nav-group {
    display: flex;
    align-items: center;
    gap: 0.2rem;
    flex-shrink: 0;
    margin-left: auto;
  }
  .nav-btn {
    background: none;
    border: 1px solid var(--border, #555);
    color: var(--text-secondary, #ccc);
    border-radius: 4px;
    padding: 0.1rem 0.4rem;
    cursor: pointer;
    font-size: 0.72rem;
    line-height: 1.4;
  }
  .nav-btn:hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
  .nav-count {
    font-family: ui-monospace, monospace;
    font-size: 0.72rem;
    color: var(--text-muted, #888);
    min-width: 3rem;
    text-align: center;
  }

  /* Body */
  .diff-body {
    flex: 1;
    min-height: 0;
    overflow-x: auto;
    overflow-y: scroll;
  }
  .empty {
    padding: 1rem;
    font-size: 0.82rem;
    color: var(--text-muted, #666);
    text-align: center;
  }
  .diff-table {
    width: 100%;
    border-collapse: collapse;
    font-family: ui-monospace, "Cascadia Code", "Consolas", monospace;
    font-size: 0.78rem;
    line-height: 1.5;
  }

  /* Line number cells */
  .ln-cell {
    width: 3rem;
    min-width: 3rem;
    padding: 0 0.4rem;
    text-align: right;
    color: var(--text-muted, #555);
    background: rgba(0,0,0,0.2);
    border-right: 1px solid rgba(255,255,255,0.06);
    user-select: none;
    font-size: 0.72rem;
    vertical-align: top;
  }

  /* Hunk header */
  .hunk-row td { background: rgba(74,126,245,0.08); border-top: 1px solid rgba(74,126,245,0.2); }
  .hunk-content {
    padding: 0.1rem 0.75rem;
    color: #6e9fff;
    font-style: italic;
    white-space: pre-wrap;
    word-break: break-all;
  }

  /* Diff lines */
  .line-content {
    padding: 0 0.75rem;
    white-space: pre-wrap;
    word-break: break-all;
    vertical-align: top;
  }
  .sigil {
    display: inline-block;
    width: 1ch;
    margin-right: 0.25rem;
    user-select: none;
  }

  /* Add line */
  .line-row.add { background: rgba(102, 187, 106, 0.1); }
  .line-row.add .ln-cell { background: rgba(102, 187, 106, 0.08); }
  .line-row.add .sigil { color: #66bb6a; }
  .line-row.add .line-content { color: #c8e6c9; }

  /* Del line */
  .line-row.del { background: rgba(239, 83, 80, 0.1); }
  .line-row.del .ln-cell { background: rgba(239, 83, 80, 0.08); }
  .line-row.del .sigil { color: #ef5350; }
  .line-row.del .line-content { color: #ffcdd2; }

  /* Context line */
  .line-row.ctx .line-content { color: var(--text-secondary, #bbb); }
</style>
