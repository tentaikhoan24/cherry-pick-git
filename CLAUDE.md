# CLAUDE.md

Context for AI-assisted development on `lazy-cherry-pick`. Read this first when entering the repo.

## In one paragraph

A Tauri 2 desktop app for batch Git cherry-pick workflows. Rust backend spawns a Go sidecar and talks to it over **newline-delimited JSON-RPC 2.0 on stdin/stdout**. The sidecar shells directly to `git` CLI (no library). Frontend is Svelte 5 + TypeScript via SvelteKit. M8 done: external diff/merge tool support — users can configure TortoiseGit, Beyond Compare, WinMerge, VSCode (or any exe) as diff viewer / merge tool; Go sidecar extracts file versions to temp dirs; Rust `launch_detached`/`launch_and_wait` commands launch external processes; Settings modal has path input + args template + Browse button + Auto-detect. Built-in viewers remain as fallback when external tools are disabled. All UI text in English.

## File map

| Path | Role |
|---|---|
| `app/src/routes/+page.svelte` | Main orchestrator — 3-pane UI, all app state |
| `app/src/routes/diff/+page.svelte` | File diff viewer window (`?staged=true` switches to staged diff via `git.stagedFileDiff`) |
| `app/src/routes/conflict/+page.svelte` | 3-pane conflict merge editor (Theirs / Ours / merged result) |
| `app/src/lib/Toolbar.svelte` | Repo path display + Open repo (file dialog) |
| `app/src/lib/CommitList.svelte` | Source branch dropdown + scrollable commit list with checkboxes |
| `app/src/lib/PickQueue.svelte` | Ordered pick queue + target branch dropdown + Apply button |
| `app/src/lib/ResultBanner.svelte` | Result display: ✅ applied / ⚠️ conflict / 🔴 error |
| `app/src/lib/CommitDetail.svelte` | Commit subject, body, SHA, author, date, parents panel |
| `app/src/lib/CommitFiles.svelte` | File list for a commit with status badge (+/- stats), click to open diff |
| `app/src/lib/FileDiff.svelte` | 2-panel side-by-side diff renderer (TortoiseGit-style): sync scroll, EOL marker toggle (¶), trailing-newline transform |
| `app/src/lib/ConflictResolver.svelte` | Conflict file list — Keep Ours / Use Theirs / Continue / Abort; resolved files open staged diff |
| `app/src/lib/Settings.svelte` | Settings modal — maxCommits, defaultApplyMode, showEolMarkers, autoFetchOnOpen, theme (Dark/Light); External Tools section (diff + merge: toggle, path, args template, Browse button, Auto-detect) |
| `app/src/lib/GitConsole.svelte` | Git Console panel — realtime git command log grouped by RPC call; load history from file; Clear button |
| `app/src/lib/rpc.ts` | Typed wrapper around `invoke('sidecar_call', ...)` + direct Tauri commands |
| `app/src/lib/rpc-types.ts` | TypeScript types mirroring Go sidecar types |
| `app/src-tauri/src/lib.rs` | Rust entry; `sidecar_call` (stderr `[GIT_CMD]`/`[GIT_INFO]` parsing + git-log events + file append), `settings_load/save`, `git_log_read/clear`, `recents_load/save`, `launch_detached`, `launch_and_wait`, `detect_external_tools` |
| `app/src-tauri/Cargo.toml` | `tauri-plugin-shell`, `tauri-plugin-dialog`, `tokio` (features: `sync`, `io-util`, `macros`, `process`) |
| `sidecar/internal/git/externaltool.go` | `ExtractDiffFiles`, `ExtractConflictFiles`, `StageResolvedFile`, `CleanupTmpDir` — extracts file versions to `lcp-diff-*`/`lcp-merge-*` temp dirs for external tools |
| `app/src-tauri/tauri.conf.json` | `bundle.externalBin: ["binaries/sidecar"]` |
| `app/src-tauri/capabilities/default.json` | `shell:allow-*` for sidecar + `dialog:allow-open`; window globs: `diff-*`, `conflict-*` |
| `app/src-tauri/binaries/sidecar-<triple>.exe` | Built sidecar binary — Tauri requires the target triple suffix |
| `sidecar/main.go` | NDJSON JSON-RPC dispatcher; `wrap1` generic helper; `toRPCError` bridge; `git.cherryPick` uses manual handler to inject progress callback |
| `sidecar/internal/rpc/server.go` | NDJSON JSON-RPC 2.0 transport; BOM stripping; dispatch loop; `ProgressWriter` + `WriteProgress` |
| `sidecar/internal/git/` | Git operations layer (20 files) — see sidecar/README.md |
| `sidecar/go.mod` | Module `github.com/lazy-cherry-pick/sidecar`; Go 1.23 |
| `docs/IPC.md` | Full protocol spec — method signatures, types, error codes |

