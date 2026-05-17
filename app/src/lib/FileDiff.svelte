<script lang="ts">
  import type { CommitFile } from "./rpc-types";

  // ── Types ──────────────────────────────────────────────────────────
  type Eol = "lf" | "crlf" | "none";

  type SxsRow =
    | { kind: "hunk"; content: string }
    | { kind: "ctx";  leftNo: number; rightNo: number; text: string; eol: Eol }
    | { kind: "change";
        leftNo:   number | null; leftText:  string | null; leftEol:  Eol | null;
        rightNo:  number | null; rightText: string | null; rightEol: Eol | null;
      };

  interface Props {
    diff: string;
    file: CommitFile | null;
    loading: boolean;
    onback: () => void;
    leftLabel?: string;
    rightLabel?: string;
  }

  let { diff, file, loading, onback, leftLabel = "Before", rightLabel = "After" }: Props = $props();

  const statusColor: Record<string, string> = {
    A: "#66bb6a", D: "#ef5350", M: "#ffa726", R: "#42a5f5", C: "#ab47bc",
  };

  // ── Show/hide line endings toggle ─────────────────────────────────
  let showLineEndings = $state(false);

  const NO_NL = "\x00NONL"; // sentinel: line has no trailing newline

  // ── Pre-process: mark lines preceding "\ No newline at end of file"
  function preprocessDiff(raw: string): string {
    const lines = raw.split("\n");
    const out: string[] = [];
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].startsWith("\\ No newline") && out.length > 0) {
        // Find last content line in output and mark it
        for (let j = out.length - 1; j >= 0; j--) {
          const l = out[j];
          if (l.startsWith("+") || l.startsWith("-") || l.startsWith(" ")) {
            out[j] = l + NO_NL;
            break;
          }
        }
        // Don't include the "\ No newline..." line itself
      } else {
        out.push(lines[i]);
      }
    }
    return out.join("\n");
  }

  // ── Helper: strip trailing \r / sentinel and detect EOL type ──────
  function parseEol(s: string): { text: string; eol: Eol } {
    if (s.endsWith(NO_NL)) {
      const t = s.slice(0, -NO_NL.length);
      return { text: t.endsWith("\r") ? t.slice(0, -1) : t, eol: "none" };
    }
    if (s.endsWith("\r")) return { text: s.slice(0, -1), eol: "crlf" };
    return { text: s, eol: "lf" };
  }

  // ── Parser: unified diff → side-by-side rows ──────────────────────
  function parseSideBySide(raw: string): SxsRow[] {
    if (!raw) return [];
    const result: SxsRow[] = [];
    let oldNo = 0, newNo = 0;
    const lines = preprocessDiff(raw).split("\n");
    let i = 0;

    while (i < lines.length) {
      const line = lines[i];

      // Skip meta/empty lines
      if (!line ||
          line.startsWith("diff ") || line.startsWith("index ") ||
          line.startsWith("--- ")   || line.startsWith("+++ ") ||
          line.startsWith("new file") || line.startsWith("deleted file")) {
        i++; continue;
      }

      // Hunk header ── @@ -a,b +c,d @@
      if (line.startsWith("@@")) {
        const m = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
        if (m) { oldNo = parseInt(m[1], 10); newNo = parseInt(m[2], 10); }
        result.push({ kind: "hunk", content: line });
        i++; continue;
      }

      // Change group: collect consecutive del/add lines, then pair them
      if ((line.startsWith("-") && !line.startsWith("---")) ||
          (line.startsWith("+") && !line.startsWith("+++"))) {
        const dels: { no: number; text: string; eol: Eol }[] = [];
        const adds: { no: number; text: string; eol: Eol }[] = [];

        while (i < lines.length) {
          const l = lines[i];
          if (l.startsWith("-") && !l.startsWith("---")) {
            const { text, eol } = parseEol(l.slice(1));
            dels.push({ no: oldNo++, text, eol });
            i++;
          } else if (l.startsWith("+") && !l.startsWith("+++")) {
            const { text, eol } = parseEol(l.slice(1));
            adds.push({ no: newNo++, text, eol });
            i++;
          } else break;
        }

        const max = Math.max(dels.length, adds.length);
        for (let j = 0; j < max; j++) {
          result.push({
            kind: "change",
            leftNo:    dels[j]?.no   ?? null,
            leftText:  dels[j]?.text ?? null,
            leftEol:   dels[j]?.eol  ?? null,
            rightNo:   adds[j]?.no   ?? null,
            rightText: adds[j]?.text ?? null,
            rightEol:  adds[j]?.eol  ?? null,
          });
        }
        continue;
      }

      // Context line
      const raw2 = line.startsWith(" ") ? line.slice(1) : line;
      const { text, eol } = parseEol(raw2);
      result.push({ kind: "ctx", leftNo: oldNo++, rightNo: newNo++, text, eol });
      i++;
    }

    return result;
  }

  // ── Transform: trailing newline change → "editor view" (TortoiseGit-style) ─
  // When a paired change has same text but EOL differs ("none" vs LF/CRLF),
  // convert it to context + phantom empty line (the implicit line that
  // disappeared due to trailing newline removal/addition).
  // Lookahead: if the next row is already an empty del/add on the matching
  // side, SKIP the phantom — that existing row already represents the change.
  function transformTrailingNewline(input: SxsRow[]): SxsRow[] {
    const out: SxsRow[] = [];
    for (let i = 0; i < input.length; i++) {
      const row = input[i];
      if (row.kind !== "change" ||
          row.leftText === null || row.rightText === null ||
          row.leftText !== row.rightText ||
          (row.leftEol === "none") === (row.rightEol === "none")) {
        out.push(row);
        continue;
      }

      // This is a text-same / eol-different pair
      const leftHasEol = row.leftEol !== "none";
      const eol: Eol = (leftHasEol ? row.leftEol : row.rightEol) ?? "lf";

      // Lookahead: does next row already represent the line-count change?
      const next = input[i + 1];
      let skipPhantom = false;
      if (next && next.kind === "change") {
        if (leftHasEol && next.leftText === "" && next.rightText === null) {
          skipPhantom = true; // existing empty del covers the trailing-NL removal
        } else if (!leftHasEol && next.leftText === null && next.rightText === "") {
          skipPhantom = true; // existing empty add covers the trailing-NL addition
        }
      }

      // Convert the pair to a context row
      out.push({
        kind: "ctx",
        leftNo: row.leftNo as number,
        rightNo: row.rightNo as number,
        text: row.leftText,
        eol,
      });

      // Append phantom empty line if needed
      if (!skipPhantom) {
        if (leftHasEol) {
          out.push({
            kind: "change",
            leftNo: (row.leftNo ?? 0) + 1, leftText: "", leftEol: eol,
            rightNo: null, rightText: null, rightEol: null,
          });
        } else {
          out.push({
            kind: "change",
            leftNo: null, leftText: null, leftEol: null,
            rightNo: (row.rightNo ?? 0) + 1, rightText: "", rightEol: eol,
          });
        }
      }
    }
    return out;
  }

  const rows = $derived(transformTrailingNewline(parseSideBySide(diff)));

  // ── Change-group navigation ────────────────────────────────────────
  function countGroups(rs: SxsRow[]): number {
    let count = 0, inChange = false;
    for (const r of rs) {
      if (r.kind === "change") { if (!inChange) { count++; inChange = true; } }
      else { inChange = false; }
    }
    return count;
  }

  const totalGroups = $derived(countGroups(rows));
  let currentGroup = $state(-1);

  $effect(() => { diff; currentGroup = -1; });

  let leftPane  = $state<HTMLElement | null>(null);
  let rightPane = $state<HTMLElement | null>(null);
  let syncing = false;

  function onLeftScroll() {
    if (syncing || !rightPane || !leftPane) return;
    syncing = true; rightPane.scrollTop = leftPane.scrollTop; syncing = false;
  }
  function onRightScroll() {
    if (syncing || !leftPane || !rightPane) return;
    syncing = true; leftPane.scrollTop = rightPane.scrollTop; syncing = false;
  }

  function findChangeEls(pane: HTMLElement): HTMLElement[] {
    const groups: HTMLElement[] = [];
    let inChange = false;
    for (const el of pane.querySelectorAll<HTMLElement>(".sxs-row")) {
      const isChange = el.dataset.kind === "change";
      if (isChange && !inChange) { groups.push(el); inChange = true; }
      else if (!isChange) { inChange = false; }
    }
    return groups;
  }

  function scrollToGroup(idx: number) {
    if (!leftPane) return;
    const groups = findChangeEls(leftPane);
    if (idx < 0 || idx >= groups.length) return;
    const el = groups[idx];
    const offset = el.offsetTop - leftPane.clientHeight / 3;
    const top = Math.max(0, offset);
    leftPane.scrollTop = top;
    if (rightPane) rightPane.scrollTop = top;
  }

  function nextChange() {
    if (!totalGroups) return;
    currentGroup = currentGroup < 0 ? 0 : (currentGroup + 1) % totalGroups;
    scrollToGroup(currentGroup);
  }

  function prevChange() {
    if (!totalGroups) return;
    currentGroup = currentGroup < 0
      ? totalGroups - 1
      : (currentGroup - 1 + totalGroups) % totalGroups;
    scrollToGroup(currentGroup);
  }
