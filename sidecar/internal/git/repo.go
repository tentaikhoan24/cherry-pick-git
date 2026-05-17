package git

import (
	"context"
	"path/filepath"
	"strings"
)

// Repo is a validated git working tree, rooted at its top-level path.
type Repo struct {
	Path string
}

// Open validates that path is inside a git working tree and returns a Repo
// rooted at its top level. Returns *Error with CodeRepoNotFound on failure.
func Open(ctx context.Context, path string) (*Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, &Error{Code: CodeRepoNotFound, Message: err.Error()}
	}
	out, err := run(ctx, abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, &Error{Code: CodeRepoNotFound, Message: "not a git repository: " + abs}
	}
	return &Repo{Path: strings.TrimSpace(string(out))}, nil
}

// CurrentBranch returns the current branch name. The detached flag is true
// when HEAD does not point to a branch (e.g. checkout of a tag or sha).
func (r *Repo) CurrentBranch(ctx context.Context) (name string, detached bool, err error) {
	out, runErr := run(ctx, r.Path, "symbolic-ref", "--quiet", "--short", "HEAD")
	if runErr != nil {
		// symbolic-ref exits non-zero with no output on detached HEAD.
		// Distinguish detached from real failure by checking stderr in Data.
		if e, ok := runErr.(*Error); ok {
			if stderr, _ := e.Data["stderr"].(string); strings.TrimSpace(stderr) == "" {
				return "", true, nil
			}
		}
		return "", false, runErr
	}
	return strings.TrimSpace(string(out)), false, nil
}

// IsDirty returns true if there are any tracked or untracked changes.
func (r *Repo) IsDirty(ctx context.Context) (bool, error) {
	out, err := run(ctx, r.Path, "status", "--porcelain=v1")
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}
