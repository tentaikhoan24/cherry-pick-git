<script lang="ts">
  import { open } from "@tauri-apps/plugin-dialog";
  import type { RecentRepo } from "./rpc-types";

  interface Props {
    repoPath: string;
    currentBranch: string;
    recentRepos: RecentRepo[];
    onopen: (path: string) => void;
  }

  let { repoPath, currentBranch, recentRepos, onopen }: Props = $props();

  let recentsOpen = $state(false);

  async function pickFolder() {
    const dir = await open({ directory: true, multiple: false, title: "Open Git repository" });
    if (dir) onopen(dir as string);
  }

  function openRecent(path: string) {
    recentsOpen = false;
    onopen(path);
  }

  function onClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest(".recents-wrap")) recentsOpen = false;
  }

  function basename(p: string) {
    return p.replace(/\\/g, "/").split("/").pop() ?? p;
  }
</script>

<svelte:window onclick={onClickOutside} />

<header class="toolbar">
  <div class="path">
    {#if repoPath}
      <span class="icon">📁</span>
      <span class="repo-path" title={repoPath}>{repoPath}</span>
      {#if currentBranch}
        <span class="branch-badge">{currentBranch}</span>
      {/if}
    {:else}
      <span class="placeholder">No repository open</span>
    {/if}
  </div>

  <div class="actions">
    <!-- Recent repos dropdown -->
    {#if recentRepos.length > 0}
      <div class="recents-wrap">
        <button
          class="recents-btn"
          onclick={() => (recentsOpen = !recentsOpen)}
          title="Recent repositories"
        >
          Recent ▾
        </button>
        {#if recentsOpen}
          <ul class="recents-menu">
            {#each recentRepos as r}
              <li>
                <button onclick={() => openRecent(r.path)} title={r.path}>
                  <span class="recent-name">{basename(r.path)}</span>
                  <span class="recent-path">{r.path}</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}
    <button class="open-btn" onclick={pickFolder}>Open repo</button>
  </div>
</header>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 1rem;
    background: var(--toolbar-bg, #2c2c2c);
    border-bottom: 1px solid var(--border, #3a3a3a);
    min-height: 44px;
  }
  .path {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex: 1;
    overflow: hidden;
  }
  .icon { font-size: 1rem; flex-shrink: 0; }
  .repo-path {
    font-family: ui-monospace, monospace;
    font-size: 0.82rem;
    color: var(--text-secondary, #ccc);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .branch-badge {
    flex-shrink: 0;
    padding: 0.15rem 0.55rem;
    border-radius: 12px;
    font-size: 0.75rem;
    font-family: ui-monospace, monospace;
    background: var(--accent, #396cd8);
    color: #fff;
  }
  .placeholder {
    font-size: 0.85rem;
    color: var(--text-muted, #666);
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-shrink: 0;
  }

  .open-btn {
    padding: 0.35rem 0.9rem;
    border-radius: 6px;
    border: 1px solid var(--accent, #396cd8);
    background: transparent;
    color: var(--accent, #396cd8);
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 500;
  }
  .open-btn:hover { background: var(--accent, #396cd8); color: #fff; }

  /* recents */
  .recents-wrap {
    position: relative;
  }
  .recents-btn {
    padding: 0.35rem 0.75rem;
    border-radius: 6px;
    border: 1px solid var(--border, #3a3a3a);
    background: transparent;
    color: var(--text-secondary, #ccc);
    cursor: pointer;
    font-size: 0.82rem;
  }
  .recents-btn:hover { background: var(--hover, #2a2a2a); }

  .recents-menu {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    min-width: 280px;
    max-width: 400px;
    background: #2c2c2c;
    border: 1px solid var(--border, #3a3a3a);
    border-radius: 7px;
    list-style: none;
    margin: 0;
    padding: 0.3rem 0;
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
    z-index: 200;
  }
  .recents-menu li button {
    width: 100%;
    padding: 0.45rem 1rem;
    background: none;
    border: none;
    color: var(--text, #f0f0f0);
    text-align: left;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    border-radius: 4px;
  }
  .recents-menu li button:hover { background: var(--hover, #3a3a3a); }
  .recent-name {
    font-size: 0.88rem;
    font-weight: 500;
  }
  .recent-path {
    font-size: 0.73rem;
    font-family: ui-monospace, monospace;
    color: var(--text-muted, #888);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 340px;
  }
</style>
