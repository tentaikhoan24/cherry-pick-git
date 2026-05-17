package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lazy-cherry-pick/sidecar/internal/rpc"
)

// run executes `git <args...>` in directory dir (if non-empty) and returns
// stdout on success. On failure it returns a *Error containing stderr and the
// exit code so callers can decide whether to translate or surface it.
// logCmd emits a [GIT_CMD] line to stderr after a git command completes.
// Format: [GIT_CMD] <ms>|<branch>|git <args>
func logCmd(ctx context.Context, args []string, ms int64) {
	fmt.Fprintf(os.Stderr, "[GIT_CMD] %d|%s|git %s\n", ms, rpc.BranchFromCtx(ctx), strings.Join(args, " "))
}

func run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	full := args
	if dir != "" {
		full = append([]string{"-C", dir}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	runErr := cmd.Run()
	logCmd(ctx, args, time.Since(start).Milliseconds())
	if runErr != nil {
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return stdout.Bytes(), &Error{
			Code: CodeGitCommandFailed,
			Message: fmt.Sprintf(
				"git %s: %s",
				strings.Join(args, " "),
				strings.TrimSpace(stderr.String()),
			),
			Data: map[string]any{
				"exitCode": exitCode,
				"args":     args,
				"stderr":   stderr.String(),
			},
		}
	}
	return stdout.Bytes(), nil
}

// runWithStdin is like run but feeds stdin to the git process.
func runWithStdin(ctx context.Context, dir string, stdin []byte, args ...string) ([]byte, error) {
	full := args
	if dir != "" {
		full = append([]string{"-C", dir}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	runErr := cmd.Run()
	logCmd(ctx, args, time.Since(start).Milliseconds())
	if runErr != nil {
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return stdout.Bytes(), &Error{
			Code: CodeGitCommandFailed,
			Message: fmt.Sprintf(
				"git %s: %s",
				strings.Join(args, " "),
				strings.TrimSpace(stderr.String()),
			),
			Data: map[string]any{
				"exitCode": exitCode,
				"args":     args,
				"stderr":   stderr.String(),
			},
		}
	}
	return stdout.Bytes(), nil
}
