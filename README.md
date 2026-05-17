# cherry-pick-git

A desktop Git client focused on **batch cherry-pick workflows**: multi-select commits across branches, predict and preview conflicts, apply with smart filters.

> Status: **M5 partially complete** — commit detail panels, dry-run conflict preview, file diff viewer, conflict resolver UI (in progress). M4 features (progress bar, cancel, recent repos, Fetch/Pull) are all done.

## Why

`git cherry-pick` works one commit at a time and gives no preview. Tools like GitKraken/Sourcetree treat it as a 1-click operation, not a workflow. This project treats batch cherry-pick as the primary workflow: pick many commits, dry-run conflicts, define auto-resolve rules, apply atomically with rollback.

## Architecture

Hybrid: **Tauri 2** (Rust + Svelte 5) desktop shell drives a **Go sidecar** over newline-delimited JSON-RPC 2.0 on stdin/stdout. The sidecar shells directly to `git` CLI — no library, no fork.

```
+----------------------------+
|  Tauri desktop window      |
|  +----------+ +----------+ |
|  | Svelte 5 |<-> Rust    | |
|  | frontend |  | backend | |
|  +----------+ +-----+----+ |
+----------------------+-----+
                       | spawn + stdio (one process per call, multi-line for progress)
                       v
                +------------------+
                | Go sidecar       |   only this layer
                | JSON-RPC server  | — shells out to `git` —
                +------------------+
                       |
                       v
                    `git` CLI
```

## Layout

```
.
├── app/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── rpc-types.ts          # TypeScript types (mirrors Go types)
│   │   │   ├── rpc.ts                # Typed invoke wrapper
│   │   │   ├── Toolbar.svelte        # Repo path + Open (file dialog)
│   │   │   ├── CommitList.svelte     # Source branch + multi-select commit list
│   │   │   ├── PickQueue.svelte      # Pick queue + target dropdown + Apply/Create Branch
│   │   │   ├── ResultBanner.svelte   # Applied / conflict / error result
│   │   │   ├── CommitDetail.svelte   # Commit subject, body, SHA, author, date, parents
│   │   │   ├── CommitFiles.svelte    # File list for a commit (+/- stats, click to diff)
│   │   │   ├── FileDiff.svelte       # Unified diff renderer with change block navigation
│   │   │   └── ConflictResolver.svelte # Conflict file list — Keep Ours / Use Theirs / Continue
│   │   └── routes/
│   │       ├── +page.svelte          # Main orchestrator (all app state)
│   │       ├── diff/+page.svelte     # File diff viewer window
│   │       └── conflict/+page.svelte # 3-pane conflict merge editor
│   └── src-tauri/
│       ├── src/lib.rs                # sidecar_call (HashMap concurrency), sidecar_cancel, recents
│       ├── Cargo.toml                # tauri-plugin-shell, dialog, opener, tokio
│       ├── capabilities/default.json # shell:allow-* + dialog:allow-open; diff-* conflict-* windows
│       └── binaries/                 # sidecar-<triple>.exe
├── sidecar/
│   ├── main.go                       # Handler registration, wrap1 helper
│   ├── go.mod
│   └── internal/
│       ├── rpc/server.go             # NDJSON JSON-RPC 2.0 transport
│       └── git/                      # git ops: 19 files covering all RPC methods
├── docs/IPC.md                       # Protocol spec (all methods + types)
├── dev.ps1                           # One-liner dev launcher (Windows)
└── CLAUDE.md                         # AI dev context (read first)
```

## Quick start

### Prerequisites (Windows)

- Node.js 18+, Rust stable (MSVC), Go 1.21+, Git, MSVC Build Tools 2022, WebView2

```powershell
winget install Rustlang.Rustup GoLang.Go OpenJS.NodeJS Git.Git
winget install Microsoft.VisualStudio.2022.BuildTools --override `
  "--quiet --wait --norestart --nocache --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"
rustup default stable
```

### Build & run

```powershell
$env:Path = "$env:USERPROFILE\.cargo\bin;C:\Program Files\Go\bin;" + $env:Path

# 1) Build Go sidecar (Tauri requires target-triple suffix in filename)
$triple = (rustc -vV | Select-String "host:").ToString().Split(":")[1].Trim()
Set-Location sidecar
go build -o ..\app\src-tauri\binaries\sidecar-$triple.exe .
Set-Location ..

# 2) Install frontend deps (first run only)
Set-Location app && npm install

