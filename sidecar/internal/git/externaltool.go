package git

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
)

// ── ExtractDiffFiles ──────────────────────────────────────────────────────────

type ExtractDiffFilesArgs struct {
	Sha  string `json:"sha"`
	File string `json:"file"`
}

type ExtractDiffFilesResult struct {
	LeftPath   string `json:"leftPath"`
	RightPath  string `json:"rightPath"`
	LeftLabel  string `json:"leftLabel"`
	RightLabel string `json:"rightLabel"`
	TmpDir     string `json:"tmpDir"`
}

// ExtractDiffFiles extracts the before (sha^) and after (sha) versions of a
// file to a temp directory and returns their paths for use by an external diff
// viewer. The caller is responsible for calling CleanupTmpDir when done.
func (r *Repo) ExtractDiffFiles(ctx context.Context, args ExtractDiffFilesArgs) (*ExtractDiffFilesResult, error) {
	tmp, err := os.MkdirTemp("", "lcp-diff-*")
	if err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot create temp dir: " + err.Error()}
	}

	base := filepath.Base(args.File)

	// Right side: sha version (may be empty if file was deleted in this commit).
	rightData, _ := run(ctx, r.Path, "show", args.Sha+":"+args.File)
	rightPath := filepath.Join(tmp, "right_"+base)
	if werr := os.WriteFile(rightPath, rightData, 0644); werr != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot write right file: " + werr.Error()}
	}

	// Left side: sha^ version (empty for added files or initial commits).
	leftData, _ := run(ctx, r.Path, "show", args.Sha+"^:"+args.File)
	leftPath := filepath.Join(tmp, "left_"+base)
	if werr := os.WriteFile(leftPath, leftData, 0644); werr != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot write left file: " + werr.Error()}
	}

	shortSha := args.Sha
	if len(shortSha) > 8 {
		shortSha = shortSha[:8]
	}

	return &ExtractDiffFilesResult{
		LeftPath:   leftPath,
		RightPath:  rightPath,
		LeftLabel:  args.Sha + "^",
		RightLabel: shortSha,
		TmpDir:     tmp,
	}, nil
}

// ── ExtractConflictFiles ──────────────────────────────────────────────────────

type ExtractConflictFilesArgs struct {
	File string `json:"file"`
}

type ExtractConflictFilesResult struct {
	BasePath   string `json:"basePath"`
	OursPath   string `json:"oursPath"`
	TheirsPath string `json:"theirsPath"`
	OutputPath string `json:"outputPath"`
	TmpDir     string `json:"tmpDir"`
}

// ExtractConflictFiles extracts the three conflict stages of a file (:1 base,
// :2 ours, :3 theirs) plus a copy of the working-tree file as the output
// target. The merge tool should write its result to outputPath; call
// StageResolvedFile afterwards to write it back and stage it.
func (r *Repo) ExtractConflictFiles(ctx context.Context, args ExtractConflictFilesArgs) (*ExtractConflictFilesResult, error) {
	tmp, err := os.MkdirTemp("", "lcp-merge-*")
	if err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot create temp dir: " + err.Error()}
	}

	base := filepath.Base(args.File)

	writeStage := func(name, stage string) (string, error) {
		// stage is ":1", ":2", or ":3" — git index stage prefixes.
		data, _ := run(ctx, r.Path, "show", stage+":"+args.File)
		path := filepath.Join(tmp, name+"_"+base)
		if werr := os.WriteFile(path, data, 0644); werr != nil {
			return "", &Error{Code: CodeGitCommandFailed, Message: "cannot write " + name + ": " + werr.Error()}
		}
		return path, nil
	}

	basePath, err := writeStage("base", ":1")
	if err != nil {
		return nil, err
	}
	oursPath, err := writeStage("ours", ":2")
	if err != nil {
		return nil, err
	}
	theirsPath, err := writeStage("theirs", ":3")
	if err != nil {
		return nil, err
	}

	// Output: copy of the working-tree file (with conflict markers). The merge
	// tool edits this copy; StageResolvedFile writes it back to the repo.
	workingData, err := os.ReadFile(filepath.Join(r.Path, filepath.FromSlash(args.File)))
	if err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot read working file: " + err.Error()}
	}
	outputPath := filepath.Join(tmp, "output_"+base)
	if werr := os.WriteFile(outputPath, workingData, 0644); werr != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot write output file: " + werr.Error()}
	}

	return &ExtractConflictFilesResult{
		BasePath:   basePath,
		OursPath:   oursPath,
		TheirsPath: theirsPath,
		OutputPath: outputPath,
		TmpDir:     tmp,
	}, nil
}

// ── StageResolvedFile ─────────────────────────────────────────────────────────

type StageResolvedFileArgs struct {
	File        string `json:"file"`
	ContentPath string `json:"contentPath"`
}

// StageResolvedFile reads the merge result from contentPath (written by the
// external merge tool), writes it to the working tree, and stages it with
// git add.
func (r *Repo) StageResolvedFile(ctx context.Context, args StageResolvedFileArgs) (any, error) {
	data, err := os.ReadFile(args.ContentPath)
	if err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot read output file: " + err.Error()}
	}
	if bytes.Contains(data, []byte("<<<<<<<")) {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "file still contains conflict markers — resolve all conflicts before staging"}
	}
	dest := filepath.Join(r.Path, filepath.FromSlash(args.File))
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot write file: " + err.Error()}
	}
	if _, err := run(ctx, r.Path, "add", "--", args.File); err != nil {
		return nil, err
	}
	return map[string]any{"staged": true}, nil
}

// ── CleanupTmpDir ─────────────────────────────────────────────────────────────

type CleanupTmpDirArgs struct {
	TmpDir string `json:"tmpDir"`
}

// CleanupTmpDir removes a temp directory created by ExtractDiffFiles or
// ExtractConflictFiles. Only directories whose base name starts with "lcp-"
// can be removed as a safety guard against arbitrary path deletion.
func CleanupTmpDir(_ context.Context, args CleanupTmpDirArgs) (any, error) {
	if !strings.HasPrefix(filepath.Base(args.TmpDir), "lcp-") {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "tmpDir must be an lcp-* directory"}
	}
	if err := os.RemoveAll(args.TmpDir); err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cleanup failed: " + err.Error()}
	}
	return map[string]any{}, nil
}