</script>

<div class="diff-view">
  <!-- ── Toolbar ── -->
  <div class="diff-header">
    <button class="back-btn" onclick={onback} title="Close">◀</button>
    {#if file}
      <span class="status-badge" style="color: {statusColor[file.status] ?? '#aaa'}">{file.status}</span>
      <span class="file-path">{file.path}</span>
      <span class="stat-added">+{file.added}</span>
      <span class="stat-removed">-{file.removed}</span>
    {/if}
    {#if loading}<span class="loading-label">Loading…</span>{/if}

    <div class="toolbar-right">
      <!-- Show/hide line endings toggle -->
      <button
        class="eol-toggle"
        class:eol-toggle-active={showLineEndings}
        onclick={() => showLineEndings = !showLineEndings}
        title={showLineEndings ? "Hide line endings" : "Show line endings (LF / CRLF)"}
      >¶</button>

      <!-- Change navigation -->
      {#if totalGroups > 0}
        <div class="nav-group">
          <button class="nav-btn" onclick={prevChange} title="Previous change">▲</button>
          <span class="nav-count">{currentGroup >= 0 ? currentGroup + 1 : 0}/{totalGroups}</span>
          <button class="nav-btn" onclick={nextChange} title="Next change">▼</button>
        </div>
      {/if}
    </div>
  </div>

  <!-- ── 2-panel side-by-side ── -->
  <div class="panels">

    <!-- Left panel (old / deleted lines) -->
    <div class="panel">
      <div class="panel-label panel-label-left">{leftLabel}</div>
      <div class="panel-scroll" bind:this={leftPane} onscroll={onLeftScroll}>
        {#if !loading}
          {#if rows.length === 0}
            <div class="empty">No changes.</div>
          {:else}
            {#each rows as row}
              {#if row.kind === "hunk"}
                <div class="sxs-row" data-kind="hunk">
                  <span class="hunk-text">{row.content}</span>
                </div>
              {:else if row.kind === "ctx"}
                <div class="sxs-row" data-kind="ctx">
                  <span class="ln ctx-ln">{row.leftNo}</span>
                  <span class="code ctx-code"
                    >{row.text || " "}{#if showLineEndings && row.eol !== "none"}<span class="eol-marker eol-{row.eol}">{row.eol === "crlf" ? "CRLF" : "LF"}</span>{/if}</span>
                </div>
              {:else}
                {#if row.leftText !== null}
                  <div class="sxs-row del-row" data-kind="change">
                    <span class="ln del-ln">{row.leftNo}</span>
                    <span class="code del-code"
                      >{row.leftText || " "}{#if showLineEndings && row.leftEol && row.leftEol !== "none"}<span class="eol-marker eol-{row.leftEol}">{row.leftEol === "crlf" ? "CRLF" : "LF"}</span>{/if}</span>
                  </div>
                {:else}
                  <div class="sxs-row filler-row" data-kind="change">
                    <span class="ln filler-ln"></span>
                    <span class="code filler-code"></span>
                  </div>
                {/if}
              {/if}
            {/each}
          {/if}
        {/if}
      </div>
    </div>

    <div class="panel-divider"></div>

    <!-- Right panel (new / added lines) -->
    <div class="panel">
      <div class="panel-label panel-label-right">{rightLabel}</div>
      <div class="panel-scroll" bind:this={rightPane} onscroll={onRightScroll}>
        {#if !loading}
          {#if rows.length === 0}
            <div class="empty">No changes.</div>
          {:else}
            {#each rows as row}
              {#if row.kind === "hunk"}
                <div class="sxs-row" data-kind="hunk">
                  <span class="hunk-text">{row.content}</span>
                </div>
              {:else if row.kind === "ctx"}
                <div class="sxs-row" data-kind="ctx">
                  <span class="ln ctx-ln">{row.rightNo}</span>
                  <span class="code ctx-code"
                    >{row.text || " "}{#if showLineEndings && row.eol !== "none"}<span class="eol-marker eol-{row.eol}">{row.eol === "crlf" ? "CRLF" : "LF"}</span>{/if}</span>
                </div>
              {:else}
                {#if row.rightText !== null}
                  <div class="sxs-row add-row" data-kind="change">
                    <span class="ln add-ln">{row.rightNo}</span>
                    <span class="code add-code"
                      >{row.rightText || " "}{#if showLineEndings && row.rightEol && row.rightEol !== "none"}<span class="eol-marker eol-{row.rightEol}">{row.rightEol === "crlf" ? "CRLF" : "LF"}</span>{/if}</span>
                  </div>
                {:else}
                  <div class="sxs-row filler-row" data-kind="change">
                    <span class="ln filler-ln"></span>
                    <span class="code filler-code"></span>
                  </div>
                {/if}
              {/if}
            {/each}
          {/if}
        {/if}
      </div>
    </div>

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

  /* ── Toolbar ── */
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
  .status-badge  { font-family: ui-monospace, monospace; font-weight: 700; font-size: 0.75rem; flex-shrink: 0; }
  .file-path {
    font-family: ui-monospace, monospace; font-size: 0.82rem;
    color: var(--text, #f0f0f0); flex: 1;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .stat-added   { font-family: ui-monospace, monospace; font-size: 0.78rem; color: #66bb6a; flex-shrink: 0; }
  .stat-removed { font-family: ui-monospace, monospace; font-size: 0.78rem; color: #ef5350; flex-shrink: 0; }
  .loading-label { font-size: 0.8rem; color: var(--text-muted, #666); }

  .toolbar-right { display: flex; align-items: center; gap: 0.35rem; margin-left: auto; flex-shrink: 0; }

  /* EOL toggle button */
  .eol-toggle {
    background: none;
    border: 1px solid var(--border, #555);
    color: var(--text-muted, #666);
    border-radius: 4px;
    padding: 0.1rem 0.4rem;
    cursor: pointer;
    font-size: 0.85rem;
    line-height: 1.4;
    font-family: ui-monospace, monospace;
    transition: color 0.15s, background 0.15s, border-color 0.15s;
  }
  .eol-toggle:hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
  .eol-toggle-active {
    background: rgba(74,126,245,0.15);
    border-color: rgba(74,126,245,0.5);
    color: #7aaeff;
  }

  .nav-group { display: flex; align-items: center; gap: 0.2rem; }
  .nav-btn {
    background: none; border: 1px solid var(--border, #555);
    color: var(--text-secondary, #ccc); border-radius: 4px;
    padding: 0.1rem 0.4rem; cursor: pointer; font-size: 0.72rem; line-height: 1.4;
  }
  .nav-btn:hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
  .nav-count {
    font-family: ui-monospace, monospace; font-size: 0.72rem;
    color: var(--text-muted, #888); min-width: 3rem; text-align: center;
  }

  /* ── 2-panel layout ── */
  .panels {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .panel {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }

  .panel-label {
    flex-shrink: 0;
    padding: 0.18rem 0.75rem;
    font-family: ui-monospace, monospace;
    font-size: 0.72rem;
    font-weight: 600;
    border-bottom: 1px solid rgba(255,255,255,0.06);
    user-select: none;
    letter-spacing: 0.01em;
  }
  .panel-label-left  { background: rgba(239,83,80,0.07); color: #ef9a9a; border-right: 1px solid rgba(255,255,255,0.05); }
  .panel-label-right { background: rgba(102,187,106,0.07); color: #a5d6a7; }

  .panel-divider { width: 1px; background: rgba(255,255,255,0.07); flex-shrink: 0; }

  .panel-scroll { flex: 1; min-height: 0; overflow: auto; }

  /* ── Rows ── */
  .sxs-row {
    display: flex;
    align-items: stretch;
    min-height: 1.45em;
    font-family: ui-monospace, "Cascadia Code", Consolas, monospace;
    font-size: 0.78rem;
    line-height: 1.45;
  }

  .ln {
    flex-shrink: 0;
    width: 3.4rem;
    padding: 0 0.4rem;
    text-align: right;
    border-right: 1px solid rgba(255,255,255,0.05);
    user-select: none;
    font-size: 0.7rem;
    white-space: nowrap;
  }

  .code {
    flex: 1;
    padding: 0 0.65rem;
    white-space: pre-wrap;
    word-break: break-all;
  }

  /* Hunk header */
  .sxs-row[data-kind="hunk"] {
    background: rgba(74,126,245,0.08);
    border-top: 1px solid rgba(74,126,245,0.18);
    border-bottom: 1px solid rgba(74,126,245,0.1);
  }
  .hunk-text {
    padding: 0.1rem 0.75rem;
    color: #6e9fff;
    font-style: italic;
    font-size: 0.72rem;
    font-family: ui-monospace, monospace;
  }

  /* Context */
  .sxs-row[data-kind="ctx"] { background: #1d1d1d; }
  .ctx-ln   { background: rgba(0,0,0,0.18); color: #484848; }
  .ctx-code { color: #999; }

  /* Deletion (left panel only) */
  .del-row  { background: rgba(239,83,80,0.12); }
  .del-ln   { background: rgba(239,83,80,0.1); color: #c06060; border-right-color: rgba(239,83,80,0.2); }
  .del-code { color: #ffcdd2; }

  /* Addition (right panel only) */
  .add-row  { background: rgba(102,187,106,0.11); }
  .add-ln   { background: rgba(102,187,106,0.08); color: #5a9e60; border-right-color: rgba(102,187,106,0.2); }
  .add-code { color: #c8e6c9; }

  /* Filler (opposite side has no line) */
  .filler-row {
    background: repeating-linear-gradient(
      135deg,
      #161616 0px, #161616 4px,
      #1b1b1b 4px, #1b1b1b 8px
    );
  }
  .filler-ln   { background: transparent; color: transparent; border-right-color: rgba(255,255,255,0.03); }
  .filler-code { color: transparent; user-select: none; }

  /* ── EOL markers ── */
  .eol-marker {
    display: inline-block;
    font-family: ui-monospace, monospace;
    font-size: 0.62rem;
    font-weight: 600;
    padding: 0 0.18rem;
    border-radius: 2px;
    vertical-align: middle;
    margin-left: 0.2rem;
    user-select: none;
    line-height: 1.2;
  }
  .eol-lf   { color: #5090d0; background: rgba(80,144,208,0.12); }
  .eol-crlf { color: #d09050; background: rgba(208,144,80,0.12); }

  .empty {
    padding: 2rem;
    font-size: 0.82rem;
    color: var(--text-muted, #555);
    text-align: center;
    font-family: ui-monospace, monospace;
  }
</style>
