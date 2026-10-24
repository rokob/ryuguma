package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/openlibrary"
)

var testNow = time.Date(2026, 10, 25, 9, 30, 0, 0, time.UTC)

type fakeSearch struct {
	docs map[string][]openlibrary.Doc
	err  error
}

func (f fakeSearch) Search(_ context.Context, title string) ([]openlibrary.Doc, error) {
	return f.docs[title], f.err
}

type harness struct {
	t      *testing.T
	root   string
	out    bytes.Buffer
	search fakeSearch
}

// newHarness makes a site repo in a temp dir with the given content files.
func newHarness(t *testing.T, files map[string]string) *harness {
	t.Helper()
	h := &harness{t: t, root: t.TempDir()}
	write := func(rel, data string) {
		p := filepath.Join(h.root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(configFile, "title: test\n")
	for rel, data := range files {
		write(filepath.Join("content", rel), data)
	}
	return h
}

func (h *harness) run(input string, args ...string) error {
	h.t.Helper()
	h.out.Reset()
	e := &env{
		project: &project{root: h.root},
		now:     func() time.Time { return testNow },
		in:      bufio.NewReader(strings.NewReader(input)),
		out:     &h.out,
		search:  h.search,
	}
	return run(args, e)
}

func (h *harness) entry(rel string) content.Entry {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.root, "content", rel))
	if err != nil {
		h.t.Fatal(err)
	}
	e, err := content.Parse(rel, data)
	if err != nil {
		h.t.Fatal(err)
	}
	return e
}

func TestNewPost(t *testing.T) {
	h := newHarness(t, map[string]string{"posts/old.md": "---\ntitle: Old\ndate: 2014-01-01\ntags: [life, code]\n---\n"})
	if err := h.run("life  ideas\n", "new", "post", "What's in a Type?"); err != nil {
		t.Fatal(err)
	}
	e := h.entry("posts/whats-in-a-type.md")
	if e.Title != "What's in a Type?" || e.Date.String() != "2026-10-25" || !e.Draft {
		t.Errorf("post = %+v", e)
	}
	if !slices.Equal(e.Tags, []string{"life", "ideas"}) {
		t.Errorf("tags = %v", e.Tags)
	}
	out := h.out.String()
	if !strings.Contains(out, "code, life") || !strings.Contains(out, "content/posts/whats-in-a-type.md") {
		t.Errorf("output should list existing tags and the new path:\n%s", out)
	}

	err := h.run("\n", "new", "post", "What's in a type")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate slug: err = %v", err)
	}
}

func TestNewPostPromptsForTitle(t *testing.T) {
	h := newHarness(t, nil)
	if err := h.run("My Title\n\n", "new", "post"); err != nil {
		t.Fatal(err)
	}
	if e := h.entry("posts/my-title.md"); e.Title != "My Title" || len(e.Tags) != 0 {
		t.Errorf("post = %+v", e)
	}
}

func TestBookAddUsesOpenLibrary(t *testing.T) {
	h := newHarness(t, nil)
	h.search.docs = map[string][]openlibrary.Doc{"Piranesi": {
		{Title: "Piranesi", Authors: []string{"Susanna Clarke"}, FirstPublishYear: 2020, Pages: 272},
		{Title: "Piranesi Notes", Authors: []string{"Someone Else"}},
	}}
	if err := h.run("1\nFrom a friend.\n", "book", "add", "Piranesi"); err != nil {
		t.Fatal(err)
	}
	e := h.entry("books/piranesi.md")
	b := e.Book
	if !slices.Equal(b.Authors, []string{"Susanna Clarke"}) || b.Pages != 272 || b.Added.String() != "2026-10-25" {
		t.Errorf("book = %+v", b)
	}
	if b.Status() != content.StatusToRead || e.Body != "From a friend.\n" {
		t.Errorf("status %s body %q", b.Status(), e.Body)
	}
	if !strings.Contains(h.out.String(), "Susanna Clarke") {
		t.Errorf("candidates not shown:\n%s", h.out.String())
	}
}

func TestBookAddByHandWhenLookupFails(t *testing.T) {
	h := newHarness(t, nil)
	h.search.err = errors.New("offline")
	if err := h.run("Jane Doe\n\n", "book", "add", "Some Book"); err != nil {
		t.Fatal(err)
	}
	b := h.entry("books/some-book.md").Book
	if !slices.Equal(b.Authors, []string{"Jane Doe"}) || b.Pages != 0 {
		t.Errorf("book = %+v", b)
	}
	if !strings.Contains(h.out.String(), "offline") {
		t.Errorf("lookup error not reported:\n%s", h.out.String())
	}
}

