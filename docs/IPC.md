# IPC Protocol

The Tauri Rust backend talks to the Go sidecar over **newline-delimited JSON-RPC 2.0** on stdin/stdout.

## Transport

- One **request** per line on the sidecar's stdin (JSON object + `\n`)
- One **response** per line on the sidecar's stdout (JSON object + `\n`)
- Logs, panics, and warnings go to stderr (not parsed by the Rust caller)
- UTF-8 BOM at the start of any input line is stripped (Windows pipe tolerance)
- The Rust caller (`sidecar_call`) spawns one process per request, sends one line, then reads lines in a loop until a `result` or `error` line arrives. Lines carrying a `progress` field (same `id`) are emitted as Tauri events (`cp-progress`) and are not the final response.
- Concurrent calls from multiple windows are safe: each call gets a unique `call_id` (atomic u64) and stores its child process under that ID. Calls clean up only their own child on finish.

## Request shape

```json
{ "jsonrpc": "2.0", "id": 1, "method": "version", "params": null }
```

- `jsonrpc` — always `"2.0"`
- `id` — integer or string; echoed in the response. Required (no notifications today).
- `method` — string; dot notation namespaces (`git.status`, `app.config`)
- `params` — optional, `null` or object/array

## Response — success

```json
{ "jsonrpc": "2.0", "id": 1, "result": { "sidecar": "0.2.0", "go": "go1.26.3", "git": "git version 2.50.0" } }
```

## Progress notification (M4)

For long-running methods (`git.cherryPick`), the sidecar emits zero or more intermediate lines before the final result. Each progress line shares the request `id` and carries a `progress` field instead of `result`/`error`:

```json
{ "jsonrpc": "2.0", "id": 1, "progress": { "n": 1, "total": 3, "sha": "abc1234" } }
```

The Rust `sidecar_call` handler detects progress lines (`parsed.get("progress").is_some()`) and emits them as Tauri events (`cp-progress`) without resolving the pending call. The final `result`/`error` line closes the call.

## Response — error

```json
{ "jsonrpc": "2.0", "id": 1, "error": { "code": -32601, "message": "Method not found: foo" } }
```

## Error codes

Standard JSON-RPC 2.0 codes:

| Code     | Meaning |
|----------|---------|
| -32700   | Parse error (malformed JSON) |
| -32600   | Invalid request |
| -32601   | Method not found |
| -32602   | Invalid params |
| -32603   | Internal error |

Application-specific codes (reserved range -32000 to -32099):

| Code     | Meaning |
|----------|---------|
| -32001   | Git command failed (stderr in `data.stderr`, exit code in `data.exitCode`) |
| -32002   | Working tree dirty — refuse target switch / cherry-pick |
| -32003   | Cherry-pick produced conflicts — partial result in `data.applied`, conflict list in `data.conflicts` |
| -32004   | Repo not found / not a git repo |

## Methods — implemented (M1)

| Method  | Params | Result | Purpose |
|---------|--------|--------|---------|
| `ping`    | —    | `"pong"` | Liveness |
| `version` | —    | `{ sidecar: string, go: string, git: string }` | Validates `git` is on PATH and reports versions |

## Methods — implemented (M2)

All M2 methods take `{ repo: string }` (absolute path to repo root) plus method-specific fields.

### `git.openRepo`

Params: `{ repo: string }`

Result:
```ts
{ path: string; branch: string; detached: boolean; cherryPickHead?: string }
```

Validates that `path` is a git repo (via `git rev-parse --show-toplevel`). Returns the resolved repo root, current branch name, and detached HEAD flag. If a cherry-pick is in progress (`.git/CHERRY_PICK_HEAD` exists), `cherryPickHead` contains the SHA being applied — the frontend uses this to auto-restore the ConflictResolver on reload.

Errors: `-32004` if not a repo.

---

### `git.status`

Params: `{ repo: string }`

Result:
```ts
{
  branch: string;
  upstream: string;       // empty if no upstream
  ahead: number;
  behind: number;
  dirty: boolean;
  staged: FileStatus[];
  unstaged: FileStatus[];
  untracked: string[];
  detached: boolean;
}

type FileStatus = { path: string; status: string }
// status values: "M" modified, "A" added, "D" deleted, "R" renamed, "C" copied, "U" unmerged
```

---

### `git.branches`

Params: `{ repo: string; includeRemote?: boolean }`

Result:
```ts
Branch[]

type Branch = {
  name: string;
  sha: string;
  isHead: boolean;
  upstream: string;  // empty if none
}
```

`includeRemote: false` (default) returns only local branches. Set to `true` to include `remotes/*`.

---

### `git.commits`

Params:
```ts
{
  repo: string;
  ref?: string;        // default: "HEAD"
  limit?: number;      // default: 100
  skip?: number;
  filter?: CommitFilter;
}
```