## Commands

Fresh PowerShell sessions on Windows often lack cargo/go on PATH. Always prepend:
```powershell
$env:Path = "$env:USERPROFILE\.cargo\bin;C:\Program Files\Go\bin;" + $env:Path
```

**Rebuild sidecar** (required after editing any `sidecar/*.go`):
```powershell
$triple = (rustc -vV | Select-String "host:").ToString().Split(":")[1].Trim()
Set-Location sidecar
go build -o ..\app\src-tauri\binaries\sidecar-$triple.exe .
```

**Run Go integration tests:**
```powershell
Set-Location sidecar
go test ./internal/git/ -v
```

**Sidecar smoke test** (no Tauri — verify Go layer in isolation):
```powershell
$s = ".\app\src-tauri\binaries\sidecar-x86_64-pc-windows-msvc.exe"
'{"jsonrpc":"2.0","id":1,"method":"ping"}' | & $s
'{"jsonrpc":"2.0","id":2,"method":"git.status","params":{"repo":"C:\\path\\to\\repo"}}' | & $s
```

**Rust compile check** (no Vite needed):
```powershell
Set-Location app\src-tauri
cargo build
```

**Dev server (full app)** — must be run in the user's own terminal (see hard rule #1):
```powershell
.\dev.ps1
# or: cd app && npm run tauri dev
```

## Hard rules for AI agents

1. **Never run `npm run tauri dev` via a background/agent task.** Gets terminated when parent shell exits. Always ask the user to run it in their own terminal. (Verified during M1.)

2. **After editing `sidecar/*.go`, rebuild the binary to `app/src-tauri/binaries/sidecar-<triple>.exe`.** Tauri bundles the binary at dev time — source changes are invisible until rebuild. On this machine the triple is `x86_64-pc-windows-msvc`.

3. **When adding RPC methods:** (a) implement in `sidecar/internal/git/`, (b) register in `sidecar/main.go` via `s.Register("name", wrap1(...))` — except methods that need progress streaming, which use a manual handler (see `git.cherryPick` pattern), (c) update `docs/IPC.md`, (d) add TypeScript types to `rpc-types.ts`, (e) add typed wrapper to `rpc.ts`. No Rust change needed per method.

4. **JSON over stdio is NDJSON** — one JSON object per line. Never embed raw `\n` inside a JSON value. Go's `json.Encoder.Encode` always appends `\n` — keep using it.

5. **Strip UTF-8 BOM on every input line.** `rpc/server.go` does `bytes.TrimPrefix(line, utf8BOM)`. Don't remove — Windows PowerShell pipes prepend BOM.

6. **Target branch model is Model B with default = current HEAD.** Sidecar always checks dirty tree before applying. If `target != current`, checkout happens only after dirty-tree check passes. Implemented and tested in `cherrypick.go`.

7. **The sidecar is the only layer that shells out to `git`.** No `git` calls in Rust, no direct `git_*` invocations from Svelte. Path: Frontend → `rpc.ts` → `invoke('sidecar_call')` → Rust `sidecar_call` → Go sidecar → `git`.

