<script lang="ts">
  import type { AppSettings } from "./rpc-types";

  interface Props {
    settings: AppSettings;
    onclose: () => void;
    onsave: (s: AppSettings) => void;
  }

  let { settings, onclose, onsave }: Props = $props();

  let maxCommits = $state(settings.maxCommits);
  let defaultApplyMode = $state(settings.defaultApplyMode);
  let showEolMarkers = $state(settings.showEolMarkers);
  let autoFetchOnOpen = $state(settings.autoFetchOnOpen);
  let theme = $state(settings.theme);

  function save() {
    onsave({ maxCommits, defaultApplyMode, showEolMarkers, autoFetchOnOpen, theme });
    onclose();
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="overlay" onclick={onclose} role="presentation">
  <div class="modal" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Settings">
    <div class="modal-header">
      <span class="modal-title">Settings</span>
      <button class="close-btn" onclick={onclose} aria-label="Close">✕</button>
    </div>

    <div class="modal-body">
      <div class="setting-row">
        <label class="setting-label" for="max-commits">Max commits to load</label>
        <input
          id="max-commits"
          type="number"
          class="setting-input"
          bind:value={maxCommits}
          min={10}
          max={5000}
          step={50}
        />
      </div>

      <div class="setting-row">
        <label class="setting-label" for="apply-mode">Default apply mode</label>
        <select id="apply-mode" class="setting-select" bind:value={defaultApplyMode}>
          <option value="apply">Apply only</option>
          <option value="apply-push">Apply &amp; Push</option>
        </select>
      </div>

      <div class="setting-row">
        <span class="setting-label">Show EOL markers (¶)</span>
        <button
          class="toggle"
          class:on={showEolMarkers}
          onclick={() => (showEolMarkers = !showEolMarkers)}
          aria-checked={showEolMarkers}
          role="switch"
        >
          {showEolMarkers ? "On" : "Off"}
        </button>
      </div>

      <div class="setting-row">
        <span class="setting-label">Auto-fetch on repo open</span>
        <button
          class="toggle"
          class:on={autoFetchOnOpen}
          onclick={() => (autoFetchOnOpen = !autoFetchOnOpen)}
          aria-checked={autoFetchOnOpen}
          role="switch"
        >
          {autoFetchOnOpen ? "On" : "Off"}
        </button>
      </div>

      <div class="setting-row">
        <span class="setting-label">Theme</span>
        <div class="theme-seg">
          <button class="seg-btn" class:active={theme === "dark"} onclick={() => (theme = "dark")}>Dark</button>
          <button class="seg-btn" class:active={theme === "light"} onclick={() => (theme = "light")}>Light</button>
        </div>
      </div>
    </div>

    <div class="modal-footer">
      <button class="cancel-btn" onclick={onclose}>Cancel</button>
      <button class="save-btn" onclick={save}>Save</button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 500;
  }
  .modal {
    background: var(--surface, #252525);
    border: 1px solid var(--border, #3a3a3a);
    border-radius: 10px;
    width: 380px;
    max-width: calc(100vw - 2rem);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border, #3a3a3a);
  }
  .modal-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text, #f0f0f0);
  }
  .close-btn {
    background: none;
    border: none;
    color: var(--text-muted, #888);
    font-size: 0.85rem;
    cursor: pointer;
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
  }
  .close-btn:hover { color: var(--text, #f0f0f0); background: var(--hover, #3a3a3a); }

  .modal-body {
    padding: 0.75rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .setting-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
  }
  .setting-label {
    font-size: 0.83rem;
    color: var(--text-secondary, #ccc);
    flex: 1;
  }
  .setting-input {
    width: 90px;
    padding: 0.28rem 0.5rem;
    border-radius: 5px;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #1e1e1e);
    color: var(--text, #f0f0f0);
    font-size: 0.83rem;
    text-align: right;
    outline: none;
  }
  .setting-input:focus { border-color: var(--accent, #4a7ef5); }
  .setting-select {
    width: 140px;
    padding: 0.28rem 0.5rem;
    border-radius: 5px;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #1e1e1e);
    color: var(--text, #f0f0f0);
    font-size: 0.83rem;
    outline: none;
    cursor: pointer;
  }
  .setting-select:focus { border-color: var(--accent, #4a7ef5); }

  .toggle {
    min-width: 54px;
    padding: 0.28rem 0.65rem;
    border-radius: 99px;
    border: 1px solid var(--border, #555);
    background: var(--input-bg, #2a2a2a);
    color: var(--text-muted, #888);
    font-size: 0.78rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
    text-align: center;
  }
  .toggle.on {
    background: var(--accent, #4a7ef5);
    border-color: var(--accent, #4a7ef5);
    color: #fff;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding: 0.65rem 1rem;
    border-top: 1px solid var(--border, #3a3a3a);
  }
  .cancel-btn {
    padding: 0.35rem 0.9rem;
    border-radius: 6px;
    border: 1px solid var(--border, #555);
    background: none;
    color: var(--text-secondary, #aaa);
    font-size: 0.83rem;
    cursor: pointer;
  }
  .cancel-btn:hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
  .save-btn {
    padding: 0.35rem 1.1rem;
    border-radius: 6px;
    border: none;
    background: var(--accent, #4a7ef5);
    color: #fff;
    font-size: 0.83rem;
    font-weight: 600;
    cursor: pointer;
  }
  .save-btn:hover { opacity: 0.85; }

  .theme-seg {
    display: flex;
    border: 1px solid var(--border, #555);
    border-radius: 5px;
    overflow: hidden;
  }
  .seg-btn {
    padding: 0.28rem 0.75rem;
    background: none;
    border: none;
    color: var(--text-muted, #888);
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }
  .seg-btn + .seg-btn { border-left: 1px solid var(--border, #555); }
  .seg-btn.active {
    background: var(--accent, #4a7ef5);
    color: #fff;
  }
  .seg-btn:not(.active):hover { background: var(--hover, #3a3a3a); color: var(--text, #f0f0f0); }
</style>