Result:
```ts
Commit[]

type Commit = {
  sha: string;
  parents: string[];
  author: string;
  email: string;
  time: number;        // Unix timestamp
  subject: string;
  refs: string[];      // branch/tag decorations
}
```

Filter maps directly to `git log` flags (server-side filtering, no post-processing):

```ts
type CommitFilter = {
  author?: string;            // --author=
  messageContains?: string;   // --fixed-strings --grep=
  since?: string;             // --since= (ISO 8601 or relative like "7 days ago")
  until?: string;             // --until=
  pathGlob?: string;          // pathspec appended after --
}
```

---

### `git.cherryPick`

Params:
```ts
{
  repo: string;
  target: string;      // branch to apply commits onto
  shas: string[];      // ordered list of commit SHAs to apply
  strategy?: "smart" | "theirs" | "ours";  // default: "smart" (no --strategy-option)
}
```

Result:
```ts
{ applied: string[]; conflicts: ConflictInfo[] }

type ConflictInfo = { sha: string; files: string[] }
```

Behavior:
1. Checks dirty tree — returns `-32002` if any uncommitted changes
2. If `target` differs from current branch, checks out `target`
3. Applies `shas` sequentially via `git cherry-pick`
4. After each successful commit emits a progress notification (see Progress section above)
5. On first conflict: **leaves the repo in conflict state** (does not abort), returns `-32003` with partial `applied` list and `conflicts`. The frontend drives resolution via `ConflictResolver` + `git.conflictFiles`/`git.resolveConflict`/`git.continueCherry`.

Errors:
- `-32002` `CodeDirtyTree` — uncommitted changes present
- `-32003` `CodeCherryPickConflict` — conflict; `data.applied` and `data.conflicts` carry partial results

---

### `git.abort`

Params: `{ repo: string }`

Result: `null`

Runs `git cherry-pick --abort`. Idempotent — silently succeeds even when no cherry-pick is in progress. Used by the frontend after `sidecar_cancel` kills a running cherry-pick.

---

### `git.push`

Params: `{ repo: string; branch: string; remote?: string }` (remote defaults to `"origin"`)

Result: `{ remote: string; branch: string }`

Runs `git push <remote> <branch>`. Used by "Apply & Push" mode.

---

### `git.fetch`

Params: `{ repo: string; remote?: string }` (remote defaults to `"origin"`)

Result: `{ remote: string }`

Runs `git fetch --prune <remote>`. Updates all remote-tracking refs without modifying local branches. Safe to call at any time.

---

### `git.pull`

Params: `{ repo: string; branch: string; remote?: string }` (remote defaults to `"origin"`)

Result: `{ remote: string; branch: string }`

Runs `git fetch <remote> <branch>:<branch>`. Fast-forward updates the local branch from remote **without checkout**. Fails (non-zero exit) if the update is not a fast-forward — this is intentional safe behavior.

## Adding a new method — checklist

