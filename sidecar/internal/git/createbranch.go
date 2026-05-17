package git

import (
	"context"
	"strings"
)

type CreateBranchArgs struct {
	Name string `json:"name"`
	Base string `json:"base"` // optional — defaults to HEAD
}

type CreateBranchResult struct {
	Name string `json:"name"`
	Sha  string `json:"sha"`
}

// CreateBranch runs git branch <name> [base]. The new branch is not checked
// out. Returns the SHA the branch points to.
func (r *Repo) CreateBranch(ctx context.Context, args CreateBranchArgs) (*CreateBranchResult, error) {
	gitArgs := []string{"branch", args.Name}
	if args.Base != "" {
		gitArgs = append(gitArgs, args.Base)
	}
	if _, err := run(ctx, r.Path, gitArgs...); err != nil {
		return nil, err
	}
	out, err := run(ctx, r.Path, "rev-parse", args.Name)
	if err != nil {
		return nil, err
	}
	return &CreateBranchResult{Name: args.Name, Sha: strings.TrimSpace(string(out))}, nil
}
