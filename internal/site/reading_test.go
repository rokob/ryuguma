package site_test

import (
	"testing"
	"time"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/site"
)

func readBook(slug, finished string, pages int, rating content.Rating, form content.Form, authors ...string) *site.Page {
	d := content.MustDate(finished)
	return &site.Page{Entry: content.Entry{
		Kind: content.KindBook, Slug: slug, Title: slug, Date: d,
		Book: &content.Book{Finished: d, Pages: pages, Rating: rating, Form: form, Authors: authors},
	}}
}

func TestReadingStats(t *testing.T) {
	books := []*site.Page{ // newest first, as site.Books is
		readBook("e", "2026-03-10", 100, content.RatingGreat, content.FormFiction, "Le Guin"),
		readBook("d", "2024-01-05", 900, content.RatingGood, content.FormNonfiction, "Caro"),
		readBook("c", "2024-01-20", 300, content.RatingOkay, content.FormFiction, "Le Guin"),
		readBook("b", "2022-12-01", 200, content.RatingGood, content.FormFiction, "Le Guin", "Someone"),
		readBook("a", "2022-02-01", 0, content.RatingBad, content.FormNonfiction),
	}
	now := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	r := site.Reading(books, now)

	if r.Books != 5 || r.Pages != 1500 || r.First.String() != "2022-02-01" {
		t.Errorf("totals = %d books %d pages first %s", r.Books, r.Pages, r.First)
	}
	if r.AvgPages != 375 {
		t.Errorf("AvgPages = %d, want mean over books with page counts", r.AvgPages)
	}
	var years []int
	for _, y := range r.Years {
		years = append(years, y.Year)
	}
	if len(years) != 5 || years[0] != 2026 || years[2] != 2024 || years[4] != 2022 {
		t.Errorf("years = %v, want 2026 down to 2022 including empty years", years)
	}
	y2024 := r.Years[2]
	if y2024.Books != 2 || y2024.Pages != 1200 || y2024.Fiction != 1 || y2024.AvgPages != 600 {
		t.Errorf("2024 = %+v", y2024)
	}
	if y2024.Months[0].Books != 2 || y2024.Months[0].Pages != 1200 {
		t.Errorf("Jan 2024 = %+v", y2024.Months[0])
	}
	if y2024.Ratings != [4]int{0, 1, 1, 0} {
		t.Errorf("2024 ratings = %v", y2024.Ratings)
	}
	if r.Years[1].Books != 0 {
		t.Errorf("2025 should be an empty year, got %+v", r.Years[1])
	}
	if r.MaxYearBooks != 2 || r.MaxYearPages != 1200 || r.MaxMonthBooks != 2 {
		t.Errorf("max = %d %d %d", r.MaxYearBooks, r.MaxYearPages, r.MaxMonthBooks)
	}
	if !r.Years[0].Months[5].Future || r.Years[0].Months[4].Future || r.Years[1].Months[11].Future {
		t.Error("only months after now should be future")
	}
	if r.ThisYear.Year != 2026 || r.ThisYear.Books != 1 {
		t.Errorf("ThisYear = %+v", r.ThisYear)
	}
	if len(r.Longest) != 5 || r.Longest[0].Slug != "d" || r.Longest[1].Slug != "c" {
		t.Errorf("Longest = %v", r.Longest)
	}
	if len(r.Authors) != 1 || r.Authors[0].Name != "Le Guin" || r.Authors[0].Books != 3 {
		t.Errorf("Authors = %+v, want only authors with 2+ books", r.Authors)
	}
}

func TestReadingStatsEmpty(t *testing.T) {
	r := site.Reading(nil, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if r.Books != 0 || len(r.Years) != 0 || r.ThisYear.Year != 2026 {
		t.Errorf("empty = %+v", r)
	}
}
