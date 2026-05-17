package git

import "context"

// Abort runs git cherry-pick --abort to restore the repo to the state before
// the cherry-pick. Idempotent: if no cherry-pick is in progress git exits
// non-zero, which we silently ignore.
func (r *Repo) Abort(ctx context.Context) error {
	_, _ = run(ctx, r.Path, "cherry-pick", "--abort")
	return nil
}