8. **The frontend has no `@tauri-apps/plugin-shell` dependency.** Svelte uses `@tauri-apps/plugin-dialog` (for folder picker) and our own `sidecar_call` command only. Never add `plugin-shell` to Svelte — it would expose shell APIs to the WebView.

## Architecture invariants

- **One sidecar process per RPC call (current model).** `sidecar_call` does `spawn → write 1 line → read N lines → kill`. Progress-streaming calls (`git.cherryPick`) emit multiple `progress` lines before the final `result` line. Rust reads in a loop, emits Tauri event `cp-progress` for each progress line, then resolves on the final line. ~30–50 ms overhead per call.
- **Concurrent sidecar calls are safe.** `ActiveSidecar` is `Mutex<HashMap<u64, CommandChild>>` with a per-call atomic `call_id`. Each call stores its child by `call_id` and removes only its own entry on finish — concurrent calls from multiple windows (e.g. main + conflict editor) cannot kill each other's processes. `sidecar_cancel` drains the entire map.
- **Progress protocol:** intermediate lines carry `"progress"` field (not `"result"/"error"`) with the same `id`. Rust detects by `parsed.get("progress").is_some()`. Frontend listens via `@tauri-apps/api/event` `listen("cp-progress", ...)`.
- **Cancel:** `sidecar_cancel` Tauri command drains `ActiveSidecar` (`Mutex<HashMap<u64, CommandChild>>`), killing all active children. Frontend then calls `git.abort` via a new sidecar spawn to run `git cherry-pick --abort`.
- **Recent repos:** stored in `%APPDATA%/com.lazycherrypick.app/recents.json` by Rust commands `recents_load`/`recents_save`. Max 10 entries. Not routed through the Go sidecar.
- **App settings:** stored in `%APPDATA%/com.lazycherrypick.app/settings.json` by Rust `settings_load`/`settings_save`. Fields: `maxCommits`, `defaultApplyMode`, `showEolMarkers`, `autoFetchOnOpen`, `theme`, `externalDiffEnabled`, `externalDiffPath`, `externalDiffArgs`, `externalMergeEnabled`, `externalMergePath`, `externalMergeArgs`. Returns defaults on missing/corrupted file. All fields have `#[serde(default)]` — backward compatible.
- **External tool launch:** `launch_detached(program, args)` — fire-and-forget for diff; `launch_and_wait(program, args)` — async block until tool exits for merge. Both are Rust Tauri commands using `tokio::process::Command`. `detect_external_tools()` checks 5 known paths (TortoiseGit, BC3, BC4, WinMerge, VSCode) and returns found tools as `[{name, path}]`.
- **Git command log:** sidecar emits `[GIT_CMD] git <args>` (without `-C dir`) and `[GIT_INFO] git.method → branch` to stderr. Rust batchs stdout (RPC protocol) and stderr separately — stderr lines with those prefixes are emitted as Tauri `git-log` events and appended to `%APPDATA%/com.lazycherrypick.app/git.log`. Format per line: `<type> <unix-ts> <content>` where type is `cmd` or `info`.
- **Sidecar logs to stderr, responses to stdout.** Don't write logs to stdout.
- **Capabilities are scoped to `binaries/sidecar`.** Don't broaden to `shell:default`.
- **Write operations require explicit user action.** `git.cherryPick` is only triggered by the Apply button — never called automatically. Read methods (`git.status`, `git.commits`, `git.fetch`) are fine to call on load/branch-change.

## What's done / what's not

**M1 done**: project scaffold, IPC contract validated end-to-end (`ping`, `version`).

**M2 done**: full Go git ops layer — `git.openRepo`, `git.status`, `git.branches`, `git.commits`, `git.cherryPick`. 9 integration tests (all pass). TypeScript typed wrapper (`rpc.ts`, `rpc-types.ts`).

**M3 done**: main UI — Toolbar (file dialog open), CommitList (source branch + multi-select), PickQueue (ordered queue + target dropdown + Apply), ResultBanner (applied/conflict/error). Happy path fully wired end-to-end.

