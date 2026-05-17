package git

import (
	"context"
	"strings"
)

type DryRunArgs struct {
	Target string   `json:"target"`
	Shas   []string `json:"shas"`
}

type DryRunItem struct {
	Sha          string   `json:"sha"`
	WillConflict bool     `json:"willConflict"`
	Files        []string `json:"files"` // conflicting file paths, empty when clean
}

type DryRunResult struct {
	Results []DryRunItem `json:"results"`
}

// DryRunPick checks whether each sha in args.Shas would conflict if applied
// onto target (defaulting to HEAD). It uses "git format-patch | git apply
// --3way --check" which is read-only — the working tree is never modified.
//
// Limitations: each sha is checked independently against the current HEAD,
// not against the cumulative result of prior picks. This means a chain of
// non-conflicting commits may still be flagged if their individual diffs
// overlap. Good enough for a pre-flight warning UI.
func (r *Repo) DryRunPick(ctx context.Context, args DryRunArgs) (*DryRunResult, error) {
	result := &DryRunResult{Results: make([]DryRunItem, 0, len(args.Shas))}

	for _, sha := range args.Shas {
		item := DryRunItem{Sha: sha}

		// Generate the patch for this commit and pipe it to git apply --check.
		// We run them as separate commands: format-patch to a temp var, then apply.
		// "git apply --check" exits 0 on clean, non-zero on conflict/error.
		patch, err := run(ctx, r.Path, "format-patch", "-1", "--stdout", sha)
		if err != nil {
			// If we can't even format the patch, skip this sha.
			item.WillConflict = false
			result.Results = append(result.Results, item)
			continue
		}

		conflictFiles := applyCheck(ctx, r.Path, patch)
		if conflictFiles != nil {
			item.WillConflict = true
			item.Files = conflictFiles
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

// applyCheck pipes the patch bytes into "git apply --3way --check" and returns
// the list of conflicting file paths on failure, or nil on success.
func applyCheck(ctx context.Context, repoPath string, patch []byte) []string {
	// run() only supports git sub-commands with string args, so we use runWithStdin.
	out, err := runWithStdin(ctx, repoPath, patch, "apply", "--3way", "--check", "-")
	if err == nil {
		return nil // clean apply
	}
	// Parse stderr (stored in err.Data["stderr"]) for "error: ... already exists"
	// or "error: patch failed:" lines to extract file names.
	var files []string
	stderr := ""
	if e, ok := err.(*Error); ok {
		stderr, _ = e.Data["stderr"].(string)
	} else {
		stderr = string(out)
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		// "error: patch failed: <file>:<line>"
		if strings.HasPrefix(line, "error: patch failed: ") {
			part := strings.TrimPrefix(line, "error: patch failed: ")
			// strip ":linenum" suffix
			if idx := strings.LastIndex(part, ":"); idx > 0 {
				part = part[:idx]
			}
			files = appendUnique(files, part)
		}
	}
	if len(files) == 0 {
		files = []string{} // conflict detected but file list unknown
	}
	return files
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
