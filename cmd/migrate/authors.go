package main

import (
	"cmp"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/openlibrary"
)

type searcher interface {
	Search(ctx context.Context, title string) ([]openlibrary.Doc, error)
}

// authorRow is one line of the review file.
type authorRow struct {
	Slug    string
	Match   string
	Title   string
	Found   string // title of the matched Open Library work
	Year    int
	Authors []string
}

var matchOrder = []string{"error", "none", "guess", "subtitle", "exact"}

const lookupAttempts = 3

// lookupAuthors searches for every book without authors. wait is called
// before each request so the caller can rate limit. A title that keeps
// failing gets an "error" row rather than stopping the whole run.
func lookupAuthors(ctx context.Context, s searcher, books []content.Entry, wait func()) ([]authorRow, error) {
	var rows []authorRow
	for _, e := range books {
		if e.Book == nil || len(e.Book.Authors) > 0 {
			continue
		}
		var (
			docs []openlibrary.Doc
			err  error
		)
		for range lookupAttempts {
			wait()
			if docs, err = s.Search(ctx, e.Title); err == nil {
				break
			}
		}
		if err != nil {
			rows = append(rows, authorRow{Slug: e.Slug, Match: "error", Title: e.Title, Found: err.Error()})
			continue
		}
		d, m := openlibrary.Best(docs, e.Title, e.Book.Pages)
		rows = append(rows, authorRow{
			Slug: e.Slug, Match: m.String(), Title: e.Title,
			Found: d.Title, Year: d.FirstPublishYear, Authors: d.Authors,
		})
	}
	slices.SortFunc(rows, func(a, b authorRow) int {
		return cmp.Or(
			cmp.Compare(slices.Index(matchOrder, a.Match), slices.Index(matchOrder, b.Match)),
			cmp.Compare(a.Slug, b.Slug),
		)
	})
	return rows, nil
}

var header = []string{"slug", "match", "title", "found", "year", "authors"}

func writeRows(w io.Writer, rows []authorRow) error {
	cw := csv.NewWriter(w)
	cw.Comma = '\t'
	cw.Write(header)
	for _, r := range rows {
		year := ""
		if r.Year != 0 {
			year = strconv.Itoa(r.Year)
		}
		cw.Write([]string{r.Slug, r.Match, r.Title, r.Found, year, strings.Join(r.Authors, "; ")})
	}
	cw.Flush()
	return cw.Error()
}

func readRows(r io.Reader) ([]authorRow, error) {
	cr := csv.NewReader(r)
	cr.Comma = '\t'
	cr.LazyQuotes = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	var rows []authorRow
	for i, rec := range records {
		if i == 0 {
			continue
		}
		if len(rec) != len(header) {
			return nil, fmt.Errorf("line %d: want %d columns, got %d", i+1, len(header), len(rec))
		}
		row := authorRow{Slug: rec[0], Match: rec[1], Title: rec[2], Found: rec[3]}
		row.Year, _ = strconv.Atoi(rec[4])
		for a := range strings.SplitSeq(rec[5], ";") {
			if a = strings.TrimSpace(a); a != "" {
				row.Authors = append(row.Authors, a)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// applyAuthors returns the book files whose authors changed.
func applyAuthors(files map[string][]byte, rows []authorRow) (map[string][]byte, error) {
	out := map[string][]byte{}
	var errs []error
	for _, r := range rows {
		if len(r.Authors) == 0 {
			continue
		}
		p := "books/" + r.Slug + ".md"
		data, ok := files[p]
		if !ok {
			errs = append(errs, fmt.Errorf("no book with slug %q", r.Slug))
			continue
		}
		e, err := content.Parse(p, data)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		e.Book.Authors = r.Authors
		encoded, err := content.Encode(e)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out[p] = encoded
	}
	return out, errors.Join(errs...)
}
