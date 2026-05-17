package git

import "context"

type PushArgs struct {
	Branch string `json:"branch"`
	Remote string `json:"remote"` // defaults to "origin"
}

type PushResult struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
}

func (r *Repo) Push(ctx context.Context, args PushArgs) (*PushResult, error) {
	remote := args.Remote
	if remote == "" {
		remote = "origin"
	}
	if _, err := run(ctx, r.Path, "push", remote, args.Branch); err != nil {
		return nil, err
	}
	return &PushResult{Remote: remote, Branch: args.Branch}, nil
}