# 3) Dev server
npm run tauri dev
```

Or run `.\dev.ps1` which handles PATH and launches the dev server.

### Run Go tests

```powershell
$env:Path = "C:\Program Files\Go\bin;" + $env:Path
Set-Location sidecar
go test ./internal/git/ -v
```

Integration tests use real temp git repos (no mocks):

| Test | Covers |
|---|---|
| TestOpen_NotARepo | -32004 on non-repo path |
| TestStatus_Clean / TestStatus_Dirty | Branch, unstaged, untracked detection |
| TestBranches | Local branch list, HEAD flag |
| TestCommits / TestCommits_FilterMessage | Log parsing, --grep filter |
| TestCherryPick_Happy | Apply commit, verify new HEAD |
| TestCherryPick_DirtyTreeBlocked | -32002 before touching target |
| TestCherryPick_Conflict | -32003, repo left in conflict state |

## Usage (M5 UI)

1. Click **Open repo** → pick any Git repository folder (or use **Recent ▾** dropdown in toolbar)
2. **Source branch** dropdown (left) — select the branch to pick from
3. **Fetch ▾** button — fetch latest commits from remote; dropdown lets you switch to **Pull** (fast-forward only)
4. **Tick commits** in the list — they appear in the Pick queue (right), in selection order
   - ⚠ icon on queued commits means a conflict is predicted (dry-run preview, debounced)
   - Click a commit to open the **detail panel** below (message body + file list with +/- stats)
   - Click a file in the detail panel to open the **diff viewer** in a new window
5. **Target branch** dropdown (right) — where commits will land (default: current branch); **Create Branch** button to create a new branch on the fly
6. Click **Apply** (or **Apply & Push** via the ▾ dropdown) — runs `git cherry-pick` sequentially:
   - Progress bar shows `n/total — sha` for each commit as it applies
   - **Cancel** button aborts mid-batch and restores the repo to a clean state
   - On conflict: banner shows ⚠️ with file list → **ConflictResolver** panel appears
     - **Keep Ours / Use Theirs** — one-click resolution per file
     - Click filename → opens **3-pane merge editor** (TortoiseGit style):
       - Left pane = Theirs (CHERRY_PICK_HEAD), Right pane = Ours (HEAD), each with line numbers
       - **Inline action bar** on each conflict block: "← Theirs", "Ours →", "T+O", "O+T" — no need to use toolbar
       - **Independent drag-select** on each pane — select lines from both sides then right-click to combine
       - **Context menu**: use selected lines / use whole block / combine both sides in any order; cross-pane combine when both panes have selection
       - **Bottom pane**: resolved lines shown normally; unresolved blocks shown as hatched red placeholder with quick-resolve buttons
       - Resizable top/bottom divider; ↑↓ keyboard navigation between conflicts
     - **Continue →** after all files resolved; **Abort** to cancel
   - Result shown in banner: ✅ applied / ⚠️ conflict / 🔴 error

## Smoke-testing sidecar without Tauri

```powershell
$s = ".\app\src-tauri\binaries\sidecar-x86_64-pc-windows-msvc.exe"
'{"jsonrpc":"2.0","id":1,"method":"ping"}' | & $s
'{"jsonrpc":"2.0","id":2,"method":"git.branches","params":{"repo":"C:\\path\\to\\repo"}}' | & $s
```

See [docs/IPC.md](./docs/IPC.md) for all method signatures.

## Design decisions

- **Target branch Model B** — dropdown defaults to current branch, user can pick any; dirty-tree guard fires before checkout.
- **Sidecar is the only `git` caller** — Rust and Svelte never shell to git directly.
- **Direct git CLI, no library** — considered lazygit fork, too much coupling. ~600 lines of clean, testable Go.
- **One process per RPC call** — 30–50 ms overhead acceptable for user-initiated ops. Concurrent calls from multiple windows (main + conflict editor) are safe: `ActiveSidecar` is `Mutex<HashMap<u64, CommandChild>>` with per-call atomic IDs so calls can't kill each other.
- **Progress streaming** — `git.cherryPick` emits intermediate `progress` lines before the final `result`. Rust reads in a loop and forwards them as Tauri events (`cp-progress`) to the frontend.
- **Leave-in-conflict state** — on first conflict, cherry-pick stops but does not abort. The repo stays in conflict state so the frontend can drive a 3-way resolver. `git.continueCherry` completes the pick after all files are resolved.

## Roadmap

- ✅ **M1** Scaffold: Tauri + Svelte + Go, JSON-RPC stdio, IPC validated end-to-end
- ✅ **M2** Git ops layer: `git.status/branches/commits/cherryPick`, integration tests, TypeScript wrapper
- ✅ **M3** Main UI: Toolbar + CommitList + PickQueue + ResultBanner, Apply flow wired end-to-end
- ✅ **M3.1** Apply & Push: TortoiseGit-style mode dropdown, `git.push` RPC method
- ✅ **M4** Progress & UX: per-commit progress bar, Cancel/abort, recent repos list, Fetch/Pull refresh dropdown
- ⏳ **M5** Commit detail + Conflict resolver *(partially done)*:
  - ✅ M5a — click commit → CommitDetail + CommitFiles panels (resizable); file diff viewer window
  - ✅ M5b — dry-run conflict preview (`git.dryRunPick`), ⚠ icons in queue; debounce 400 ms
  - ✅ Create Branch button in target dropdown
  - 🔧 M5c — conflict resolver: ConflictResolver panel + TortoiseGit-style 3-pane merge editor (line numbers, inline action bars, independent dual-pane selection, hatched result pane); end-to-end testing still needed
- ⏳ **M6** Smart filter: author, message regex, file glob, date range, conventional commits, JIRA pattern, trailer labels; named filter presets

## License

TBD. Planning **MIT**.

## See also

- [CLAUDE.md](./CLAUDE.md) — AI dev context (read first when entering the repo)
- [docs/IPC.md](./docs/IPC.md) — JSON-RPC protocol spec
- [sidecar/README.md](./sidecar/README.md) — sidecar build, test, smoke test
