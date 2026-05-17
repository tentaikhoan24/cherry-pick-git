package git

import "fmt"

// Error is a typed error from the git layer that maps cleanly onto a
// JSON-RPC application error. Code follows the convention in docs/IPC.md.
type Error struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("git error %d: %s", e.Code, e.Message)
}

const (
	CodeGitCommandFailed   = -32001
	CodeDirtyTree          = -32002
	CodeCherryPickConflict = -32003
	CodeRepoNotFound       = -32004
)