1. Implement in `sidecar/internal/git/` (or appropriate package).
2. Register handler in `sidecar/main.go` via `s.Register("method.name", wrap1(...))`.
3. Add the method to the **implemented** table in this file.
4. Rebuild the sidecar into `app/src-tauri/binaries/sidecar-<triple>.exe` (see CLAUDE.md rule #2).
5. Add TypeScript types to `app/src/lib/rpc-types.ts`.
6. Add a typed wrapper to `app/src/lib/rpc.ts`.
7. Call from Svelte via the typed wrapper (which calls `invoke<RpcResponse>("sidecar_call", { method, params })`). No Rust change needed.

## Methods — implemented (M5)

### `git.createBranch`

Params: `{ repo: string; name: string; startPoint?: string }`

Result: `{ name: string; sha: string }`

Creates a new local branch at `startPoint` (defaults to HEAD) via `git checkout -b`. Errors if branch already exists or working tree is dirty.

---

### `git.commitDetail`

Params: `{ repo: string; sha: string }`

Result:
```ts
{
  sha: string;
  parents: string[];
  author: string;
  email: string;
  time: number;       // Unix timestamp
  subject: string;
  body: string;
}
```

Returns full commit metadata including body text. Uses `git log -1 --format=...` (not `git show --no-patch`, which is unstable on Windows).

---

### `git.commitFiles`

Params: `{ repo: string; sha: string }`

Result:
```ts
{
  files: CommitFileInfo[];
}

type CommitFileInfo = {
  path: string;
  oldPath: string;      // non-empty for renames
  status: "M" | "A" | "D" | "R" | "C";
  added: number;
  removed: number;
}
```

Returns the list of files changed in a commit with status and line counts. Uses `git diff-tree --root -M --numstat --name-status`.

---

### `git.dryRunPick`

Params: `{ repo: string; sha: string }`

Result:
```ts
{ hasConflict: boolean; conflictFiles: string[] }
```

Tests whether applying `sha` would produce conflicts using `git apply --3way --check` in a temporary index. Does not modify the working tree. Used for dry-run preview icons (⚠) in the pick queue.

---

### `git.fileDiff`

Params: `{ repo: string; sha: string; file: string }`

Result:
```ts
{ sha: string; file: string; diff: string }
```

Returns the full unified diff for a single file in a commit (`git show --unified=99999 --format= <sha> -- <file>`). Falls back to `git diff <empty-tree> <sha>` for initial commits with no parent. The large context value means the entire file is shown, not just changed hunks.

---

### `git.stagedFileDiff` (M5d)

Params: `{ repo: string; file: string }`

Result:
```ts
{ sha: string; file: string; diff: string }  // sha is always "" for staged diffs
```

Returns the unified diff for a single staged file (`git diff --cached --unified=99999 -- <file>`). Used to review the resolved content after staging a conflict resolution.

---

### `git.conflictFiles`

Params: `{ repo: string }`

Result:
```ts
{
  files: ConflictFileInfo[];
}

type ConflictFileInfo = {
  path: string;
  status: "UU" | "AA" | "DD" | "AU" | "UA" | "DU" | "UD";
}
```

Returns the list of files currently in conflict state using `git status --porcelain`. Detects all conflict types including `AA` (both sides added), which `git diff --diff-filter=U` misses.

---

### `git.resolveConflict`

Params: `{ repo: string; path: string; resolution: "ours" | "theirs" }`

Result: `null`

Resolves a conflict file by checking out one side (`git checkout --ours/--theirs <path>`) then staging it (`git add <path>`).

---

### `git.continueCherry`

Params: `{ repo: string }`

Result: `{ sha: string }`

Runs `git cherry-pick --continue --no-edit` to complete a cherry-pick after all conflicts are resolved and staged. Returns the new commit SHA.

---

### `git.fileContent`

Params: `{ repo: string; path: string }`

Result: `{ content: string }`

Reads the current working-tree content of a file (not from git history). Used by the conflict merge editor to load file content for manual editing.

---

### `git.writeAndStageFile`

Params: `{ repo: string; path: string; content: string }`

Result: `null`

Writes `content` to the file at `path` in the working tree and stages it (`git add`). Used by the conflict merge editor's Save & Stage action.

---

### `git.extractDiffFiles` (M8)

Params: `{ repo: string; sha: string; file: string }`

Result:
```ts
{ leftPath: string; rightPath: string; leftLabel: string; rightLabel: string; tmpDir: string }
```

Extracts two versions of a file to a temp directory for use by an external diff viewer. `leftPath` is the `sha^` version (before), `rightPath` is the `sha` version (after). If the file was added in this commit, `leftPath` contains an empty file; if deleted, `rightPath` is empty. `leftLabel` is `"<sha>^"`, `rightLabel` is the first 8 chars of sha. Temp dir is named `lcp-diff-*` — **not cleaned up automatically** (OS removes on reboot). Call `git.cleanupTmpDir` explicitly only if needed.

---

### `git.extractConflictFiles` (M8)

Params: `{ repo: string; file: string }`

Result:
```ts
{ basePath: string; oursPath: string; theirsPath: string; outputPath: string; tmpDir: string }
```

Extracts all three conflict stages of a file plus a working-tree copy to a temp directory (`lcp-merge-*`) for use by an external merge tool. Stages: `:1` → `basePath` (common ancestor), `:2` → `oursPath` (HEAD/target branch), `:3` → `theirsPath` (cherry-picked commit). `outputPath` is a copy of the current working-tree file (with conflict markers) — the merge tool should write its result here. Call `git.stageResolvedFile` after the tool closes, then `git.cleanupTmpDir`.

---

### `git.stageResolvedFile` (M8)

Params: `{ repo: string; file: string; contentPath: string }`

Result: `{ staged: true }`

Reads the merge result from `contentPath` (written by the external merge tool), writes it to the working tree at `file`, and stages it with `git add`. Returns an error if the file still contains conflict markers (`<<<<<<<`) — prevents staging an unresolved file when the user closes the merge tool without finishing.

---

### `git.cleanupTmpDir` (M8)

Params: `{ tmpDir: string }`

Result: `{}`

Removes a temp directory created by `git.extractDiffFiles` or `git.extractConflictFiles`. Only directories whose base name starts with `lcp-` can be removed — safety guard against deleting arbitrary paths.

---

## Concurrency model

**Current**: one process per call. Rust spawns, writes one request line, reads lines until `result`/`error`, kills child. Progress lines are forwarded as Tauri events mid-read. ~30–50 ms overhead per call.

**Concurrent calls are safe**: `ActiveSidecar` is `Mutex<HashMap<u64, CommandChild>>`. Each call gets a unique atomic `call_id`, stores its child under that ID, and removes only its own entry on completion. Multiple windows (e.g. main window + conflict merge editor) can make concurrent sidecar calls without interfering. `sidecar_cancel` drains the full map.
