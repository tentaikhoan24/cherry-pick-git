# sidecar

Go binary speaking newline-delimited JSON-RPC 2.0 on stdin/stdout. Spawned by the Tauri Rust backend as a child process; all git operations go through here.

## Build

```powershell
# Build standalone (for smoke testing)
go build -o sidecar.exe .

# Build into Tauri binaries dir (required for the desktop app)
$triple = (rustc -vV | Select-String "host:").ToString().Split(":")[1].Trim()
go build -o ..\app\src-tauri\binaries\sidecar-$triple.exe .
```

## Run tests

```powershell
go test ./internal/git/ -v
```

9 integration tests against real temp git repos. All should pass in ~25 s.

## Smoke test (no Tauri needed)

```powershell
# Liveness
'{"jsonrpc":"2.0","id":1,"method":"ping"}' | .\sidecar.exe

# Versions
'{"jsonrpc":"2.0","id":2,"method":"version"}' | .\sidecar.exe

# Repo status
'{"jsonrpc":"2.0","id":3,"method":"git.status","params":{"repo":"C:\\path\\to\\repo"}}' | .\sidecar.exe

# Branch list
'{"jsonrpc":"2.0","id":4,"method":"git.branches","params":{"repo":"C:\\path\\to\\repo"}}' | .\sidecar.exe

# Commits on a branch
'{"jsonrpc":"2.0","id":5,"method":"git.commits","params":{"repo":"C:\\path\\to\\repo","ref":"main","limit":10}}' | .\sidecar.exe
```

## Package structure

```
sidecar/
├── main.go              # Handler registration; wrap1 generic helper; toRPCError bridge
│                        # git.cherryPick uses manual handler (injects progress callback)
├── go.mod
└── internal/
    ├── rpc/
    │   └── server.go    # NDJSON JSON-RPC 2.0 transport; BOM stripping; dispatch loop
    │                    # ProgressWriter / WriteProgress — context-injected progress emitter
    └── git/
        ├── errors.go    # *Error type; error codes (-32001 to -32004)
        ├── exec.go      # run() — git -C <dir> <args>; stderr + exitCode in error Data
        ├── types.go     # Status, FileStatus, Branch, Commit, CommitFilter, CherryPickResult, ConflictInfo
        ├── repo.go      # Open(); CurrentBranch(); IsDirty()
        ├── status.go    # git status --porcelain=v2 --branch parser
        ├── branches.go  # git for-each-ref parser
        ├── commits.go   # git log NUL/RS-delimited parser; server-side filter flags
        ├── cherrypick.go# Sequential apply; leave-in-conflict; dirty-tree guard; OnProgress callback
        ├── abort.go     # git cherry-pick --abort (idempotent)
        ├── fetch.go     # git.fetch (fetch --prune) + git.pull (fetch <branch>:<branch>)
        ├── push.go      # git push <remote> <branch>
        ├── createbranch.go  # git checkout -b <name> [<startPoint>]
        ├── commitdetail.go  # git log -1 full metadata (subject + body)
        ├── commitfiles.go   # git diff-tree --root -M --numstat --name-status
        ├── dryrun.go        # git apply --3way --check (conflict prediction, no working tree change)
        ├── filediff.go      # git show --unified=99999 (full file diff for a commit)
        ├── stageddiff.go    # git diff --cached --unified=99999 (staged file diff, M5d)
        ├── conflict.go      # ConflictFiles (git status --porcelain), ResolveConflict, ContinueCherry
        ├── workingfile.go   # FileContent (os.ReadFile), WriteAndStageFile (os.WriteFile + git add)
        └── git_test.go      # Integration tests against real temp git repos
```

## Methods

| Method | Purpose |
|---|---|
| `ping` | Liveness — returns `"pong"` |
| `version` | Sidecar + Go + Git versions |
| `git.openRepo` | Validate repo path; return branch + detached flag + cherryPickHead if in-progress |
| `git.status` | Branch, ahead/behind, staged/unstaged/untracked files |
| `git.branches` | Local (+ optional remote) branch list |
| `git.commits` | Paginated commit log with optional filter |
| `git.cherryPick` | Batch apply onto target branch; leaves in conflict state on first conflict; streams progress |
| `git.abort` | `git cherry-pick --abort` — idempotent cleanup |
| `git.fetch` | `git fetch --prune <remote>` — update remote-tracking refs, no local branch change |
| `git.pull` | `git fetch <remote> <branch>:<branch>` — fast-forward local branch without checkout |
| `git.push` | `git push <remote> <branch>` |
| `git.createBranch` | `git checkout -b <name>` at optional start point |
| `git.commitDetail` | Full commit metadata including body text |
| `git.commitFiles` | Files changed in a commit with status and line counts |
| `git.dryRunPick` | Predict conflicts via `git apply --3way --check` without touching working tree |
| `git.fileDiff` | Full unified diff for a single file in a commit |
| `git.stagedFileDiff` | Unified diff for a staged file (`git diff --cached`) — used by ConflictResolver to preview staged content |
| `git.conflictFiles` | List files currently in conflict state (all types: UU/AA/DD/AU/UA/DU/UD) |
| `git.resolveConflict` | Resolve a conflict file by checking out ours/theirs and staging |
| `git.continueCherry` | `git cherry-pick --continue --no-edit` after all conflicts resolved |
| `git.fileContent` | Read working-tree file content |
| `git.writeAndStageFile` | Write content to working-tree file and stage it |

Full request/response types: see [../docs/IPC.md](../docs/IPC.md).

## Error codes

| Code | Meaning |
|---|---|
| -32001 | Git command failed |
| -32002 | Working tree dirty |
| -32003 | Cherry-pick conflict (partial result in `data`) |
| -32004 | Not a git repo |
