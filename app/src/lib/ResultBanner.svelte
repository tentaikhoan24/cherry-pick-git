<script lang="ts">
  import type { CherryPickResult } from "./rpc-types";

  interface Props {
    result: CherryPickResult | null;
    error: string;
    targetBranch: string;
    ondismiss: () => void;
  }

  let { result, error, targetBranch, ondismiss }: Props = $props();

  const hasResult = $derived(result !== null || error !== "");

  function shortSha(sha: string) { return sha.slice(0, 7); }
</script>

{#if hasResult}
  <div class="banner" class:success={result && result.conflicts.length === 0 && !error} class:conflict={result && result.conflicts.length > 0} class:err={!!error}>
    <div class="banner-body">
      {#if error}
        <span class="icon">🔴</span>
        <span class="msg">{error}</span>
      {:else if result && result.conflicts.length > 0}
        <span class="icon">⚠️</span>
        <div class="msg">
          <strong>Conflict</strong> on <code>{shortSha(result.conflicts[0].sha)}</code>
          — files: {result.conflicts[0].files.join(", ")}
          {#if result.applied.length > 0}
            <br /><span class="applied-note">({result.applied.length} commit{result.applied.length === 1 ? "" : "s"} applied before conflict)</span>
          {/if}
        </div>
      {:else if result}
        <span class="icon">✅</span>
        <span class="msg">
          {result.applied.length} commit{result.applied.length === 1 ? "" : "s"} applied to <strong>{targetBranch}</strong>
          {#if (result as any)._pushed}· pushed to origin{/if}
        </span>
      {/if}
    </div>
    <button class="dismiss" onclick={ondismiss} title="Dismiss">✕</button>
  </div>
{/if}

<style>
  .banner {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    padding: 0.6rem 1rem;
    font-size: 0.875rem;
    border-top: 1px solid var(--border, #3a3a3a);
    flex-shrink: 0;
  }
  .banner.success { background: #1b3a1f; color: #d4f5d8; }
  .banner.conflict { background: #3a2e1a; color: #f5e4c0; }
  .banner.err      { background: #3a1b1b; color: #f5d4d4; }
  .banner-body {
    flex: 1;
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .icon { flex-shrink: 0; }
  .msg { flex: 1; line-height: 1.5; }
  .applied-note { opacity: 0.7; font-size: 0.82rem; }
  code {
    font-family: ui-monospace, monospace;
    font-size: 0.82em;
    background: rgba(255,255,255,0.1);
    padding: 0.1em 0.3em;
    border-radius: 3px;
  }
  .dismiss {
    flex-shrink: 0;
    background: none;
    border: none;
    cursor: pointer;
    opacity: 0.5;
    font-size: 0.8rem;
    padding: 0.1rem 0.3rem;
    color: inherit;
  }
  .dismiss:hover { opacity: 1; }
</style>
