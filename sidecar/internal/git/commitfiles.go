package git

import (
	"context"
	"strconv"
	"strings"
)

type CommitFilesArgs struct {
	Sha string `json:"sha"`
}

type CommitFile struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Status  string `json:"status"` // M, A, D, R, C, T, U
}

// CommitFiles returns the list of files changed by a commit with +/- line counts.
// Handles initial commits (no parent) via --root flag.
func (r *Repo) CommitFiles(ctx context.Context, args CommitFilesArgs) ([]CommitFile, error) {
	baseArgs := []string{"diff-tree", "--no-commit-id", "-r", "--root"}

	// --numstat: "<added>\t<removed>\t<path>"  (binary files show "-\t-\t<path>")
	numOut, err := run(ctx, r.Path, append(baseArgs, "--numstat", args.Sha)...)
	if err != nil {
		return nil, err
	}
	// --name-status: "<status>\t<path>"  or  "<status>\t<old>\t<new>" for renames
	nsOut, err := run(ctx, r.Path, append(baseArgs, "--name-status", args.Sha)...)
	if err != nil {
		return nil, err
	}

	// Build status map: path → status letter
	statusMap := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(nsOut)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		statusLetter := string([]rune(parts[0])[0:1]) // "R100" → "R"
		if statusLetter == "R" || statusLetter == "C" {
			// Rename/copy: name-status has 3 fields, numstat path shows "old => new"
			// Map both old and new path to the status
			if len(parts) == 3 {
				statusMap[parts[1]] = statusLetter
				statusMap[parts[2]] = statusLetter
			}
		} else {
			statusMap[parts[1]] = statusLetter
		}
	}

	var files []CommitFile
	for _, line := range strings.Split(strings.TrimSpace(string(numOut)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		added, _ := strconv.Atoi(parts[0])
		removed, _ := strconv.Atoi(parts[1])
		path := parts[2]

		// Look up status; for rename paths numstat shows "old => new", try both
		status := statusMap[path]
		if status == "" {
			// Try extracting the new path from "old => new" rename format
			if idx := strings.Index(path, " => "); idx >= 0 {
				newPath := path[idx+4:]
				status = statusMap[newPath]
			}
		}
		if status == "" {
			status = "M"
		}
		files = append(files, CommitFile{
			Path:    path,
			Added:   added,
			Removed: removed,
			Status:  status,
		})
	}
	if files == nil {
		files = []CommitFile{}
	}
	return files, nil
}
