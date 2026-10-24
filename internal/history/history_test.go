package history_test

import (
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/history"
)

// Newest first, as git log prints it. Fields are separated by \x1f.
const gitLog = "commit c4\x1f2026-10-24T23:00:00-07:00\x1fchore(content): migrate frontmatter\n" +
	"\n" +
	"M\tcontent/books/dune.md\n" +
	"M\tcontent/books/ubik.md\n" +
	"commit c3\x1f2026-10-24T22:00:00-07:00\x1frefactor(content): move posts into typed directories\n" +
	"\n" +
	"R100\tcontent/2015-08-03-dune.md\tcontent/books/dune.md\n" +
	"R100\tcontent/2015-12-22-ubik.md\tcontent/books/ubik.md\n" +
	"commit c2\x1f2018-12-29T10:00:00Z\x1fMigrate to new site using gatsby\n" +
	"\n" +
	"R097\t_posts/2015-08-03-dune.md\tcontent/2015-08-03-dune.md\n" +
	"D\t_posts/gone.md\n" +
	"A\tcontent/2015-12-22-ubik.md\n" +
	"commit c1b\x1f2015-08-04T09:00:00Z\x1ffix typo in dune: spice\n" +
	"\n" +
	"M\t_posts/2015-08-03-dune.md\n" +
	"commit c1\x1f2015-08-03T20:00:00Z\x1fpost about dune\n" +
	"\n" +
	"A\t_posts/2015-08-03-dune.md\n" +
	"A\t_posts/gone.md\n"

func TestParseFollowsRenames(t *testing.T) {
	log, err := history.Parse(strings.NewReader(gitLog), []string{"content/books/dune.md", "content/books/ubik.md", "content/books/new.md"})
	if err != nil {
		t.Fatal(err)
	}
	dune := log.For("content/books/dune.md")
	var got []string
	for _, r := range dune {
		got = append(got, r.Commit+"@"+r.Path)
	}
	want := "c4@content/books/dune.md c3@content/books/dune.md c2@content/2015-08-03-dune.md c1b@_posts/2015-08-03-dune.md c1@_posts/2015-08-03-dune.md"
	if strings.Join(got, " ") != want {
		t.Errorf("dune history =\n%s\nwant\n%s", strings.Join(got, " "), want)
	}
	if dune[3].Subject != "fix typo in dune: spice" || dune[3].Date.Format("2006-01-02") != "2015-08-04" {
		t.Errorf("revision = %+v", dune[3])
	}
	// Ubik's history starts where it was added, not before.
	if n := len(log.For("content/books/ubik.md")); n != 3 {
		t.Errorf("ubik has %d revisions, want 3", n)
	}
	// A file with no commits yet has no history.
	if n := len(log.For("content/books/new.md")); n != 0 {
		t.Errorf("uncommitted file has %d revisions", n)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := history.Parse(strings.NewReader("M\tcontent/x.md\n"), nil); err == nil {
		t.Error("expected error for status line before any commit")
	}
}
