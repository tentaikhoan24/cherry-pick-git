package git

import (
	"bufio"
	"context"
	"strconv"
	"strings"
)

// Status returns the working tree status using `git status --porcelain=v2 --branch`.
// Filenames containing literal newlines are not supported (extreme edge case).
func (r *Repo) Status(ctx context.Context) (*Status, error) {
	out, err := run(ctx, r.Path, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return nil, err
	}

	st := &Status{
		Staged:    []FileStatus{},
		Unstaged:  []FileStatus{},
		Untracked: []string{},
	}

	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			head := strings.TrimPrefix(line, "# branch.head ")
			if head == "(detached)" {
				st.Detached = true
			} else {
				st.Branch = head
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			st.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			parts := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(parts) >= 2 {
				a, _ := strconv.Atoi(strings.TrimPrefix(parts[0], "+"))
				b, _ := strconv.Atoi(strings.TrimPrefix(parts[1], "-"))
				st.Ahead = a
				st.Behind = b
			}
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			parseChangedEntry(line, st)
		case strings.HasPrefix(line, "u "):
			parseUnmergedEntry(line, st)
		case strings.HasPrefix(line, "? "):
			st.Untracked = append(st.Untracked, strings.TrimPrefix(line, "? "))
		}
	}

	st.Dirty = len(st.Staged) > 0 || len(st.Unstaged) > 0 || len(st.Untracked) > 0
	return st, nil
}

// parseChangedEntry handles "1 XY ..." (ordinary) and "2 XY ..." (renamed/copied).
// Format reference: https://git-scm.com/docs/git-status#_porcelain_format_version_2
//
//	"1 XY sub mH mI mW hH hI path"
//	"2 XY sub mH mI mW hH hI Xscore path<TAB>origPath"
func parseChangedEntry(line string, st *Status) {
	fields := strings.SplitN(line, " ", 9)
	if len(fields) < 9 {
		return
	}
	xy := fields[1]
	rest := fields[8]
	var path, oldPath string
	if line[0] == '2' {
		sub := strings.SplitN(rest, " ", 2)
		if len(sub) == 2 {
			paths := strings.SplitN(sub[1], "\t", 2)
			path = paths[0]
			if len(paths) == 2 {
				oldPath = paths[1]
			}
		}
	} else {
		path = rest
	}
	fs := FileStatus{Path: path, OldPath: oldPath, XY: xy}
	if len(xy) >= 2 {
		if xy[0] != '.' {
			st.Staged = append(st.Staged, fs)
		}
		if xy[1] != '.' {
			st.Unstaged = append(st.Unstaged, fs)
		}
	}
}

// parseUnmergedEntry handles "u XY ..." (unmerged / conflict).
//
//	"u XY sub m1 m2 m3 mW h1 h2 h3 path"
func parseUnmergedEntry(line string, st *Status) {
	fields := strings.SplitN(line, " ", 11)
	if len(fields) < 11 {
		return
	}
	st.Unstaged = append(st.Unstaged, FileStatus{Path: fields[10], XY: fields[1]})
}
