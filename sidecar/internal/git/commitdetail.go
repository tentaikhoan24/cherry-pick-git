package git

import (
	"context"
	"strconv"
	"strings"
)

type CommitDetailArgs struct {
	Sha string `json:"sha"`
}

type CommitDetail struct {
	Sha     string   `json:"sha"`
	Parents []string `json:"parents"`
	Author  string   `json:"author"`
	Email   string   `json:"email"`
	Time    int64    `json:"time"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

// CommitDetail fetches full commit metadata including the message body.
// Uses "git log -1" (same pattern as commits.go) instead of "git show --no-patch"
// because git show adds extra output on some platforms.
func (r *Repo) CommitDetail(ctx context.Context, args CommitDetailArgs) (*CommitDetail, error) {
	// Fields separated by NUL; body may contain newlines so it comes last.
	const format = "%H%x00%P%x00%an%x00%ae%x00%at%x00%s%x00%b"
	out, err := run(ctx, r.Path, "log", "-1", "--format=format:"+format, args.Sha)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(out))
	// Split on NUL — body is the last field and may be empty
	parts := strings.SplitN(text, "\x00", 7)
	if len(parts) < 6 {
		return nil, &Error{Code: CodeGitCommandFailed, Message: "unexpected git log output for " + args.Sha}
	}
	ts, _ := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
	parents := []string{}
	if p := strings.TrimSpace(parts[1]); p != "" {
		parents = strings.Fields(p)
	}
	d := &CommitDetail{
		Sha:     strings.TrimSpace(parts[0]),
		Parents: parents,
		Author:  parts[2],
		Email:   parts[3],
		Time:    ts,
		Subject: parts[5],
	}
	if len(parts) == 7 {
		d.Body = strings.TrimSpace(parts[6])
	}
	return d, nil
}
