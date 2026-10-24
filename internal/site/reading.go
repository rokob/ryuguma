package site

import (
	"cmp"
	"slices"
	"time"

	"github.com/rokob/ryuguma/internal/content"
)

// ReadingStats summarizes read books for the reading page.
type ReadingStats struct {
	Books    int
	Pages    int
	AvgPages int
	First    content.Date
	// Years runs from the current year back to the first, including years
	// with no books.
	Years    []YearStats
	ThisYear YearStats

	MaxYearBooks  int
	MaxYearPages  int
	MaxMonthBooks int

	Longest []*Page
	// Authors holds up to ten authors with at least two books, most read first.
	Authors []AuthorCount
}

type YearStats struct {
	Year       int
	Books      int
	Pages      int
	AvgPages   int
	Fiction    int
	Nonfiction int
	// Ratings counts books per rating, in content.Ratings order.
	Ratings [4]int
	Months  [12]MonthStats
}

type MonthStats struct {
	Month  time.Month
	Books  int
	Pages  int
	Future bool
}

type AuthorCount struct {
	Name  string
	Books int
}

// Reading computes stats over read books, newest first. now decides the
// current year and which months haven't happened yet.
func Reading(books []*Page, now time.Time) ReadingStats {
	r := ReadingStats{ThisYear: YearStats{Year: now.Year()}}
	if len(books) == 0 {
		return r
	}
	r.First = books[0].Book.Finished
	for _, b := range books {
		if b.Book.Finished.Before(r.First) {
			r.First = b.Book.Finished
		}
	}

	byYear := map[int]*YearStats{}
	lastYear := max(now.Year(), books[0].Book.Finished.Time().Year())
	for y := lastYear; y >= r.First.Time().Year(); y-- {
		r.Years = append(r.Years, YearStats{Year: y})
	}
	for i := range r.Years {
		ys := &r.Years[i]
		byYear[ys.Year] = ys
		for m := range ys.Months {
			ys.Months[m].Month = time.Month(m + 1)
			ys.Months[m].Future = time.Date(ys.Year, time.Month(m+1), 1, 0, 0, 0, 0, time.UTC).After(now)
		}
	}

	authors := map[string]int{}
	withPages := 0
	for _, b := range books {
		t := b.Book.Finished.Time()
		ys := byYear[t.Year()]
		ms := &ys.Months[t.Month()-1]
		r.Books++
		ys.Books++
		ms.Books++
		r.Pages += b.Book.Pages
		ys.Pages += b.Book.Pages
		ms.Pages += b.Book.Pages
		if b.Book.Pages > 0 {
			withPages++
		}
		switch b.Book.Form {
		case content.FormFiction:
			ys.Fiction++
		case content.FormNonfiction:
			ys.Nonfiction++
		}
		if i := slices.Index(content.Ratings, b.Book.Rating); i >= 0 {
			ys.Ratings[i]++
		}
		for _, a := range b.Book.Authors {
			authors[a]++
		}
	}
	if withPages > 0 {
		r.AvgPages = r.Pages / withPages
	}

	for i := range r.Years {
		ys := &r.Years[i]
		if ys.Books > 0 {
			ys.AvgPages = ys.Pages / ys.Books
		}
		r.MaxYearBooks = max(r.MaxYearBooks, ys.Books)
		r.MaxYearPages = max(r.MaxYearPages, ys.Pages)
		for _, m := range ys.Months {
			r.MaxMonthBooks = max(r.MaxMonthBooks, m.Books)
		}
		if ys.Year == now.Year() {
			r.ThisYear = *ys
		}
	}

	r.Longest = slices.Clone(books)
	slices.SortStableFunc(r.Longest, func(a, b *Page) int { return cmp.Compare(b.Book.Pages, a.Book.Pages) })
	r.Longest = r.Longest[:min(5, len(r.Longest))]

	for name, n := range authors {
		if n >= 2 {
			r.Authors = append(r.Authors, AuthorCount{Name: name, Books: n})
		}
	}
	slices.SortFunc(r.Authors, func(a, b AuthorCount) int {
		return cmp.Or(cmp.Compare(b.Books, a.Books), cmp.Compare(a.Name, b.Name))
	})
	r.Authors = r.Authors[:min(10, len(r.Authors))]
	return r
}
