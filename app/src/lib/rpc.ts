import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import type {
  RpcResponse,
  RpcError,
  OpenRepoResult,
  RepoStatus,
  Branch,
  Commit,
  CommitFilter,
  CherryPickResult,
  CherryPickProgress,
  CreateBranchResult,
  FetchResult,
  PullResult,
  PushResult,
  RecentRepo,
  CommitDetail,
  CommitFile,
  DryRunResult,
  FileDiffResult,
  ConflictFilesResult,
  ContinueCherryResult,
  FileContentResult,
  WriteAndStageResult,
  AppSettings,
} from "./rpc-types";

export class RpcCallError extends Error {
  constructor(public readonly rpcError: RpcError) {
    super(`[${rpcError.code}] ${rpcError.message}`);
  }
}

async function call<T>(method: string, params?: Record<string, unknown>): Promise<T> {
  const res = await invoke<RpcResponse<T>>("sidecar_call", { method, params: params ?? null });
  if (res.error) throw new RpcCallError(res.error);
  return res.result as T;
}

export const rpc = {
  ping: () => call<string>("ping"),

  version: () => call<{ sidecar: string; go: string; git: string }>("version"),

  cancel: () => invoke<void>("sidecar_cancel"),

  recents: {
    load: () => invoke<RecentRepo[]>("recents_load"),
    save: (items: RecentRepo[]) => invoke<void>("recents_save", { recents: items }),
  },

  settings: {
    load: () => invoke<AppSettings>("settings_load"),
    save: (settings: AppSettings) => invoke<void>("settings_save", { settings }),
  },

  git: {
    openRepo: (repo: string) =>
      call<OpenRepoResult>("git.openRepo", { repo }),

    status: (repo: string) =>
      call<RepoStatus>("git.status", { repo }),

    branches: (repo: string, includeRemote = false) =>
      call<Branch[]>("git.branches", { repo, includeRemote }),

    commits: (repo: string, ref = "HEAD", limit = 100, skip = 0, filter?: CommitFilter) =>
      call<Commit[]>("git.commits", { repo, ref, limit, skip, filter }),

    cherryPick: async (
      repo: string,
      target: string,
      shas: string[],
      strategy?: "smart" | "theirs" | "ours",
      onprogress?: (p: CherryPickProgress) => void,
    ) => {
      const unlisten = onprogress
        ? await listen<CherryPickProgress>("cp-progress", (e) => onprogress(e.payload))
        : null;
      try {
        return await call<CherryPickResult>("git.cherryPick", { repo, target, shas, strategy });
      } finally {
        unlisten?.();
      }
    },

    abort: (repo: string) => call<void>("git.abort", { repo }),

    createBranch: (repo: string, name: string, base?: string) =>
      call<CreateBranchResult>("git.createBranch", { repo, name, base }),

    fetch: (repo: string, remote = "origin") =>
      call<FetchResult>("git.fetch", { repo, remote }),

    pull: (repo: string, branch: string, remote = "origin") =>
      call<PullResult>("git.pull", { repo, branch, remote }),

    push: (repo: string, branch: string, remote = "origin") =>
      call<PushResult>("git.push", { repo, branch, remote }),

    commitDetail: (repo: string, sha: string) =>
      call<CommitDetail>("git.commitDetail", { repo, sha }),

    commitFiles: (repo: string, sha: string) =>
      call<CommitFile[]>("git.commitFiles", { repo, sha }),

    dryRunPick: (repo: string, target: string, shas: string[]) =>
      call<DryRunResult>("git.dryRunPick", { repo, target, shas }),

    fileDiff: (repo: string, sha: string, file: string) =>
      call<FileDiffResult>("git.fileDiff", { repo, sha, file }),
    stagedFileDiff: (repo: string, file: string) =>
      call<FileDiffResult>("git.stagedFileDiff", { repo, file }),

    conflictFiles: (repo: string) =>
      call<ConflictFilesResult>("git.conflictFiles", { repo }),

    resolveConflict: (repo: string, file: string, strategy: "ours" | "theirs") =>
      call<{ resolved: boolean }>("git.resolveConflict", { repo, file, strategy }),

    continueCherry: (repo: string) =>
      call<ContinueCherryResult>("git.continueCherry", { repo }),

    fileContent: (repo: string, file: string) =>
      call<FileContentResult>("git.fileContent", { repo, file }),

    writeAndStageFile: (repo: string, file: string, content: string) =>
      call<WriteAndStageResult>("git.writeAndStageFile", { repo, file, content }),
  },
};
