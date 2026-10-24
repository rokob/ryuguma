// Package history reads each content file's revisions from git, following
// renames, so pages can show and link to their own history.
package history

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Revision struct {
	Commit  string
	Date    time.Time
	Subject string
	// Path is where the file lived in this commit.
	Path string
}

// Log holds revisions, newest first, keyed by each file's current path.
type Log struct {
	revs map[string][]Revision
}

func (l *Log) For(path string) []Revision {
	if l == nil {
		return nil
	}
	return l.revs[path]
}

// FromGit reads the history of the given files (relative to repoDir) from
// the repo's HEAD.
func FromGit(repoDir string, paths []string) (*Log, error) {
	shallow, err := git(repoDir, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(shallow)) == "true" {
		return nil, errors.New("git history is shallow; fetch the full history (e.g. fetch-depth: 0) so post history is complete")
	}
	out, err := git(repoDir, "-c", "diff.renameLimit=10000", "log",
		"--format=commit %H%x1f%aI%x1f%s", "--name-status", "-M", "HEAD")
	if err != nil {
		return nil, err
	}
	return Parse(bytes.NewReader(out), paths)
}

func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// Parse reads `git log --name-status -M` output in the FromGit format.
// Walking from newest to oldest, a rename moves tracking to the old name, so
// each current path collects revisions from all of its earlier names.
func Parse(r io.Reader, paths []string) (*Log, error) {
	log := &Log{revs: map[string][]Revision{}}
	// tracked maps a name in the commit being read to its current path.
	tracked := map[string]string{}
	for _, p := range paths {
		tracked[p] = p
	}

	var commit *Revision
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "commit "); ok {
			fields := strings.SplitN(rest, "\x1f", 3)
			if len(fields) != 3 {
				return nil, fmt.Errorf("unexpected commit line %q", line)
			}
			date, err := time.Parse(time.RFC3339, fields[1])
			if err != nil {
				return nil, err
			}
			commit = &Revision{Commit: fields[0], Date: date, Subject: fields[2]}
			continue
		}
		if commit == nil {
			return nil, errors.New("file status before any commit line")
		}
		status := strings.Split(line, "\t")
		switch {
		case len(status) == 3 && (status[0][0] == 'R' || status[0][0] == 'C'):
			from, to := status[1], status[2]
			if current, ok := tracked[to]; ok {
				log.add(current, *commit, to)
				if status[0][0] == 'R' {
					delete(tracked, to)
					tracked[from] = current
				}
			}
		case len(status) == 2:
			name := status[1]
			if current, ok := tracked[name]; ok {
				log.add(current, *commit, name)
				if status[0] == "A" {
					delete(tracked, name)
				}
			}
		}
	}
	return log, sc.Err()
}

func (l *Log) add(current string, rev Revision, path string) {
	rev.Path = path
	l.revs[current] = append(l.revs[current], rev)
}
