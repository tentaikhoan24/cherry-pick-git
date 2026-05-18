<script lang="ts">
  import { rpc, RpcCallError } from "$lib/rpc";
  import type { Branch, Commit, CommitFilter, CherryPickResult, CherryPickProgress, RecentRepo, CommitDetail, CommitFile, DryRunItem, ConflictFileInfo, AppSettings } from "$lib/rpc-types";
  import { invoke } from "@tauri-apps/api/core";
  import { WebviewWindow } from "@tauri-apps/api/webviewWindow";
  import { listen } from "@tauri-apps/api/event";
  import { check, type Update } from "@tauri-apps/plugin-updater";
  import { relaunch } from "@tauri-apps/plugin-process";
  import Toolbar from "$lib/Toolbar.svelte";
  import CommitList from "$lib/CommitList.svelte";
  import PickQueue from "$lib/PickQueue.svelte";
  import ResultBanner from "$lib/ResultBanner.svelte";
  import UpdateBanner from "$lib/UpdateBanner.svelte";
  import CommitDetailPanel from "$lib/CommitDetail.svelte";
  import CommitFilesPanel from "$lib/CommitFiles.svelte";
  import ConflictResolver from "$lib/ConflictResolver.svelte";
  import SettingsModal from "$lib/Settings.svelte";
  import GitConsole from "$lib/GitConsole.svelte";

  // ── settings ──────────────────────────────────────────────
  const DEFAULT_SETTINGS: AppSettings = {
    maxCommits: 100, defaultApplyMode: "apply", showEolMarkers: false, autoFetchOnOpen: false, theme: "dark",
    externalDiffEnabled: false, externalDiffPath: "", externalDiffArgs: "",
    externalMergeEnabled: false, externalMergePath: "", externalMergeArgs: "",
    checkForUpdatesOnStartup: true,
  };
  let settings = $state<AppSettings>(DEFAULT_SETTINGS);
  let settingsOpen = $state(false);
  let pendingUpdate = $state<Update | null>(null);
  let updateDownloading = $state(false);
  let updateProgress = $state(0);
  let consoleOpen = $state(false);
  let consoleHeight = $state(180);

  function startConsoleResize(e: MouseEvent) {
    e.preventDefault();
    const startY = e.clientY;
    const startH = consoleHeight;
    function onMove(ev: MouseEvent) {
      consoleHeight = Math.max(80, Math.min(600, startH + (startY - ev.clientY)));
    }
    function onUp() {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    }
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  rpc.settings.load().then(s => {
    settings = s;
    if (s.checkForUpdatesOnStartup) {
      check().then(u => { if (u) pendingUpdate = u; }).catch(() => {});
    }
  }).catch(() => {});

  async function installUpdate() {
    if (!pendingUpdate) return;
    updateDownloading = true;
    updateProgress = 0;
    let totalBytes = 0;
    let downloadedBytes = 0;
    await pendingUpdate.downloadAndInstall((event) => {
      if (event.event === "Started") {
        totalBytes = event.data.contentLength ?? 0;
      } else if (event.event === "Progress") {
        downloadedBytes += event.data.chunkLength;
        if (totalBytes > 0) updateProgress = (downloadedBytes / totalBytes) * 100;
      }
    });
    await relaunch();
  }

  async function checkForUpdates() {
    const u = await check().catch(() => null);
    if (u) pendingUpdate = u;
    return !!u;
  }

  $effect(() => {
    document.body.classList.toggle("light", settings.theme === "light");
  });

  async function saveSettings(s: AppSettings) {
    settings = s;
    try { await rpc.settings.save(s); } catch { /* ignore */ }
  }

  // ── repo state ────────────────────────────────────────────
  let repoPath = $state("");
  let currentBranch = $state("");

  // ── branch / commit lists ─────────────────────────────────
  let branches = $state<Branch[]>([]);
  let sourceBranch = $state("");
  let targetBranch = $state("");
  let commits = $state<Commit[]>([]);
  let loadingCommits = $state(false);

  // ── selection: Map preserves insertion order for queue ─────
  let selectionMap = $state(new Map<string, Commit>());
  const queue = $derived([...selectionMap.values()]);

  // ── refresh (fetch / pull) ────────────────────────────────
  let refreshing = $state(false);

  async function doFetch() {
    if (!repoPath) return;
    refreshing = true;
    applyError = "";
    try {
      await rpc.git.fetch(repoPath);
      branches = await rpc.git.branches(repoPath);
      await loadCommits(sourceBranch);
    } catch (e) {
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    } finally {
      refreshing = false;
    }
  }

  async function doPull() {
    if (!repoPath) return;
    refreshing = true;
    applyError = "";
    try {
      await rpc.git.pull(repoPath, sourceBranch);
      branches = await rpc.git.branches(repoPath);
      await loadCommits(sourceBranch);
    } catch (e) {
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    } finally {
      refreshing = false;
    }
  }

  // ── apply / progress ──────────────────────────────────────
  let busy = $state(false);
  let progress = $state<CherryPickProgress | null>(null);
  let applyResult = $state<CherryPickResult | null>(null);
  let applyError = $state("");

  // ── conflict resolver (M5c) ──────────────────────────────
  let conflictFiles = $state<ConflictFileInfo[]>([]);
  let conflictSha = $state("");
  let resolvedSet = $state(new Set<string>());
  let conflictBusy = $state(false);
  // Files touched by each remaining queued commit (fetched when conflict mode enters)
  let remainingCommitFiles = $state<Map<string, string[]>>(new Map());

  async function loadConflictFiles(): Promise<boolean> {
    try {
      const r = await rpc.git.conflictFiles(repoPath);
      conflictFiles = r.files;
      resolvedSet = new Set();
      return true;
    } catch {
      return false; // don't touch applyError — caller decides how to surface this
    }
  }

  // Fetch which files each remaining queue commit touches, so the UI can show them.
  async function loadRemainingCommitFiles(conflictingSha: string) {
    const idx = queue.findIndex(c =>
      c.sha === conflictingSha || c.sha.startsWith(conflictingSha) || conflictingSha.startsWith(c.sha)
    );
    const remaining = idx >= 0 ? queue.slice(idx + 1) : [];
    if (remaining.length === 0) { remainingCommitFiles = new Map(); return; }
    const m = new Map<string, string[]>();
    await Promise.all(remaining.map(async c => {
      try {
        const r = await rpc.git.commitFiles(repoPath, c.sha);
        m.set(c.sha, r.map(f => f.path));
      } catch { /* best-effort */ }
    }));
    remainingCommitFiles = m;
  }

  async function resolveConflictFile(file: string, strategy: "ours" | "theirs") {
    conflictBusy = true;
    try {
      await rpc.git.resolveConflict(repoPath, file, strategy);
      resolvedSet = new Set([...resolvedSet, file]);
    } catch (e) {
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    } finally {
      conflictBusy = false;
    }
  }

  async function continueCherry() {
    conflictBusy = true;
    try {
      await rpc.git.continueCherry(repoPath);
      // The resolved commit is now committed. Find remaining queue commits to apply next.
      // (Sidecar applies one commit at a time — git's sequencer has no knowledge of the rest.)
      const resolvedSha = conflictSha;
      const resolvedIdx = queue.findIndex(c =>
        c.sha === resolvedSha || c.sha.startsWith(resolvedSha) || resolvedSha.startsWith(c.sha)
      );
      const remainingShas = resolvedIdx >= 0 ? queue.slice(resolvedIdx + 1).map(c => c.sha) : [];

      // Clear conflict state
      conflictFiles = [];
      resolvedSet = new Set();
      remainingCommitFiles = new Map();
      conflictSha = "";
      applyError = "";
      applyResult = { applied: [resolvedSha], conflicts: [] };
      branches = await rpc.git.branches(repoPath);
      const updated = branches.find((b) => b.isHead);
      if (updated) currentBranch = updated.name;

      // Apply any remaining queued commits (may produce new conflicts)
      if (remainingShas.length > 0) {
        conflictBusy = false;
        await applyPickShas(remainingShas);
        return;
      }
      // All commits applied — clear the queue selection
      selectionMap = new Map();
    } catch (e) {
      // git cherry-pick --continue failed (e.g. unstaged files, or genuine error)
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
      await loadConflictFiles();
      if (conflictFiles.length === 0) {
        conflictSha = "";
        remainingCommitFiles = new Map();
      } else {
        try {
          const r = await rpc.git.openRepo(repoPath);
          if (r.cherryPickHead) conflictSha = r.cherryPickHead;
        } catch { /* ignore */ }
        await loadRemainingCommitFiles(conflictSha);
      }
    } finally {
      conflictBusy = false;
    }
  }

  async function viewConflictFile(file: string) {
    if (!repoPath) return;

    if (settings.externalMergeEnabled && settings.externalMergePath) {
      conflictBusy = true;
      try {
        const res = await rpc.git.extractConflictFiles(repoPath, file);
        const template = settings.externalMergeArgs || '"{theirs}" "{ours}" "{base}" "{output}"';
        const args = buildArgs(template, {
          base: res.basePath, ours: res.oursPath,
          theirs: res.theirsPath, output: res.outputPath,
        });
        await invoke("launch_and_wait", { program: settings.externalMergePath, args });
        await rpc.git.stageResolvedFile(repoPath, file, res.outputPath);
        await rpc.git.cleanupTmpDir(res.tmpDir);
        resolvedSet = new Set([...resolvedSet, file]);
      } catch (e) {
        applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
      } finally {
        conflictBusy = false;
      }
      return;
    }

    const params = new URLSearchParams({ repo: repoPath, file });
    new WebviewWindow(`conflict-${Date.now()}`, {
      url: `${window.location.origin}/conflict?${params}`,
      title: `Conflict: ${file}`,
      width: 800,
      height: 600,
    });
  }

  function viewConflictFileDiff(file: string) {
    if (!repoPath) return;
    const params = new URLSearchParams({ repo: repoPath, file, staged: "true", status: "M" });
    new WebviewWindow(`diff-${Date.now()}`, {
      url: `${window.location.origin}/diff?${params}`,
      title: `Staged diff: ${file}`,
      width: 900,
      height: 650,
    });
  }

  async function abortConflict() {
    conflictBusy = true;
    try { await rpc.git.abort(repoPath); } catch { /* ignore */ }
    conflictFiles = [];
    conflictSha = "";
    resolvedSet = new Set();
    applyResult = null;
    applyError = "Cherry-pick aborted.";
    conflictBusy = false;
  }

  // ── commit detail (M5a) ───────────────────────────────────
  let selectedCommit = $state<Commit | null>(null);
  let commitDetail = $state<CommitDetail | null>(null);
  let commitFiles = $state<CommitFile[]>([]);
  let loadingDetail = $state(false);
  let detailError = $state("");
  let detailHeight = $state(200);

  async function selectCommit(commit: Commit) {
    if (selectedCommit?.sha === commit.sha) {
      selectedCommit = null;
      commitDetail = null;
      commitFiles = [];
      detailError = "";
      return;
    }
    selectedCommit = commit;
    commitDetail = null;
    commitFiles = [];
    detailError = "";
    selectedFile = null;
    if (!repoPath) return;
    loadingDetail = true;
    try {
      // Sequential — Rust ActiveSidecar only tracks one child at a time.
      commitDetail = await rpc.git.commitDetail(repoPath, commit.sha);
      commitFiles = await rpc.git.commitFiles(repoPath, commit.sha);
    } catch (e) {
      detailError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    } finally {
      loadingDetail = false;
    }
  }

  // ── external tool helpers ─────────────────────────────────

  // Parse an args template with {placeholder} substitution into a string[].
  function buildArgs(template: string, vars: Record<string, string>): string[] {
    let s = template;
    for (const [k, v] of Object.entries(vars)) s = s.replaceAll(`{${k}}`, v);
    const result: string[] = [];
    // Match prefix+"quoted value" (e.g. /path:"C:\some path\file") as ONE arg,
    // or a bare quoted string, or a non-space token.
    // Note: [^\s"] (not \S) for the prefix — \S includes " which breaks the parse.
    const re = /([^\s"]*)"([^"]*)"|\S+/g;
    let m: RegExpExecArray | null;
    while ((m = re.exec(s)) !== null) result.push(m[2] !== undefined ? m[1] + m[2] : m[0]);
    return result;
  }

  // ── file diff viewer ─────────────────────────────────────
  let selectedFile = $state<CommitFile | null>(null);

  async function selectFile(file: CommitFile) {
    if (!selectedCommit || !repoPath) return;
    selectedFile = file;

    if (settings.externalDiffEnabled && settings.externalDiffPath) {
      try {
        const res = await rpc.git.extractDiffFiles(repoPath, selectedCommit.sha, file.path);
        const template = settings.externalDiffArgs || '"{left}" "{right}"';
        const args = buildArgs(template, {
          left: res.leftPath, right: res.rightPath,
          leftLabel: res.leftLabel, rightLabel: res.rightLabel,
        });
        await invoke("launch_detached", { program: settings.externalDiffPath, args });
        return;
      } catch (e) {
        applyError = `External diff tool error: ${e instanceof Error ? e.message : String(e)}`;
        return;
      }
    }

    const params = new URLSearchParams({
      repo: repoPath,
      sha: selectedCommit.sha,
      file: file.path,
      status: file.status,
      added: String(file.added),
      removed: String(file.removed),
    });
    new WebviewWindow(`diff-${Date.now()}`, {
      url: `${window.location.origin}/diff?${params}`,
      title: `Diff: ${file.path}`,
      width: 900,
      height: 650,
    });
  }

  function startDetailResize(e: MouseEvent) {
    e.preventDefault();
    const startY = e.clientY;
    const startH = detailHeight;
    function onMove(ev: MouseEvent) {
      detailHeight = Math.max(80, Math.min(520, startH + (startY - ev.clientY)));
    }
    function onUp() {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    }
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  // ── column (left / right pane) resize ────────────────────
  let pickQueueWidth = $state(340);

  function startColResize(e: MouseEvent) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = pickQueueWidth;
    function onMove(ev: MouseEvent) {
      pickQueueWidth = Math.max(220, Math.min(600, startW - (ev.clientX - startX)));
    }
    function onUp() {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    }
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  }

  // ── dry-run conflict preview (M5b) ────────────────────────
  let dryRunMap = $state(new Map<string, DryRunItem>());
  let dryRunTimer: ReturnType<typeof setTimeout> | null = null;

  function scheduleDryRun() {
    if (dryRunTimer) clearTimeout(dryRunTimer);
    dryRunTimer = setTimeout(runDryRun, 400);
  }

  async function runDryRun() {
    if (!repoPath || queue.length === 0) {
      dryRunMap = new Map();
      return;
    }
    const shas = queue.map((c) => c.sha);
    try {
      const res = await rpc.git.dryRunPick(repoPath, targetBranch, shas);
      const m = new Map<string, DryRunItem>();
      for (const item of res.results) m.set(item.sha, item);
      dryRunMap = m;
    } catch { /* silently ignore — dry-run is best-effort */ }
  }

  // ── recent repos ──────────────────────────────────────────
  let recentRepos = $state<RecentRepo[]>([]);

  async function loadRecents() {
    try { recentRepos = await rpc.recents.load(); } catch { /* ignore */ }
  }

  async function saveRecent(path: string) {
    const now = Math.floor(Date.now() / 1000);
    const filtered = recentRepos.filter((r) => r.path !== path);
    recentRepos = [{ path, lastOpened: now }, ...filtered].slice(0, 10);
    try { await rpc.recents.save(recentRepos); } catch { /* ignore */ }
  }

  loadRecents();

  // Listen for conflict-file-resolved events emitted by the conflict merge editor window.
  $effect(() => {
    let unlisten: (() => void) | null = null;
    listen<{ file: string }>("conflict-file-resolved", (e) => {
      resolvedSet = new Set([...resolvedSet, e.payload.file]);
    }).then(fn => { unlisten = fn; });
    return () => { unlisten?.(); };
  });

  // ── open repo ─────────────────────────────────────────────
  async function openRepo(path: string) {
    applyResult = null;
    applyError = "";
    selectionMap = new Map();
    try {
      const r = await rpc.git.openRepo(path);
      repoPath = r.path;
      currentBranch = r.branch;
      branches = await rpc.git.branches(r.path);
      // default source = any non-current branch; target = current
      const nonCurrent = branches.find((b) => !b.isHead);
      sourceBranch = nonCurrent?.name ?? r.branch;
      targetBranch = r.branch;
      await loadCommits(sourceBranch);
      await saveRecent(r.path);
      if (settings.autoFetchOnOpen) {
        try { await rpc.git.fetch(r.path); branches = await rpc.git.branches(r.path); await loadCommits(sourceBranch); } catch { /* ignore */ }
      }
      if (r.cherryPickHead) {
        conflictSha = r.cherryPickHead;
        await loadConflictFiles();
        await loadRemainingCommitFiles(r.cherryPickHead);
        // If sidecar failed, conflictSha is still set — ConflictResolver won't show
        // but user can abort. Error shown via applyError.
      }
    } catch (e) {
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    }
  }

  let activeFilter = $state<CommitFilter>({});

  async function loadCommits(branch: string, filter?: CommitFilter) {
    loadingCommits = true;
    commits = [];
    try {
      commits = await rpc.git.commits(repoPath, branch, settings.maxCommits, 0, filter ?? activeFilter);
    } catch (e) {
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    } finally {
      loadingCommits = false;
    }
  }

  function changeSourceBranch(branch: string) {
    sourceBranch = branch;
    selectionMap = new Map();
    applyResult = null;
    applyError = "";
    selectedCommit = null;
    commitDetail = null;
    commitFiles = [];
    dryRunMap = new Map();
    activeFilter = {};
    loadCommits(branch, {});
  }

  function applyCommitFilter(filter: CommitFilter) {
    activeFilter = filter;
    loadCommits(sourceBranch, filter);
  }

  function toggleCommit(sha: string) {
    const next = new Map(selectionMap);
    if (next.has(sha)) {
      next.delete(sha);
    } else {
      const c = commits.find((c) => c.sha === sha);
      if (c) next.set(sha, c);
    }
    selectionMap = next;
    if (applyError) { applyError = ""; applyResult = null; }
    scheduleDryRun();
  }

  function removeFromQueue(sha: string) {
    const next = new Map(selectionMap);
    next.delete(sha);
    selectionMap = next;
    if (applyError) { applyError = ""; applyResult = null; }
    scheduleDryRun();
  }

  async function createBranch(name: string, base: string) {
    if (!repoPath) return;
    applyError = "";
    try {
      await rpc.git.createBranch(repoPath, name, base);
      branches = await rpc.git.branches(repoPath);
      targetBranch = name;
    } catch (e) {
      applyError = e instanceof RpcCallError ? e.rpcError.message : String(e);
    }
  }

  async function cancelPick() {
    try { await rpc.cancel(); } catch { /* ignore */ }
    try { await rpc.git.abort(repoPath); } catch { /* ignore */ }
    busy = false;
    progress = null;
    applyError = "Cherry-pick cancelled.";
  }

  // Core apply logic — called by applyPick (full queue) and continueCherry (remaining shas).
  async function applyPickShas(shas: string[], andPush = false) {
    busy = true;
    progress = null;
    applyResult = null;
    applyError = "";
    try {
      applyResult = await rpc.git.cherryPick(
        repoPath, targetBranch, shas, undefined,
        (p) => { progress = p; }
      );
      // success — clear selection, refresh branches
      selectionMap = new Map();
      branches = await rpc.git.branches(repoPath);
      const updated = branches.find((b) => b.isHead);
      if (updated) currentBranch = updated.name;

      if (andPush) {
        await rpc.git.push(repoPath, targetBranch);
        applyError = "";
        (applyResult as any)._pushed = true;
      }
    } catch (e) {
      if (e instanceof RpcCallError) {
        if (e.rpcError.code === -32003) {
          const d = e.rpcError.data as { applied?: string[]; conflicts?: { sha: string; files: string[] }[] };
          const conflicts = d?.conflicts ?? [];
          applyResult = { applied: d?.applied ?? [], conflicts };
          if (conflicts.length > 0) {
            conflictSha = conflicts[0].sha;
            const loaded = await loadConflictFiles();
            if (!loaded && conflicts[0].files.length > 0) {
              conflictFiles = conflicts[0].files.map(f => ({ path: f, status: "UU" as const }));
              applyError = "";
            }
            await loadRemainingCommitFiles(conflicts[0].sha);
          }
        } else {
          applyError = `[${e.rpcError.code}] ${e.rpcError.message}`;
        }
      } else {
        applyError = String(e);
      }
    } finally {
      busy = false;
      progress = null;
    }
  }

  async function applyPick(andPush = false) {
    if (!repoPath || queue.length === 0) return;
    await applyPickShas(queue.map((c) => c.sha), andPush);
  }

  function dismissResult() {
    applyResult = null;
    applyError = "";
    conflictFiles = [];
    conflictSha = "";
    resolvedSet = new Set();
  }
</script>

<div class="app">
  <Toolbar {repoPath} {currentBranch} {recentRepos} {consoleOpen} onopen={openRepo} onsettings={() => (settingsOpen = true)} onconsole={() => (consoleOpen = !consoleOpen)} />
  {#if pendingUpdate}
    <UpdateBanner
      version={pendingUpdate.version}
      downloading={updateDownloading}
      progress={updateProgress}
      onupdate={installUpdate}
      ondismiss={() => (pendingUpdate = null)}
    />
  {/if}

  {#if repoPath}
    <div class="workspace">
      <div class="left-pane">
        <CommitList
          {branches}
          {sourceBranch}
          {commits}
          selected={new Set(selectionMap.keys())}
          selectedSha={selectedCommit?.sha ?? ""}
          loading={loadingCommits}
          {refreshing}
          onsourcebranch={changeSourceBranch}
          ontoggle={toggleCommit}
          onselect={selectCommit}
          onfetch={doFetch}
          onpull={doPull}
          onfilter={applyCommitFilter}
        />
      </div>
      <div class="col-resize-handle" onmousedown={startColResize} role="separator" aria-label="Resize panels"></div>
      <div class="right-pane" style="width: {pickQueueWidth}px">
        <PickQueue
          {queue}
          {branches}
          {targetBranch}
          {sourceBranch}
          {busy}
          {progress}
          {dryRunMap}
          defaultApplyMode={settings.defaultApplyMode}
          ontargetbranch={(b) => { targetBranch = b; applyResult = null; applyError = ""; scheduleDryRun(); }}
          onremove={removeFromQueue}
          onapply={() => applyPick(false)}
          onapplypush={() => applyPick(true)}
          oncancel={cancelPick}
          oncreate={createBranch}
        />
      </div>
    </div>

    {#if selectedCommit}
      <div class="detail-resize-handle" onmousedown={startDetailResize} role="separator" aria-label="Resize detail panel"></div>
      <div class="detail-area" style="height: {detailHeight}px; grid-template-columns: 1fr {pickQueueWidth + 5}px">
        {#if detailError}
          <div class="detail-error">{detailError}</div>
        {:else}
          <CommitDetailPanel detail={commitDetail} loading={loadingDetail} />
          <CommitFilesPanel
            files={commitFiles}
            loading={loadingDetail}
            selectedPath={selectedFile?.path ?? ""}
            onselect={selectFile}
          />
        {/if}
      </div>
    {/if}

    {#if conflictFiles.length > 0}
      <ConflictResolver
        files={conflictFiles}
        conflictSha={conflictSha}
        queue={queue}
        dryRunMap={dryRunMap}
        remainingCommitFiles={remainingCommitFiles}
        busy={conflictBusy}
        resolvedSet={resolvedSet}
        onresolve={resolveConflictFile}
        oncontinue={continueCherry}
        onabort={abortConflict}
        onviewfile={viewConflictFile}
        onviewdiff={viewConflictFileDiff}
      />
    {/if}

    <ResultBanner
      result={applyResult}
      error={applyError}
      {targetBranch}
      ondismiss={dismissResult}
    />
  {:else}
    <div class="welcome">
      <p>Open a Git repository to get started.</p>
    </div>
  {/if}

  {#if consoleOpen}
    <div class="console-resize-handle" onmousedown={startConsoleResize} role="separator" aria-label="Resize console"></div>
    <GitConsole height={consoleHeight} onclose={() => (consoleOpen = false)} />
  {/if}

  {#if settingsOpen}
    <SettingsModal
      {settings}
      onclose={() => (settingsOpen = false)}
      onsave={saveSettings}
      onchecknow={checkForUpdates}
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
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }
  .workspace {
    display: flex;
    flex: 1;
    overflow: hidden;
    min-height: 0;
  }
  .left-pane {
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }
  .right-pane {
    flex-shrink: 0;
    overflow: hidden;
  }
  .col-resize-handle {
    flex-shrink: 0;
    width: 5px;
    background: var(--border, #3a3a3a);
    cursor: ew-resize;
    transition: background 0.15s;
  }
  .col-resize-handle:hover { background: var(--accent, #4a7ef5); }
  .detail-resize-handle {
    flex-shrink: 0;
    height: 5px;
    background: var(--border, #3a3a3a);
    cursor: ns-resize;
    transition: background 0.15s;
  }
  .detail-resize-handle:hover { background: var(--accent, #4a7ef5); }
  .console-resize-handle {
    flex-shrink: 0;
    height: 5px;
    background: #2a2a2a;
    cursor: ns-resize;
    transition: background 0.15s;
  }
  .console-resize-handle:hover { background: #4a7ef5; }
  .detail-area {
    display: grid;
    grid-template-columns: 1fr 340px;
    flex-shrink: 0;
    border-top: none;
    overflow: hidden;
  }
  .detail-error {
    grid-column: 1 / -1;
    padding: 0.75rem 1rem;
    font-size: 0.82rem;
    color: #ef5350;
    font-family: ui-monospace, monospace;
  }
  .welcome {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
    font-size: 0.95rem;
  }
</style>
