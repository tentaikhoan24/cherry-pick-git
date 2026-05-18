package git

import "context"

type StagedFileDiffArgs struct {
	File string `json:"file"`
}

// StagedFileDiff returns the unified diff for a staged file (git diff --cached).
// Used to review the resolved content after staging a conflict resolution.
func (r *Repo) StagedFileDiff(ctx context.Context, args StagedFileDiffArgs) (*FileDiffResult, error) {
	out, err := run(ctx, r.Path, "diff", "--cached", "--unified=99999", "--", args.File)
	if err != nil {
		return nil, err
	}
	return &FileDiffResult{
		Sha:  "",
		File: args.File,
		Diff: string(out),
	}, nil
}