**M3.1 done**: Apply & Push — TortoiseGit-style dropdown on Apply button. `git.push` RPC method. Mode state persists between operations.

**M4 done**:
- **Progress streaming**: `git.cherryPick` emits progress lines (`{"progress": {"n":1,"total":3,"sha":"..."}}`) before final result. Rust reads in loop + emits Tauri event `cp-progress`. `PickQueue` shows progress bar `n/total — sha`.
- **Cancel**: `sidecar_cancel` Tauri command kills current sidecar child. Frontend calls `git.abort` to cleanup git state. Cancel button visible during apply.
- **Recent repos (M4b)**: Rust `recents_load`/`recents_save` commands write `%APPDATA%/lazy-cherry-pick/recents.json`. Toolbar shows "Recent ▾" dropdown (max 10 entries).
- **Fetch / Pull refresh**: `git.fetch` (`git fetch --prune`) and `git.pull` (`git fetch <remote> <branch>:<branch>`, fast-forward only). CommitList header has Fetch/Pull dropdown button.

**M5 mostly done**:
- **M5a — Commit detail panels**: click commit → resizable panel below shows CommitDetail (subject, body, SHA, author, date, parents) + CommitFiles (file list with status badge, +/- stats, click to open diff window). Horizontal resize handle between CommitList and PickQueue panes.
- **M5b — Dry-run conflict preview**: `git.dryRunPick` RPC uses `git apply --3way --check`; ⚠ icon on queue items with predicted conflicts; debounce 400 ms.
- **File diff viewer**: `git.fileDiff` RPC (`git show --unified=99999`); opens in a new Tauri window (`diff-${Date.now()}`); nav buttons ▲/▼ to jump between change blocks. Now **2-panel side-by-side (M5d)** — see below.
- **Create Branch**: `git.createBranch` RPC; Create Branch button in PickQueue target dropdown.
- **M5c — Conflict resolver (done)**: `git.conflictFiles`, `git.resolveConflict`, `git.continueCherry`, `git.fileContent`, `git.writeAndStageFile` RPCs; `ConflictResolver.svelte` with Keep Ours/Use Theirs/Continue/Abort; `/conflict` route — TortoiseGit-style 3-pane merge editor:
  - **Line numbers** in each top pane (independent theirs/ours counters)
  - **Inline conflict-header bar** per block: "Conflict N/M · ← Theirs · T+O" (left) / "Ours → · O+T" (right) — action buttons right at the conflict, no need to use toolbar
  - **Independent drag-select per pane**: `leftSel` + `rightSel` state — selecting on one pane does NOT clear the other; right-click shows cross-pane combine options when both have selections
  - **Context menu**: use selected lines from this pane / use whole block / combine both sides (4 order combinations)
  - **Resizable H-divider**: drag to resize top/bottom ratio (20–80%)
  - **Keyboard**: ↑↓ to navigate conflicts
  - **Provisional resolution model**: clicking Theirs/Ours/T+O sets a *soft choice* in `provisionalChoices: Map<number, string[]>` — `mergedText` keeps conflict markers intact. Bottom pane shows chosen lines in **orange** immediately, but user can click any button again to change. `buildFinalText()` applies all choices only at save time.
  - **Bottom pane (TortoiseGit style)**: provisionally-resolved blocks shown in orange with change buttons (← Theirs / Ours → / T+O) always visible; unresolved blocks shown as **hatched red placeholder** with Ours/Theirs/T+O buttons; click block → jumps top panes to that conflict
  - **Toggle "✎ Raw"**: enables inline contenteditable on `.mv-text` spans (visual mode + direct edit). Yellow highlight on edited lines. Exit syncs DOM back only if `rawEdited` was set (avoids overwriting state when user just looked at raw mode).
  - **Save status**: button "Lưu & Stage" disabled when `hasUnresolved || applying || saved`; after save shows "✓ Đã lưu" badge instead of auto-closing window.
  - Flow: cherry-pick stops in conflict state → ConflictResolver shown → click file → conflict window opens → resolve each block (changeable) → Lưu & Stage → click Continue → `continueCherry` applies the staged commit + remaining queue commits via `applyPickShas()`.

