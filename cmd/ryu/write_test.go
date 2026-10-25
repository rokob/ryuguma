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
		{Key: "/works/OL20111013W", Title: "Piranesi", Authors: []string{"Susanna Clarke"}, FirstPublishYear: 2020, Pages: 272},
		{Title: "Piranesi Notes", Authors: []string{"Someone Else"}},
	}}
	// Pick the first match, accept its Open Library page as the link, add a note.
	if err := h.run("1\n\nFrom a friend.\n", "book", "add", "Piranesi"); err != nil {
		t.Fatal(err)
	}
	e := h.entry("books/piranesi.md")
	b := e.Book
	if !slices.Equal(b.Authors, []string{"Susanna Clarke"}) || b.Pages != 272 || b.Added.String() != "2026-10-25" {
		t.Errorf("book = %+v", b)
	}
	if b.Link != "https://openlibrary.org/works/OL20111013W" {
		t.Errorf("Link = %q, want the Open Library page by default", b.Link)
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
	// Something that isn't a URL is asked again.
	if err := h.run("Jane Doe\nnot a url\nhttps://example.com/some-book\n\n", "book", "add", "Some Book"); err != nil {
		t.Fatal(err)
	}
	b := h.entry("books/some-book.md").Book
	if !slices.Equal(b.Authors, []string{"Jane Doe"}) || b.Pages != 0 {
		t.Errorf("book = %+v", b)
	}
	if b.Link != "https://example.com/some-book" {
		t.Errorf("Link = %q", b.Link)
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
	if err := h.run("meh\ngreat\nfiction\n\nhttps://www.amazon.com/dp/B0CHVGKQH1\n", "book", "finish", "-date", "2026-10-20", "james"); err != nil {
		t.Fatal(err)
	}
	b := h.entry("books/james.md").Book
	if b.Status() != content.StatusRead || b.Finished.String() != "2026-10-20" {
		t.Errorf("finished = %s", b.Finished)
	}
	if b.Link != "https://www.amazon.com/dp/B0CHVGKQH1" {
		t.Errorf("Link = %q", b.Link)
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
	if !strings.Contains(h.out.String(), "usage: ./ryu book finish") {
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

func TestNewAtom(t *testing.T) {
	h := newHarness(t, nil)
	input := "ideas\nIt is easier to say more when you don't understand something.\n\nShort communication requires deeper knowledge.\n"
	if err := h.run(input, "new", "atom", "Short is better than long"); err != nil {
		t.Fatal(err)
	}
	e := h.entry("atoms/short-is-better-than-long.md")
	if e.Title != "Short is better than long" || e.Date.String() != "2026-10-25" || e.Draft {
		t.Errorf("atom = %+v, want titled, dated today, published", e)
	}
	if !slices.Equal(e.Tags, []string{"ideas"}) {
		t.Errorf("tags = %v", e.Tags)
	}
	want := "It is easier to say more when you don't understand something.\n\nShort communication requires deeper knowledge.\n"
	if e.Body != want {
		t.Errorf("body = %q, want %q", e.Body, want)
	}
}

func TestNewAtomNeedsText(t *testing.T) {
	h := newHarness(t, nil)
	err := h.run("\n", "new", "atom", "Empty")
	if err == nil || !strings.Contains(err.Error(), "text") {
		t.Errorf("err = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(h.root, "content", "atoms", "empty.md")); statErr == nil {
		t.Error("an empty atom was written")
	}
}

const readOdyssey = "---\ntitle: '2001: A Space Odyssey'\nauthor: Arthur C. Clarke\nseries: Space Odyssey\nnumber: 1\nlink: https://www.amazon.com/dp/B000FC0PBC\nfinished: 2019-06-02\npages: 297\nrating: great\nform: fiction\n---\n\nReview.\n"

func TestBookAddRereadMakesALinkedEntry(t *testing.T) {
	h := newHarness(t, map[string]string{"books/2001-a-space-odyssey.md": readOdyssey})
	h.search.err = errors.New("should not search for a re-read")
	if err := h.run("r\n\n", "book", "add", "2001: a Space Odyssey"); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	if !strings.Contains(out, "You already have 2001: A Space Odyssey by Arthur C. Clarke") || !strings.Contains(out, "finished Jun 2, 2019") {
		t.Errorf("should describe the existing book:\n%s", out)
	}
	e := h.entry("books/2001-a-space-odyssey-2026.md")
	b := e.Book
	if e.Title != "2001: A Space Odyssey" || !slices.Equal(b.Authors, []string{"Arthur C. Clarke"}) {
		t.Errorf("re-read = %+v", e)
	}
	if b.Link != "https://www.amazon.com/dp/B000FC0PBC" || b.Pages != 297 || b.Form != content.FormFiction {
		t.Errorf("details should be copied from the first read: %+v", b)
	}
	if b.Series != "Space Odyssey" || b.Number != 1 {
		t.Errorf("series should be copied from the first read: %q %v", b.Series, b.Number)
	}
	if b.Status() != content.StatusToRead || b.Rating != "" || b.Added.String() != "2026-10-25" {
		t.Errorf("re-read should start on the to-read list: %+v", b)
	}
	if !strings.Contains(e.Body, "[[2001-a-space-odyssey]]") {
		t.Errorf("body should link the first read: %q", e.Body)
	}
}

func TestBookAddSameTitleDifferentBook(t *testing.T) {
	h := newHarness(t, map[string]string{
		"books/the-power-of-the-dog.md": "---\ntitle: The Power of the Dog\nauthor: Don Winslow\nfinished: 2016-01-07\n---\n",
	})
	h.search.docs = map[string][]openlibrary.Doc{"The Power of the Dog": {
		{Key: "/works/OL1W", Title: "The Power of the Dog", Authors: []string{"Thomas Savage"}, Pages: 276},
	}}
	if err := h.run("d\n1\n\n\n", "book", "add", "The Power of the Dog"); err != nil {
		t.Fatal(err)
	}
	e := h.entry("books/the-power-of-the-dog-savage.md")
	if e.Title != "The Power of the Dog" || !slices.Equal(e.Book.Authors, []string{"Thomas Savage"}) {
		t.Errorf("entry = %+v", e)
	}
	if !strings.Contains(h.out.String(), "the-power-of-the-dog-savage") {
		t.Errorf("should say which slug it used:\n%s", h.out.String())
	}
}

func TestBookAddAlreadyOnTheList(t *testing.T) {
	h := newHarness(t, map[string]string{"books/someday.md": "---\ntitle: Someday\nadded: 2026-02-01\n---\n"})
	// A book you haven't read can't be re-read, so only "different" or "cancel".
	if err := h.run("r\nc\n", "book", "add", "Someday"); err != nil {
		t.Fatalf("cancel should not be an error: %v", err)
	}
	out := h.out.String()
	if !strings.Contains(out, "on your to-read list since Feb 1, 2026") || !strings.Contains(out, "Nothing added") {
		t.Errorf("output:\n%s", out)
	}
	entries, _ := os.ReadDir(filepath.Join(h.root, "content", "books"))
	if len(entries) != 1 {
		t.Errorf("files = %d, want nothing new", len(entries))
	}
}

func TestBookAddCustomSlug(t *testing.T) {
	h := newHarness(t, nil)
	if err := h.run("Jane Doe\n\n\n", "book", "add", "-slug", "my-slug", "Some Book"); err != nil {
		t.Fatal(err)
	}
	if e := h.entry("books/my-slug.md"); e.Title != "Some Book" {
		t.Errorf("entry = %+v", e)
	}
	if err := h.run("", "book", "add", "-slug", "Not A Slug", "X"); err == nil {
		t.Error("expected an error for an invalid slug")
	}
}

func TestBookAddPickerExplainsPageCounts(t *testing.T) {
	h := newHarness(t, nil)
	h.search.docs = map[string][]openlibrary.Doc{"2001: a Space Odyssey": {
		{Title: "The Making of 2001", Authors: []string{"Jay Cocks"}, Pages: 317, Editions: 1},
		{Key: "/works/OL2W", Title: "2001 A SPACE ODYSSEY", Authors: []string{"Arthur C. Clarke"}, FirstPublishYear: 1968, Pages: 213, Editions: 2},
	}}
	if err := h.run("1\n\n\n", "book", "add", "2001: a Space Odyssey"); err != nil {
		t.Fatal(err)
	}
	out := h.out.String()
	if !strings.Contains(out, "1) 2001 A SPACE ODYSSEY — Arthur C. Clarke, 1968, 2 editions (median 213 pages)") {
		t.Errorf("exact match should come first with an explained page count:\n%s", out)
	}
	if !strings.Contains(out, "2) The Making of 2001 — Jay Cocks, 317 pages") {
		t.Errorf("single-edition page count:\n%s", out)
	}
}

func TestChooseAcceptsAUniquePrefix(t *testing.T) {
	h := newHarness(t, map[string]string{
		"books/james.md": "---\ntitle: James\nstarted: 2026-10-01\npages: 300\n---\n",
	})
	// "g" could be good or great, so it is asked again.
	if err := h.run("g\ngre\nnon\n\n\n", "book", "finish", "james"); err != nil {
		t.Fatal(err)
	}
	if b := h.entry("books/james.md").Book; b.Rating != content.RatingGreat || b.Form != content.FormNonfiction {
		t.Errorf("book = %+v", b)
	}
}
