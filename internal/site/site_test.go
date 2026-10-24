package site_test

import (
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/site"
)

func book(slug, title string, b content.Book, body string, tags ...string) content.Entry {
	e := content.Entry{Kind: content.KindBook, Slug: slug, Path: "books/" + slug + ".md", Title: title, Tags: tags, Body: body, Book: &b}
	switch {
	case !b.Finished.IsZero():
		e.Date = b.Finished
	case !b.Started.IsZero():
		e.Date = b.Started
	default:
		e.Date = b.Added
	}
	return e
}

func post(slug, date, body string, tags ...string) content.Entry {
	return content.Entry{Kind: content.KindPost, Slug: slug, Path: "posts/" + slug + ".md", Title: slug, Date: content.MustDate(date), Tags: tags, Body: body}
}

func atom(slug, date, body string) content.Entry {
	return content.Entry{Kind: content.KindAtom, Slug: slug, Path: "atoms/" + slug + ".md", Date: content.MustDate(date), Body: body}
}

func slugs(pages []*site.Page) string {
	var s []string
	for _, p := range pages {
		s = append(s, p.Slug)
	}
	return strings.Join(s, ",")
}

func fixture() []content.Entry {
	return []content.Entry{
		book("dune", "Dune", content.Book{Finished: content.MustDate("2015-08-03")}, "", "scifi"),
		book("ubik", "Ubik", content.Book{Finished: content.MustDate("2015-12-22")}, "Better than [[dune]]?", "scifi"),
		book("james", "James", content.Book{Added: content.MustDate("2026-01-01"), Started: content.MustDate("2026-05-01")}, ""),
		book("someday", "Someday", content.Book{Added: content.MustDate("2026-02-01")}, ""),
		book("later", "Later", content.Book{Added: content.MustDate("2026-03-01")}, ""),
		post("hello", "2014-01-01", "Reading [[dune]] next.", "life"),
		atom("idea", "2026-06-01T10:00:00Z", "An idea about [[ubik]] and [[hello]]."),
	}
}

func TestBuildGroupsAndOrders(t *testing.T) {
	s, err := site.Build(fixture(), site.Options{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	checks := []struct {
		name, got, want string
	}{
		{"all", slugs(s.Pages), "idea,james,later,someday,ubik,dune,hello"},
		{"read", slugs(s.Books), "ubik,dune"},
		{"reading", slugs(s.Reading), "james"},
		{"to-read", slugs(s.ToRead), "later,someday"},
		{"posts", slugs(s.Posts), "hello"},
		{"atoms", slugs(s.Atoms), "idea"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %s, want %s", c.name, c.got, c.want)
		}
	}
	if p := s.Lookup("ubik"); p == nil || p.URL != "/books/ubik/" || !strings.Contains(string(p.HTML), `href="/books/dune/"`) {
		t.Errorf("ubik page = %+v", p)
	}
}

func TestBuildBacklinks(t *testing.T) {
	s, err := site.Build(fixture(), site.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := slugs(s.Lookup("dune").Backlinks); got != "ubik,hello" {
		t.Errorf("dune backlinks = %s, want newest first", got)
	}
	if got := slugs(s.Lookup("idea").Refs); got != "ubik,hello" {
		t.Errorf("idea refs = %s", got)
	}
	if got := slugs(s.Lookup("ubik").Backlinks); got != "idea" {
		t.Errorf("ubik backlinks = %s", got)
	}
}

func TestBuildTags(t *testing.T) {
	entries := append(fixture(), post("bye", "2015-01-01", "", "life", "scifi"))
	s, err := site.Build(entries, site.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tag := range s.Tags {
		got = append(got, tag.Name+"="+slugs(tag.Pages))
	}
	want := "life=bye,hello scifi=ubik,dune,bye"
	if strings.Join(got, " ") != want {
		t.Errorf("tags = %v, want %s", got, want)
	}
	if s.Tags[0].URL != "/tags/life/" {
		t.Errorf("tag URL = %s", s.Tags[0].URL)
	}
}

func TestDraftsExcludedUnlessRequested(t *testing.T) {
	d := post("wip", "2020-01-01", "")
	d.Draft = true
	entries := append(fixture(), d)

	s, err := site.Build(entries, site.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if s.Lookup("wip") != nil {
		t.Error("draft published")
	}
	s, err = site.Build(entries, site.Options{Drafts: true})
	if err != nil {
		t.Fatal(err)
	}
	if s.Lookup("wip") == nil {
		t.Error("draft missing with Drafts: true")
	}
}

func TestRefToDraftOrMissingIsAnError(t *testing.T) {
	d := post("wip", "2020-01-01", "")
	d.Draft = true
	entries := []content.Entry{d, post("pub", "2020-01-02", "See [[wip]] and [[gone]].")}
	_, err := site.Build(entries, site.Options{})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"posts/pub.md", "wip", "gone"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}
