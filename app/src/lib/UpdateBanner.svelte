<script lang="ts">
  interface Props {
    version: string;
    downloading: boolean;
    progress: number;
    onupdate: () => void;
    ondismiss: () => void;
  }

  let { version, downloading, progress, onupdate, ondismiss }: Props = $props();
</script>

<div class="update-banner">
  <span class="update-icon">↑</span>
  {#if downloading}
    <span class="update-msg">Downloading v{version}…</span>
    <div class="progress-bar">
      <div class="progress-fill" style="width: {progress}%"></div>
    </div>
    <span class="progress-pct">{Math.round(progress)}%</span>
  {:else}
    <span class="update-msg">Update available: <strong>v{version}</strong></span>
    <button class="btn-update" onclick={onupdate}>Update Now</button>
    <button class="btn-later" onclick={ondismiss}>Later</button>
  {/if}
</div>

<style>
  .update-banner {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 14px;
    background: #1a3a5c;
    border-bottom: 1px solid #2a6496;
    font-size: 13px;
    color: #9ecfef;
  }

  .update-icon {
    font-size: 15px;
    color: #5aacde;
  }

  .update-msg {
    flex: 1;
  }

  .update-msg strong {
    color: #ffffff;
  }

  .progress-bar {
    flex: 1;
    height: 6px;
    background: #2a4a6a;
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: #5aacde;
    border-radius: 3px;
    transition: width 0.2s ease;
  }

  .progress-pct {
    min-width: 36px;
    text-align: right;
    color: #5aacde;
  }

  .btn-update {
    padding: 3px 12px;
    background: #2a6496;
    color: #fff;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }

  .btn-update:hover {
    background: #3a7ab6;
  }

  .btn-later {
    padding: 3px 10px;
    background: transparent;
    color: #7aace6;
    border: 1px solid #2a5a8a;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }

  .btn-later:hover {
    background: #1a2a3a;
  }
</style>
