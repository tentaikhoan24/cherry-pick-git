<script lang="ts">
  import { onMount } from "svelte";
  import { rpc, RpcCallError } from "$lib/rpc";
  import type { CommitFile, FileDiffResult } from "$lib/rpc-types";
  import FileDiffPanel from "$lib/FileDiff.svelte";
  import { getCurrentWindow } from "@tauri-apps/api/window";

  let diffResult = $state<FileDiffResult | null>(null);
  let fileInfo = $state<CommitFile | null>(null);
  let loading = $state(true);
  let error = $state("");
  let leftLabel = $state("Before");
  let rightLabel = $state("After");
  let initialShowEol = $state(false);

  onMount(async () => {
    const p = new URLSearchParams(window.location.search);
    const repo = p.get("repo") ?? "";
    const sha = p.get("sha") ?? "";
    const filePath = p.get("file") ?? "";
    const status = p.get("status") ?? "M";
    const added = parseInt(p.get("added") ?? "0", 10);
    const removed = parseInt(p.get("removed") ?? "0", 10);

    fileInfo = { path: filePath, status, added, removed };

    const staged = p.get("staged") === "true";
    leftLabel = staged ? "HEAD" : "Before";
    rightLabel = staged ? "Staged" : (sha.slice(0, 8) || "Commit");

    try {
      const s = await rpc.settings.load();
      initialShowEol = s.showEolMarkers;
      document.body.classList.toggle("light", s.theme === "light");
    } catch { /* ignore */ }

    if (!repo || !filePath || (!staged && !sha)) {
      error = "Missing parameters";
      loading = false;
      return;
    }
    try {
      diffResult = staged
        ? await rpc.git.stagedFileDiff(repo, filePath)
        : await rpc.git.fileDiff(repo, sha, filePath);
    } catch (e) {
      error = e instanceof RpcCallError ? e.rpcError.message : String(e);
    } finally {
      loading = false;
    }
  });
</script>

<div class="diff-window">
  {#if error}
    <div class="error">{error}</div>
  {:else}
    <FileDiffPanel
      diff={diffResult?.diff ?? ""}
      file={fileInfo}
      {loading}
      {leftLabel}
      {rightLabel}
      {initialShowEol}
      onback={() => getCurrentWindow().close()}
    />
  {/if}
</div>

<style>
  :global(*) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    background: #1e1e1e;
    color: #f0f0f0;
    font-family: Inter, Avenir, Helvetica, Arial, sans-serif;
    font-size: 14px;
    --border: #3a3a3a;
    --border-subtle: #2e2e2e;
    --toolbar-bg: #252525;
    --input-bg: #2a2a2a;
    --hover: #333333;
    --selected: #1a2a4a;
    --accent: #4a7ef5;
    --text: #f0f0f0;
    --text-secondary: #ccc;
    --text-muted: #888;
    --surface: #252525;
    --surface-elevated: #2c2c2c;
  }
  :global(body.light) {
    background: #f5f5f5;
    color: #1a1a1a;
    --border: #d0d0d0;
    --border-subtle: #e4e4e4;
    --toolbar-bg: #ffffff;
    --input-bg: #eeeeee;
    --hover: #e4e4e4;
    --selected: #dce8ff;
    --accent: #2563eb;
    --text: #1a1a1a;
    --text-secondary: #444;
    --text-muted: #888;
    --surface: #ffffff;
    --surface-elevated: #f8f8f8;
  }
  .diff-window {
    position: fixed;
    inset: 0;
    display: flex;
    flex-direction: column;
  }
  .error {
    padding: 1rem;
    color: #ef5350;
    font-family: ui-monospace, monospace;
    font-size: 0.82rem;
  }
</style>
