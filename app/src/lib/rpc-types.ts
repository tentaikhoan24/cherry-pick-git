// Wire types shared between Go sidecar and Svelte frontend.
// Keep in sync with sidecar/internal/git/types.go.

export interface FileStatus {
  path: string;
  status: string; // "M" | "A" | "D" | "R" | "C" | "U"
}

export interface RepoStatus {
  branch: string;
  upstream: string;
  ahead: number;
  behind: number;
  dirty: boolean;
  staged: FileStatus[];
  unstaged: FileStatus[];
  untracked: string[];
  detached: boolean;
}

export interface Branch {
  name: string;
  sha: string;
  isHead: boolean;
  upstream: string;
}

export interface CommitFilter {
  author?: string;
  messageContains?: string;
  since?: string;
  until?: string;
  pathGlob?: string;
}

export interface Commit {
  sha: string;
  parents: string[];
  author: string;
  email: string;
  time: number; // Unix timestamp (seconds)
  subject: string;
  refs: string[];
}

export interface ConflictInfo {
  sha: string;
  files: string[];
}

export interface CherryPickResult {
  applied: string[];
  conflicts: ConflictInfo[];
}

export interface CherryPickProgress {
  n: number;
  total: number;
  sha: string;
}

export interface RecentRepo {
  path: string;
  lastOpened: number; // unix timestamp (seconds)
}

export interface PushResult {
  remote: string;
  branch: string;
}

export interface CreateBranchResult {
  name: string;
  sha: string;
}

export interface FetchResult {
  remote: string;
}

export interface PullResult {
  remote: string;
  branch: string;
}

export interface CommitDetail {
  sha: string;
  parents: string[];
  author: string;
  email: string;
  time: number;
  subject: string;
  body: string;
}

export interface CommitFile {
  path: string;
  added: number;
  removed: number;
  status: string; // M, A, D, R, C, T, U
}

export interface DryRunItem {
  sha: string;
  willConflict: boolean;
  files: string[];
}

export interface DryRunResult {
  results: DryRunItem[];
}

export interface FileDiffResult {
  sha: string;
  file: string;
  diff: string; // raw unified diff text
}

export interface OpenRepoResult {
  path: string;
  branch: string;
  detached: boolean;
  cherryPickHead?: string;
}

export interface ConflictFileInfo {
  path: string;
  status: string; // UU, AA, DD, AU, UA, DU, UD
}

export interface ConflictFilesResult {
  files: ConflictFileInfo[];
}

export interface ContinueCherryResult {
  done: boolean;
}

export interface FileContentResult {
  content: string;
}

export interface WriteAndStageResult {
  staged: boolean;
}

export interface AppSettings {
  maxCommits: number;
  defaultApplyMode: "apply" | "apply-push";
  showEolMarkers: boolean;
  autoFetchOnOpen: boolean;
  theme: "dark" | "light";
}

export interface RpcError {
  code: number;
  message: string;
  data?: Record<string, unknown>;
}

export interface RpcResponse<T = unknown> {
  jsonrpc: string;
  id?: unknown;
  result?: T;
  error?: RpcError;
}
