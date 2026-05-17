package git

import "context"

type FileDiffArgs struct {
	Sha  string `json:"sha"`
	File string `json:"file"`
}

type FileDiffResult struct {
	Sha  string `json:"sha"`
	File string `json:"file"`
	Diff string `json:"diff"` // raw unified diff text
}

// FileDiff returns the unified diff for a single file in a given commit.
// For initial commits (no parent), compares against an empty tree.
func (r *Repo) FileDiff(ctx context.Context, args FileDiffArgs) (*FileDiffResult, error) {
	// Try diff against parent first (normal commits).
	out, err := run(ctx, r.Path, "show", "--unified=99999", "--format=", args.Sha, "--", args.File)
	if err != nil {
		// Fallback for initial commit: diff against empty tree.
		emptyTree := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
		out, err = run(ctx, r.Path, "diff", "--unified=99999", emptyTree, args.Sha, "--", args.File)
		if err != nil {
			return nil, err
		}
	}
	return &FileDiffResult{
		Sha:  args.Sha,
		File: args.File,
		Diff: string(out),
	}, nil
}