- **M5d — Polish & diff viewer redesign (done)**:
  - **Project rename**: `cherry-pick-git` → `Lazy Cherry Pick`. Tauri productName/identifier updated; AppData path is now `lazy-cherry-pick/recents.json`; Go module is `github.com/lazy-cherry-pick/sidecar`.
  - **Staged diff for resolved files**: `git.stagedFileDiff` RPC (`git diff --cached --unified=99999 -- <file>`); ConflictResolver opens this in a diff window (gray button) when a file is already resolved, instead of the merge editor.
  - **2-panel side-by-side diff viewer (TortoiseGit-style)**: complete rewrite of `FileDiff.svelte`. Layout: left/right panels with independent labels (`Before`/`{sha}` for commit diff, `HEAD`/`Staged` for staged diff), synced scroll, hatched filler rows when one side has no matching line, hunk header spans both panels.
  - **EOL markers (Notepad++ style)**: toggle `¶` button shows `LF` (blue) / `CRLF` (orange) at end of each line. Parser auto-detects trailing `\r` and handles `\ No newline at end of file` as `eol="none"` (no marker rendered).
  - **Trailing-newline transform (Option D + lookahead)**: when a paired `change` row has `leftText === rightText` but EOL differs ("none" vs "lf"/"crlf"), convert to `ctx` row + phantom empty `change` row (representing the implicit empty line that appears/disappears with the trailing newline). Lookahead skips the phantom if the next row is already an empty del/add — prevents double-counting when the diff covers both an empty line AND the trailing newline. Matches TortoiseGit's "editor view" rather than git's "lost trailing newline" view. See `design-side-by-side-diff` memory.
  - **Raw-mode save fix in conflict editor**: setting `rawEdited = false` after successful save prevents the "blank screen" bug when toggling back to visual mode.

**M6 done**:
- **Checkbox hit area fix**: Wrapped `<input type="checkbox">` in `<label class="cb-wrap">` with extra padding — click area ~36×36px instead of 14×14px. Row `align-items` changed to `center`.
- **BranchSelect component** (`app/src/lib/BranchSelect.svelte`): Custom searchable dropdown replacing native `<select>` in both CommitList (source) and PickQueue (target). Type to filter by name; ↑↓/Enter/Escape keyboard nav; click-outside closes.
- **M6a — Commit filter bar**: Collapsible filter bar in CommitList header. Fields: Author, Message (keyword/regex), Since, Until, Path (glob). `Filter (N)` badge button toggles bar; Apply/Clear; Enter key submits. Filter state held in `+page.svelte` as `activeFilter: CommitFilter`; passed to `rpc.git.commits(..., filter)`; resets on source branch change.
- **M6b — Filter presets**: Save/load named filter presets stored in `localStorage` key `lcp-filter-presets`. "+ Save" button → inline name input → Enter to save. "Presets (N) ▾" dropdown lists saved presets; click to load; trash icon to delete. Persists across app restarts. **Active preset UX** (post-M6 polish): Presets button shows active preset name (highlighted accent color) instead of count; when preset is loaded and filter fields are modified, "+ Save" becomes a "Save | ▾" split-button — left click overwrites the active preset, ▾ opens dropdown with "Save [name]" (overwrite) and "Save as…" (new name input). State: `activePresetName: string | null`, `filtersDirtyFromPreset: boolean` (derived).
- **M6c — Commit badges**: `getBadge(subject)` in CommitList detects conventional commit prefix (`feat|fix|docs|style|refactor|perf|test|build|ci|chore`) and JIRA pattern (`[ABC-123]`); strips prefix from displayed subject. Colors: feat=green, fix=red, docs=blue, test=purple, perf=yellow, chore/…=gray, breaking `!`=dark red, JIRA=light blue. Uses `{@const badge = getBadge(c.subject)}` at `{#each}` level.
- **Bug fix — root commit loading**: Root commit `nil` parents slice → JSON `null` → `detail.parents.length` TypeError in Svelte → component stuck at "Loading...". Fix: Go initializes `parents := []string{}`; frontend adds `(detail.parents ?? [])` guard in `CommitDetail.svelte`.
- **Build & CI fixes**: `build/build.ps1` — fixed project name in comment, added PATH setup, added `go test` gate. `releases/*.md` — fixed path `.\build.ps1` → `.\build\build.ps1`. `.github/workflows/release.yml` — Go `1.21`→`1.23`, Node `lts/*`→`'22'`, `tauri-action@v0`→`v0.5`, added `go test` step.
- **gitignore**: Added `app/src-tauri/binaries/*.exe` — sidecar binary is a build artifact, not tracked in git.
- **TestCherryPick_Conflict updated**: Test was written for M4 behavior (auto-abort on conflict). M5 leaves repo in conflict state. Test now asserts dirty after conflict → calls `Abort()` → asserts clean.

