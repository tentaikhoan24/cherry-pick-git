<script lang="ts">
  import { open } from "@tauri-apps/plugin-dialog";
  import type { RecentRepo } from "./rpc-types";

  interface Props {
    repoPath: string;
    currentBranch: string;
    recentRepos: RecentRepo[];
    consoleOpen?: boolean;
    onopen: (path: string) => void;
    onsettings: () => void;
    onconsole: () => void;
  }

  let { repoPath, currentBranch, recentRepos, consoleOpen = false, onopen, onsettings, onconsole }: Props = $props();

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
    <button class="console-btn" class:active={consoleOpen} onclick={onconsole} title="Git Console" aria-label="Git Console">
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <polyline points="4 17 10 11 4 5"/>
        <line x1="12" y1="19" x2="20" y2="19"/>
      </svg>
    </button>
    <button class="settings-btn" onclick={onsettings} title="Settings" aria-label="Settings">
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
    </button>
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

  .console-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.35rem;
    border-radius: 6px;
    border: 1px solid var(--border, #3a3a3a);
    background: transparent;
    color: var(--text-muted, #888);
    cursor: pointer;
  }
  .console-btn:hover { background: var(--hover, #2a2a2a); color: var(--text-secondary, #ccc); }
  .console-btn.active { background: rgba(74,126,245,0.15); color: var(--accent, #4a7ef5); border-color: var(--accent, #4a7ef5); }

  .settings-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0.35rem;
    border-radius: 6px;
    border: 1px solid var(--border, #3a3a3a);
    background: transparent;
    color: var(--text-muted, #888);
    cursor: pointer;
  }
  .settings-btn:hover { background: var(--hover, #2a2a2a); color: var(--text-secondary, #ccc); }

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
    background: var(--surface-elevated, #2c2c2c);
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
