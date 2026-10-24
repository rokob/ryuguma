package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/openlibrary"
)

type fakeSearcher map[string][]openlibrary.Doc

func (f fakeSearcher) Search(_ context.Context, title string) ([]openlibrary.Doc, error) {
	return f[title], nil
}

func TestLookupAuthorsListsDoubtfulMatchesFirst(t *testing.T) {
	books := []content.Entry{
		{Kind: content.KindBook, Slug: "dune", Title: "Dune", Book: &content.Book{Pages: 896}},
		{Kind: content.KindBook, Slug: "ice", Title: "Ice", Book: &content.Book{Pages: 200}},
		{Kind: content.KindBook, Slug: "has-author", Title: "Has Author", Book: &content.Book{Authors: []string{"Known"}}},
	}
	search := fakeSearcher{
		"Dune": {{Title: "Dune", Authors: []string{"Frank Herbert"}, FirstPublishYear: 1965}},
		"Ice":  {{Title: "Ice Station", Authors: []string{"Matthew Reilly"}}},
	}
	rows, err := lookupAuthors(context.Background(), search, books, func() {})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v, want books without authors only", rows)
	}
	if rows[0].Slug != "ice" || rows[0].Match != "guess" || rows[1].Slug != "dune" || rows[1].Match != "exact" {
		t.Errorf("rows = %+v", rows)
	}
	if !slices.Equal(rows[1].Authors, []string{"Frank Herbert"}) || rows[1].Year != 1965 {
		t.Errorf("dune row = %+v", rows[1])
	}
}

type flakySearcher struct {
	failures map[string]int // remaining failures per title
}

func (f *flakySearcher) Search(_ context.Context, title string) ([]openlibrary.Doc, error) {
	if f.failures[title] > 0 {
		f.failures[title]--
		return nil, errors.New("timeout")
	}
	return []openlibrary.Doc{{Title: title, Authors: []string{"Author of " + title}}}, nil
}

func TestLookupAuthorsRetriesThenRecordsErrors(t *testing.T) {
	books := []content.Entry{
		{Kind: content.KindBook, Slug: "flaky", Title: "Flaky", Book: &content.Book{}},
		{Kind: content.KindBook, Slug: "down", Title: "Down", Book: &content.Book{}},
	}
	s := &flakySearcher{failures: map[string]int{"Flaky": 2, "Down": 100}}
	waits := 0
	rows, err := lookupAuthors(context.Background(), s, books, func() { waits++ })
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Slug != "down" || rows[0].Match != "error" {
		t.Fatalf("rows = %+v, want failed lookup first", rows)
	}
	if rows[1].Slug != "flaky" || rows[1].Match != "exact" {
		t.Errorf("flaky row = %+v, want success after retries", rows[1])
	}
	if waits != 6 {
		t.Errorf("waited %d times, want once per attempt (3 + 3)", waits)
	}
}

func TestAuthorRowsRoundTripThroughTSV(t *testing.T) {
	rows := []authorRow{
		{Slug: "the-whole-brain-child", Match: "exact", Title: "The Whole-Brain Child", Found: "The Whole-Brain Child", Year: 2011, Authors: []string{"Daniel J. Siegel", "Tina Payne Bryson"}},
		{Slug: "ice", Match: "none", Title: "Ice"},
	}
	var buf bytes.Buffer
	if err := writeRows(&buf, rows); err != nil {
		t.Fatal(err)
	}
	back, err := readRows(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 2 || !slices.Equal(back[0].Authors, rows[0].Authors) || back[1].Authors != nil {
		t.Errorf("back = %+v", back)
	}
}

func TestApplyAuthors(t *testing.T) {
	files := map[string][]byte{
		"books/dune.md": []byte("---\ntitle: Dune\nfinished: 2015-08-03\n---\n\nSpice.\n"),
		"books/ice.md":  []byte("---\ntitle: Ice\nfinished: 2019-02-10\n---\n"),
	}
	rows := []authorRow{
		{Slug: "dune", Authors: []string{"Frank Herbert"}},
		{Slug: "ice"}, // left blank in review: unchanged
	}
	out, err := applyAuthors(files, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("changed %d files, want 1", len(out))
	}
	e, err := content.Parse("books/dune.md", out["books/dune.md"])
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(e.Book.Authors, []string{"Frank Herbert"}) || e.Body != "Spice.\n" {
		t.Errorf("got %+v", e)
	}
}

func TestApplyAuthorsUnknownSlug(t *testing.T) {
	_, err := applyAuthors(map[string][]byte{}, []authorRow{{Slug: "nope", Authors: []string{"X"}}})
	if err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("err = %v", err)
	}
}