**M7 done**:
- **M7a — Settings panel**: `AppSettings` struct in Rust (`maxCommits`, `defaultApplyMode`, `showEolMarkers`, `autoFetchOnOpen`, `theme`). `settings_load`/`settings_save` Tauri commands write `%APPDATA%/com.lazycherrypick.app/settings.json`; returns defaults on missing/corrupted file. `Settings.svelte` modal with number input, select, toggle switches, Dark/Light segmented control. Gear button in `Toolbar.svelte`. `+page.svelte` applies settings: `$effect` toggles `body.light` class; `loadCommits` uses `settings.maxCommits`; `openRepo` calls `git.fetch` if `autoFetchOnOpen`; PickQueue uses `defaultApplyMode`; FileDiff uses `initialShowEol`.
- **Theme (dark/light)**: CSS custom properties (`--surface`, `--surface-elevated`, `--border`, `--toolbar-bg`, `--input-bg`, `--hover`, `--selected`, `--accent`, `--text`, `--text-secondary`, `--text-muted`) defined on `:global(body)` for dark; `body.light` block overrides all. Sub-windows (`diff/`, `conflict/`) load settings independently in `onMount` and apply class to `document.body`. Conflict editor panes kept dark regardless of theme (code readability). `var(--surface-elevated)` used in all dropdown menus across components.
- **English UI**: All Vietnamese text translated to English across all `.svelte` files.
- **Git Console**: `GitConsole.svelte` panel (180px, always dark). Toggle button (`>_`) in Toolbar with active state. Go `exec.go` logs `[GIT_CMD] git <args>` (without `-C dir`) to stderr before each git exec. `server.go` logs `[GIT_INFO] git.method → target/branch` to stderr before each `git.*` dispatch. Rust parses both prefixes from `CommandEvent::Stderr`, emits `git-log` event `{ts, type, cmd}`, appends `<type> <ts> <cmd>` to `git.log`. Frontend loads history via `git_log_read` on mount, listens `git-log` for realtime. Renders `info` entries as group headers (method + branch), `cmd` entries as indented lines with `git` (dim) + subcommand (cyan bold) + args (gray). Auto-scroll with pause when user scrolls up. `git_log_clear` Tauri command truncates the file.

