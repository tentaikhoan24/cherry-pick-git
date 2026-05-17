package git

import "context"

type FetchArgs struct {
	Remote string `json:"remote"` // defaults to "origin"
}

type FetchResult struct {
	Remote string `json:"remote"`
}

// Fetch runs git fetch --prune <remote>, updating all remote-tracking refs
// without modifying any local branch.
func (r *Repo) Fetch(ctx context.Context, args FetchArgs) (*FetchResult, error) {
	remote := args.Remote
	if remote == "" {
		remote = "origin"
	}
	if _, err := run(ctx, r.Path, "fetch", "--prune", remote); err != nil {
		return nil, err
	}
	return &FetchResult{Remote: remote}, nil
}

type PullArgs struct {
	Branch string `json:"branch"` // local branch to fast-forward
	Remote string `json:"remote"` // defaults to "origin"
}

type PullResult struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

// Pull fast-forwards a local branch from its remote counterpart without
// requiring a checkout. Uses "git fetch <remote> <branch>:<branch>" which
// only succeeds when the update is a fast-forward; non-fast-forward refs are
// rejected safely.
func (r *Repo) Pull(ctx context.Context, args PullArgs) (*PullResult, error) {
	remote := args.Remote
	if remote == "" {
		remote = "origin"
	}
	refspec := args.Branch + ":" + args.Branch
	if _, err := run(ctx, r.Path, "fetch", remote, refspec); err != nil {
		return nil, err
	}
	return &PullResult{Remote: remote, Branch: args.Branch}, nil
}
