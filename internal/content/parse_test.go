package content_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/content"
)

func TestParseReadBook(t *testing.T) {
	src := `---
title: Dune
author: Frank Herbert
link: https://openlibrary.org/works/OL893415W
started: 2015-07-20
finished: 2015-08-03
pages: 896
rating: great
form: fiction
tags: [scifi, classic]
---

Spice.
`
	e, err := content.Parse("books/dune.md", []byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if e.Kind != content.KindBook || e.Slug != "dune" || e.Path != "books/dune.md" {
		t.Errorf("identity = %q %q %q", e.Kind, e.Slug, e.Path)
	}
	if e.Title != "Dune" {
		t.Errorf("Title = %q", e.Title)
	}
	if !slices.Equal(e.Tags, []string{"scifi", "classic"}) {
		t.Errorf("Tags = %v", e.Tags)
	}
	if strings.TrimSpace(e.Body) != "Spice." {
		t.Errorf("Body = %q", e.Body)
	}
	b := e.Book
	if b == nil {
		t.Fatal("Book is nil")
	}
	if !slices.Equal(b.Authors, []string{"Frank Herbert"}) {
		t.Errorf("Authors = %v", b.Authors)
	}
	if b.Link != "https://openlibrary.org/works/OL893415W" {
		t.Errorf("Link = %q", b.Link)
	}
	if b.Pages != 896 || b.Rating != content.RatingGreat || b.Form != content.FormFiction {
		t.Errorf("Pages/Rating/Form = %d %q %q", b.Pages, b.Rating, b.Form)
	}
	if b.Status() != content.StatusRead {
		t.Errorf("Status = %q", b.Status())
	}
	if got := e.Date.String(); got != "2015-08-03" {
		t.Errorf("Date = %q, want finished date", got)
	}
}

func TestParseBookMultipleAuthors(t *testing.T) {
	src := "---\ntitle: The Whole-Brain Child\nauthor: [Daniel J. Siegel, Tina Payne Bryson]\nadded: 2020-01-01\n---\n"
	e, err := content.Parse("books/the-whole-brain-child.md", []byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"Daniel J. Siegel", "Tina Payne Bryson"}
	if !slices.Equal(e.Book.Authors, want) {
		t.Errorf("Authors = %v, want %v", e.Book.Authors, want)
	}
}

func TestBookStatusFollowsLifecycleDates(t *testing.T) {
	cases := []struct {
		dates string
		want  content.Status
		date  string
	}{
		{"added: 2026-01-01", content.StatusToRead, "2026-01-01"},
		{"added: 2026-01-01\nstarted: 2026-02-01", content.StatusReading, "2026-02-01"},
		{"added: 2026-01-01\nstarted: 2026-02-01\nfinished: 2026-03-01", content.StatusRead, "2026-03-01"},
		{"finished: 2026-03-01", content.StatusRead, "2026-03-01"},
	}
	for _, c := range cases {
		src := "---\ntitle: X\n" + c.dates + "\n---\n"
		e, err := content.Parse("books/x.md", []byte(src))
		if err != nil {
			t.Fatalf("%q: Parse: %v", c.dates, err)
		}
		if got := e.Book.Status(); got != c.want {
			t.Errorf("%q: Status = %q, want %q", c.dates, got, c.want)
		}
		if got := e.Date.String(); got != c.date {
			t.Errorf("%q: Date = %q, want %q", c.dates, got, c.date)
		}
	}
}

func TestParsePost(t *testing.T) {
	src := `---
title: "What's in a type?"
date: 2014-05-17
summary: Rambling about types.
tags: [code, go]
draft: true
---
Body
`
	e, err := content.Parse("posts/whats-in-a-type.md", []byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if e.Kind != content.KindPost || e.Book != nil {
		t.Errorf("Kind = %q, Book = %v", e.Kind, e.Book)
	}
	if e.Title != "What's in a type?" || e.Summary != "Rambling about types." || !e.Draft {
		t.Errorf("got %+v", e)
	}
	if e.Date.String() != "2014-05-17" {
		t.Errorf("Date = %q", e.Date)
	}
}

func TestParseToleratesSloppyDelimiters(t *testing.T) {
	src := "--- \r\ntitle: X\r\ndate: 2020-01-01\r\n---  \r\n\r\nbody\r\n"
	e, err := content.Parse("posts/x.md", []byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if e.Title != "X" || e.Body != "body\n" {
		t.Errorf("Title %q Body %q", e.Title, e.Body)
	}
}

func TestParseKeepsTimeOfDay(t *testing.T) {
	src := "---\ntitle: An idea\ndate: 2026-10-24T15:04:05Z\n---\nan idea\n"
	e, err := content.Parse("atoms/an-idea.md", []byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := e.Date.String(); got != "2026-10-24T15:04:05Z" {
		t.Errorf("Date = %q", got)
	}
}

func TestParsePage(t *testing.T) {
	e, err := content.Parse("pages/about.md", []byte("---\ntitle: About\n---\nHi.\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if e.Kind != content.KindPage || e.Title != "About" || !e.Date.IsZero() {
		t.Errorf("got %+v", e)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, path, src, want string
	}{
		{"no frontmatter", "posts/x.md", "just text\n", "frontmatter"},
		{"unterminated", "posts/x.md", "---\ntitle: x\n", "frontmatter"},
		{"unknown key", "posts/x.md", "---\ntitle: x\ndate: 2020-01-01\nlayout: post\n---\n", "layout"},
		{"post missing date", "posts/x.md", "---\ntitle: x\n---\n", "date"},
		{"post missing title", "posts/x.md", "---\ndate: 2020-01-01\n---\n", "title"},
		{"book missing title", "books/x.md", "---\nadded: 2020-01-01\n---\n", "title"},
		{"bad rating", "books/x.md", "---\ntitle: x\nfinished: 2020-01-01\nrating: meh\n---\n", "rating"},
		{"bad form", "books/x.md", "---\ntitle: x\nform: poetry\n---\n", "form"},
		{"bad link", "books/x.md", "---\ntitle: x\nlink: amazon\n---\n", "link"},
		{"bad date", "posts/x.md", "---\ntitle: x\ndate: last tuesday\n---\n", "date"},
		{"finished before started", "books/x.md", "---\ntitle: x\nstarted: 2020-02-01\nfinished: 2020-01-01\n---\n", "before"},
		{"unknown directory", "notes/x.md", "---\ntitle: x\n---\n", "notes"},
		{"bad slug", "posts/Hello World.md", "---\ntitle: x\ndate: 2020-01-01\n---\n", "slug"},
		{"bad tag", "posts/x.md", "---\ntitle: x\ndate: 2020-01-01\ntags: [sci fi]\n---\n", "sci fi"},
		{"atom missing title", "atoms/x.md", "---\ndate: 2020-01-01\n---\nidea\n", "title"},
		{"atom without text", "atoms/x.md", "---\ntitle: x\ndate: 2020-01-01\n---\n\n", "text"},
		{"page missing title", "pages/x.md", "---\n---\n", "title"},
		{"page with date", "pages/x.md", "---\ntitle: x\ndate: 2020-01-01\n---\n", "date"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := content.Parse(c.path, []byte(c.src))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not mention %q", err, c.want)
			}
		})
	}
}