**M8 done**:
- **External diff viewer**: Settings → External Tools → enable + set path/args → clicking a file in CommitFiles now calls `git.extractDiffFiles` (Go sidecar extracts `SHA^:file` and `SHA:file` to `lcp-diff-*` temp dir), builds CLI args from template (`{left}`, `{right}`, `{leftLabel}`, `{rightLabel}`), calls `launch_detached` Rust command. Temp files are NOT cleaned — OS cleans on reboot.
- **External merge tool**: Settings → External Tools → enable + set merge path/args → clicking a conflict file calls `git.extractConflictFiles` (extracts `:1:` base, `:2:` ours, `:3:` theirs, working-tree copy as output to `lcp-merge-*` temp dir), calls `launch_and_wait` (blocks until tool closes), then `git.stageResolvedFile` writes output back + stages it, then `git.cleanupTmpDir` removes temp dir.
- **Args template**: `buildArgs()` helper in `+page.svelte` substitutes `{key}` placeholders then splits respecting quoted tokens. Critical: regex uses `([^\s"]*)` (NOT `\S*`) as unquoted prefix — `\S` includes `"` causing backtracking that embeds `"` chars in args → Rust's CreateProcessW escaping corrupts paths (`D:\C:\...` prefix, `\` suffix).
- **Conflict marker guard**: `StageResolvedFile` in `externaltool.go` checks `bytes.Contains(data, []byte("<<<<<<<"))` before staging — prevents auto-staging when merge tool is closed without resolving all conflicts.
- **TortoiseGit tool split**: `TortoiseGitProc.exe` for diff; `TortoiseGitMerge.exe` (same `bin/` dir) for 3-way merge. Auto-detect fills merge path via string replace. `/command:merge` on TortoiseGitProc.exe opens wrong dialog.
- **TortoiseGit diff direction**: `/path:` arg → RIGHT pane; `/path2:` arg → LEFT pane (opposite of naming). Correct: `/command:diff /path2:"{left}" /path:"{right}"`.
- **Auto-detect auto-fills args**: `applyDetected()` in `Settings.svelte` fills `TOOL_DIFF_ARGS[name]` / `TOOL_MERGE_ARGS[name]` constants when user clicks a detected tool pill.
- **Browse button**: `@tauri-apps/plugin-dialog` `open()` with exe filter — lets user pick .exe without typing path.
- **FileDiff.svelte status-aware empty messages**: status "A" → "File did not exist" (left) / "New empty file" (right); status "D" → "Empty file" / "File was deleted". Fixes "No changes." shown for newly-added empty files.
- **conflict/+page.svelte async onMount fix**: wrapped async init in IIFE, returned sync cleanup — fixes TypeScript error from Svelte's `onMount` typing.

**Not done**: packaging, signing.

See README roadmap for M5+ scope.

## Tech stack pins (as installed and verified)

- Node 22.17.0, npm 10.9.2
- Rust 1.95.0 stable (`x86_64-pc-windows-msvc`)
- Go 1.26.3
- Git 2.50 (Git for Windows)
- Tauri 2.11, `tauri-plugin-shell` 2.3, `tauri-plugin-opener` 2.5, `tauri-plugin-dialog` 2.x
- Svelte 5.x, SvelteKit 2.9, Vite 6.4
- MSVC Build Tools 2022 at `C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools`

## Cross-session memory

When working with the user on this repo, prior decisions are stored as Claude memories:
- `user-language-vi` — communicate in Vietnamese, prefer short option-style questions
- `project-m4-complete`, `project-m5-complete`, `project-m5c-wip`, `project-m5d-complete`, `project-m6-complete`, `project-m7-complete`, `project-m8-complete` — feature checklists per milestone
- `design-conflict-merge-logic` — conflict merge editor (provisional model, parser, render pipeline)
- `design-side-by-side-diff` — 2-panel diff viewer logic incl. trailing-newline transform with lookahead

These memories complement (not replace) this file. This file describes the **codebase**; memories describe the **user and decisions**.

## When stuck

- Sidecar weirdness → run it manually with a piped JSON request to isolate from Tauri (rule #5 may apply if first response is a parse error).
- Rust compile errors after editing `lib.rs` → check plugin version in `Cargo.toml` and the plugin docs.
- Tauri window blank → check the Vite log in the terminal where `tauri dev` runs before assuming Rust is broken.
- "Method not found" with correct method name → did you rebuild the sidecar binary? Rule #2.
- Frontend type error after adding a new RPC method → add types to `rpc-types.ts` and wrapper to `rpc.ts` first.
