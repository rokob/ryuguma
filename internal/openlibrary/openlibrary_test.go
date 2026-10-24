package openlibrary_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/rokob/ryuguma/internal/openlibrary"
)

func TestSearch(t *testing.T) {
	var gotQuery, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("title")
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte(`{"docs":[{"key":"/works/OL1W","title":"Dune","author_name":["Frank Herbert"],"first_publish_year":1965,"number_of_pages_median":612,"edition_count":96}]}`))
	}))
	defer srv.Close()

	c := &openlibrary.Client{BaseURL: srv.URL, HTTP: srv.Client(), UserAgent: "test-agent"}
	docs, err := c.Search(context.Background(), "Dune")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotQuery != "Dune" || gotUA != "test-agent" {
		t.Errorf("query %q ua %q", gotQuery, gotUA)
	}
	want := openlibrary.Doc{Key: "/works/OL1W", Title: "Dune", Authors: []string{"Frank Herbert"}, FirstPublishYear: 1965, Pages: 612}
	if len(docs) != 1 || docs[0].Title != want.Title || !slices.Equal(docs[0].Authors, want.Authors) || docs[0].Pages != 612 || docs[0].Editions != 96 {
		t.Errorf("docs = %+v", docs)
	}
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := &openlibrary.Client{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Search(context.Background(), "Dune"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBestPrefersExactTitleWithClosestPages(t *testing.T) {
	docs := []openlibrary.Doc{
		{Title: "It's Not Easy", Authors: []string{"Someone"}},
		{Title: "IT", Authors: []string{"Wrong Edition"}, Pages: 300},
		{Title: "It", Authors: []string{"Stephen King"}, Pages: 1138},
		{Title: "It", Pages: 1100},
	}
	d, m := openlibrary.Best(docs, "It", 1153)
	if m != openlibrary.MatchExact || d.Authors[0] != "Stephen King" {
		t.Errorf("got %+v %v", d, m)
	}
}

func TestBestNormalizesTitles(t *testing.T) {
	docs := []openlibrary.Doc{{Title: "The Handmaid's Tale", Authors: []string{"Margaret Atwood"}}}
	if _, m := openlibrary.Best(docs, "The Handmaid’s Tale", 0); m != openlibrary.MatchExact {
		t.Errorf("match = %v", m)
	}
	docs = []openlibrary.Doc{{Title: "Inside Delta Force: The Story of America's Elite Counterterrorist Unit", Authors: []string{"Eric L. Haney"}}}
	if _, m := openlibrary.Best(docs, "Inside Delta Force", 0); m != openlibrary.MatchSubtitle {
		t.Errorf("match = %v", m)
	}
	docs = []openlibrary.Doc{{Title: "2666", Authors: []string{"Roberto Bolaño"}}}
	if _, m := openlibrary.Best(docs, "2666: A Novel", 0); m != openlibrary.MatchSubtitle {
		t.Errorf("our subtitle: match = %v", m)
	}
	docs = []openlibrary.Doc{{Title: "Ice Station", Authors: []string{"Matthew Reilly"}}}
	if _, m := openlibrary.Best(docs, "Ice", 0); m != openlibrary.MatchGuess {
		t.Errorf("longer title without subtitle: match = %v", m)
	}
}

func TestBestFallsBackToFirstWithAuthors(t *testing.T) {
	docs := []openlibrary.Doc{
		{Title: "Something Else"},
		{Title: "Another Thing", Authors: []string{"A. Person"}},
	}
	d, m := openlibrary.Best(docs, "Unrelated", 0)
	if m != openlibrary.MatchGuess || d.Authors[0] != "A. Person" {
		t.Errorf("got %+v %v", d, m)
	}
	if _, m := openlibrary.Best(nil, "Anything", 0); m != openlibrary.MatchNone {
		t.Errorf("empty: match = %v", m)
	}
}

func TestRankPutsTitleMatchesFirst(t *testing.T) {
	docs := []openlibrary.Doc{
		{Title: "The Making of Stanley Kubrick's '2001: A Space Odyssey'"},
		{Title: "The Making of 2001, a space odyssey"},
		{Title: "2001 A SPACE ODYSSEY", Authors: []string{"Arthur C. Clarke"}},
		{Title: "2001 : a space odyssey", Authors: []string{"Stanley Kubrick"}},
	}
	ranked := openlibrary.Rank(docs, "2001: a Space Odyssey")
	var got []string
	for _, d := range ranked {
		got = append(got, d.Title)
	}
	want := []string{"2001 A SPACE ODYSSEY", "2001 : a space odyssey", "The Making of Stanley Kubrick's '2001: A Space Odyssey'", "The Making of 2001, a space odyssey"}
	if !slices.Equal(got, want) {
		t.Errorf("Rank =\n%q\nwant\n%q", got, want)
	}
}