func TestBookStartPicksFromToReadList(t *testing.T) {
	h := newHarness(t, map[string]string{
		"books/older.md": "---\ntitle: Older\nadded: 2026-01-01\n---\n",
		"books/newer.md": "---\ntitle: Newer\nadded: 2026-02-01\n---\n",
		"books/done.md":  "---\ntitle: Done\nfinished: 2025-01-01\n---\n",
	})
	if err := h.run("1\n", "book", "start"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "1) Newer") || strings.Contains(h.out.String(), "Done") {
		t.Errorf("list should show to-read books, newest first:\n%s", h.out.String())
	}
	if b := h.entry("books/newer.md").Book; b.Status() != content.StatusReading || b.Started.String() != "2026-10-25" {
		t.Errorf("newer = %+v", b)
	}
}

func TestBookFinish(t *testing.T) {
	h := newHarness(t, map[string]string{
		"books/james.md": "---\ntitle: James\nauthor: Percival Everett\nstarted: 2026-10-01\npages: 300\n---\n",
	})
	// An invalid rating is asked again; a blank page count keeps the old one.
	if err := h.run("meh\ngreat\nfiction\n\n", "book", "finish", "-date", "2026-10-20", "james"); err != nil {
		t.Fatal(err)
	}
	b := h.entry("books/james.md").Book
	if b.Status() != content.StatusRead || b.Finished.String() != "2026-10-20" {
		t.Errorf("finished = %s", b.Finished)
	}
	if b.Rating != content.RatingGreat || b.Form != content.FormFiction || b.Pages != 300 {
		t.Errorf("book = %+v", b)
	}
}

func TestBookFinishUnknownSlug(t *testing.T) {
	h := newHarness(t, nil)
	err := h.run("", "book", "finish", "nope")
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("err = %v", err)
	}
}

func TestPublish(t *testing.T) {
	h := newHarness(t, map[string]string{
		"posts/wip.md":  "---\ntitle: WIP\ndate: 2019-12-01\ndraft: true\n---\nbody\n",
		"posts/live.md": "---\ntitle: Live\ndate: 2019-12-02\n---\n",
	})
	if err := h.run("1\n", "publish"); err != nil {
		t.Fatal(err)
	}
	e := h.entry("posts/wip.md")
	if e.Draft || e.Date.String() != "2026-10-25" || e.Body != "body\n" {
		t.Errorf("published = %+v", e)
	}
}

func TestPublishKeepDate(t *testing.T) {
	h := newHarness(t, map[string]string{"posts/wip.md": "---\ntitle: WIP\ndate: 2019-12-01\ndraft: true\n---\n"})
	if err := h.run("", "publish", "-keep-date", "wip"); err != nil {
		t.Fatal(err)
	}
	if e := h.entry("posts/wip.md"); e.Draft || e.Date.String() != "2019-12-01" {
		t.Errorf("published = %+v", e)
	}
}

func TestStatus(t *testing.T) {
	h := newHarness(t, map[string]string{
		"books/james.md":   "---\ntitle: James\nauthor: Percival Everett\nstarted: 2026-10-01\n---\n",
		"books/someday.md": "---\ntitle: Someday\nadded: 2026-02-01\n---\n",
		"posts/wip.md":     "---\ntitle: WIP\ndate: 2019-12-01\ndraft: true\n---\n",
	})
	if err := h.run("", "status"); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	for _, want := range []string{"Reading now", "James", "Percival Everett", "Up next", "Someday", "Drafts", "WIP"} {
		if !strings.Contains(out, want) {
			t.Errorf("status missing %q:\n%s", want, out)
		}
	}
}

func TestPromptGivesUpAtEndOfInput(t *testing.T) {
	h := newHarness(t, map[string]string{"books/a.md": "---\ntitle: A\nadded: 2026-01-01\n---\n"})
	err := h.run("9\n", "book", "start")
	if err == nil {
		t.Fatal("expected error once input runs out")
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"The Handmaid's Tale":   "the-handmaids-tale",
		"Bringing Up Bébé":      "bringing-up-bebe",
		"11/22/63":              "11-22-63",
		"2001: A Space Odyssey": "2001-a-space-odyssey",
		"  What's in a Type?  ": "whats-in-a-type",
		"Ça va—très bien":       "ca-va-tres-bien",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommandHelpIsNotAnError(t *testing.T) {
	h := newHarness(t, nil)
	if err := h.run("", "book", "finish", "-h"); err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(h.out.String(), "usage: ryu book finish") {
		t.Errorf("output:\n%s", h.out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	h := newHarness(t, nil)
	err := h.run("", "boook", "add")
	var usage usageError
	if !errors.As(err, &usage) || !strings.Contains(err.Error(), "boook add") {
		t.Errorf("err = %v", err)
	}
}
