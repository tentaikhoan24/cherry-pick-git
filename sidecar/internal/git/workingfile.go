package git

import (
	"context"
	"os"
	"path/filepath"
)

type FileContentArgs struct {
	File string `json:"file"`
}

type FileContentResult struct {
	Content string `json:"content"`
}

// FileContent reads a file from the working tree and returns its raw content.
func (r *Repo) FileContent(_ context.Context, args FileContentArgs) (*FileContentResult, error) {
	data, err := os.ReadFile(filepath.Join(r.Path, filepath.FromSlash(args.File)))
	if err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot read file: " + err.Error()}
	}
	return &FileContentResult{Content: string(data)}, nil
}

// ── write merged result ───────────────────────────────────────────────────────

type WriteAndStageArgs struct {
	File    string `json:"file"`
	Content string `json:"content"`
}

// WriteAndStageFile writes content to a working-tree file and stages it.
// Used by the conflict merge editor after the user resolves conflicts.
func (r *Repo) WriteAndStageFile(ctx context.Context, args WriteAndStageArgs) (any, error) {
	path := filepath.Join(r.Path, filepath.FromSlash(args.File))
	if err := os.WriteFile(path, []byte(args.Content), 0644); err != nil {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "cannot write file: " + err.Error()}
	}
	if _, err := run(ctx, r.Path, "add", "--", args.File); err != nil {
		return nil, err
	}
	return map[string]any{"staged": true}, nil
}
