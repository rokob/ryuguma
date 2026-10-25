package content_test

import (
	"reflect"
	"testing"

	"github.com/rokob/ryuguma/internal/content"
)

// The encoded form is the file format I edit by hand, so pin it down exactly.
func TestEncodeBookFormat(t *testing.T) {
	e := content.Entry{
		Kind:  content.KindBook,
		Slug:  "lila",
		Path:  "books/lila.md",
		Title: "Lila: An Inquiry into Morals",
		Tags:  []string{"philosophy"},
		Body:  "Body text.\n",
		Book: &content.Book{
			Authors:  []string{"Robert M. Pirsig"},
			Series:   "Phaedrus",
			Number:   2,
			Link:     "https://www.amazon.com/dp/0553299611",
			Finished: content.MustDate("2017-03-10"),
			Pages:    468,
			Rating:   content.RatingGood,
			Form:     content.FormFiction,
		},
	}
	got, err := content.Encode(e)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := `---
title: 'Lila: An Inquiry into Morals'
author: Robert M. Pirsig
series: Phaedrus
number: 2
link: https://www.amazon.com/dp/0553299611
finished: 2017-03-10
pages: 468
rating: good
form: fiction
tags: [philosophy]
---

Body text.
`
	if string(got) != want {
		t.Errorf("Encode =\n%s\nwant\n%s", got, want)
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	entries := []content.Entry{
		{
			Kind: content.KindBook, Slug: "the-whole-brain-child", Path: "books/the-whole-brain-child.md",
			Title: "The Whole-Brain Child",
			Body:  "Some *notes*.\n",
			Book: &content.Book{
				Authors: []string{"Daniel J. Siegel", "Tina Payne Bryson"},
				Series:  "Parenting",
				Number:  1.5,
				ISBN:    "9780553386691",
				Added:   content.MustDate("2019-12-01"),
				Started: content.MustDate("2020-02-01"),
			},
		},
		{
			Kind: content.KindPost, Slug: "whats-in-a-type", Path: "posts/whats-in-a-type.md",
			Title: "What's in a type?", Date: content.MustDate("2014-05-17"),
			Summary: "Rambling: about types.", Tags: []string{"code", "go"}, Draft: true,
			Body: "# Heading\n\ntext\n",
		},
		{
			Kind: content.KindAtom, Slug: "an-idea", Path: "atoms/an-idea.md",
			Title: "An idea",
			Date:  content.MustDate("2026-10-24T15:04:05Z"),
			Body:  "A small idea.\n",
		},
	}
	for _, e := range entries {
		data, err := content.Encode(e)
		if err != nil {
			t.Fatalf("%s: Encode: %v", e.Path, err)
		}
		back, err := content.Parse(e.Path, data)
		if err != nil {
			t.Fatalf("%s: Parse: %v\n%s", e.Path, err, data)
		}
		if e.Kind == content.KindBook {
			// Date is derived from the lifecycle dates on parse.
			e.Date = back.Date
		}
		if !reflect.DeepEqual(back, e) {
			t.Errorf("%s: round trip mismatch\n got %+v\nwant %+v", e.Path, back, e)
		}
	}
}
