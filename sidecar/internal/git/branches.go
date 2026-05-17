package git

import (
	"context"
	"strings"
)

// Branches lists local branches (and remote branches if includeRemote is true)
// using `git for-each-ref`. The HEAD branch is marked with IsHead=true.
func (r *Repo) Branches(ctx context.Context, includeRemote bool) ([]Branch, error) {
	args := []string{
		"for-each-ref",
		"--format=%(refname)%00%(objectname)%00%(HEAD)%00%(upstream:short)",
		"refs/heads",
	}
	if includeRemote {
		args = append(args, "refs/remotes")
	}
	out, err := run(ctx, r.Path, args...)
	if err != nil {
		return nil, err
	}

	branches := []Branch{}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 3 {
			continue
		}
		ref := parts[0]
		name := ref
		remote := false
		switch {
		case strings.HasPrefix(ref, "refs/heads/"):
			name = strings.TrimPrefix(ref, "refs/heads/")
		case strings.HasPrefix(ref, "refs/remotes/"):
			name = strings.TrimPrefix(ref, "refs/remotes/")
			remote = true
		}
		// Skip pseudo-refs like refs/remotes/origin/HEAD that just point at another branch.
		if remote && strings.HasSuffix(name, "/HEAD") {
			continue
		}
		b := Branch{
			Name:   name,
			Sha:    parts[1],
			IsHead: strings.TrimSpace(parts[2]) == "*",
			Remote: remote,
		}
		if len(parts) >= 4 {
			b.Upstream = parts[3]
		}
		branches = append(branches, b)
	}
	return branches, nil
}
