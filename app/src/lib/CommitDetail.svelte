<script lang="ts">
  import type { CommitDetail } from "./rpc-types";

  interface Props {
    detail: CommitDetail | null;
    loading: boolean;
  }

  let { detail, loading }: Props = $props();

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

<div class="panel">
  <div class="panel-header">Commit detail</div>

  {#if loading}
    <div class="empty">Loading…</div>
  {:else if !detail}
    <div class="empty">Select a commit to view details.</div>
  {:else}
    <div class="body">
      <div class="subject">{detail.subject}</div>

      {#if detail.body}
        <pre class="message-body">{detail.body}</pre>
      {/if}

      <div class="meta-grid">
        <span class="meta-key">SHA</span>
        <span class="meta-val mono">{shortSha(detail.sha)} <span class="sha-full">{detail.sha}</span></span>

        <span class="meta-key">Author</span>
        <span class="meta-val">{detail.author} &lt;{detail.email}&gt;</span>

        <span class="meta-key">Date</span>
        <span class="meta-val mono">{fmt(detail.time)}</span>

        {#if detail.parents.length > 0}
          <span class="meta-key">Parent{detail.parents.length > 1 ? "s" : ""}</span>
          <span class="meta-val mono">{detail.parents.map((p) => p.slice(0, 7)).join("  ")}</span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    border-right: 1px solid var(--border, #3a3a3a);
    background: var(--input-bg, #1e1e1e);
  }
  .panel-header {
    padding: 0.35rem 0.75rem;
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-secondary, #aaa);
    border-bottom: 1px solid var(--border, #3a3a3a);
    flex-shrink: 0;
  }
  .empty {
    padding: 1rem;
    font-size: 0.82rem;
    color: var(--text-muted, #666);
    text-align: center;
  }
  .body {
    flex: 1;
    overflow-y: auto;
    padding: 0.6rem 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  .subject {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text, #f0f0f0);
    line-height: 1.35;
  }
  .message-body {
    font-family: ui-monospace, monospace;
    font-size: 0.78rem;
    color: var(--text-secondary, #ccc);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    padding: 0.4rem 0.5rem;
    background: rgba(255,255,255,0.04);
    border-radius: 4px;
    border-left: 2px solid var(--border, #3a3a3a);
    max-height: 80px;
    overflow-y: auto;
  }
  .meta-grid {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.15rem 0.75rem;
    font-size: 0.78rem;
  }
  .meta-key {
    color: var(--text-muted, #666);
    white-space: nowrap;
  }
  .meta-val {
    color: var(--text-secondary, #ccc);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mono { font-family: ui-monospace, monospace; }
  .sha-full {
    font-size: 0.7rem;
    color: var(--text-muted, #666);
    margin-left: 0.4rem;
  }
</style>
