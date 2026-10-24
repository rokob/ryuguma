package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/content"
)

var testTargets = map[string]target{
	"annihilation":           {"Annihilation", "books"},
	"the-great-leveler":      {"The Great Leveler", "books"},
	"submergence":            {"Submergence", "books"},
	"bringing-up-bebe":       {"Bringing Up Bébé", "books"},
	"another-year-in-review": {"Another Year in Review", "posts"},
}

func TestConvertBook(t *testing.T) {
	src := `---
layout: post
title: The Martian
date: 2015-01-26 20:01:38
categories: book
tags:
- book
- fiction
- scifi
- 'okay '
pagecount: 385
---

Somehow it ended up on my list.
`
	e, err := convert("books/the-martian.md", []byte(src), testTargets)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	b := e.Book
	if e.Title != "The Martian" || b.Finished.String() != "2015-01-26" || b.Pages != 385 {
		t.Errorf("got title %q finished %s pages %d", e.Title, b.Finished, b.Pages)
	}
	if b.Rating != content.RatingOkay || b.Form != content.FormFiction {
		t.Errorf("rating %q form %q", b.Rating, b.Form)
	}
	if !slices.Equal(e.Tags, []string{"scifi"}) {
		t.Errorf("Tags = %v, want only genre tags", e.Tags)
	}
	if e.Body != "Somehow it ended up on my list.\n" {
		t.Errorf("Body = %q", e.Body)
	}
}

func TestConvertBookDateFormats(t *testing.T) {
	for _, d := range []string{
		"'2026-10-19T00:00:00+00:00'",
		"'2026-10-19T08:30:00Z'",
		"2026-10-19T08:30:00-07:00",
		"  2026-10-19 08:30:00",
	} {
		src := "---\ntitle: X\ndate: " + d + "\ncategories: book\ntags: [book, nonfiction, good]\npagecount: 1\n---\n"
		e, err := convert("books/x.md", []byte(src), testTargets)
		if err != nil {
			t.Fatalf("%s: %v", d, err)
		}
		if got := e.Book.Finished.String(); got != "2026-10-19" {
			t.Errorf("%s: finished %s", d, got)
		}
	}
}

func TestConvertBookFormOverride(t *testing.T) {
	src := "---\ntitle: Say Nothing\ndate: 2020-05-31\ncategories: book\ntags: [book, great, history]\npagecount: 1\n---\n"
	e, err := convert("books/say-nothing.md", []byte(src), testTargets)
	if err != nil {
		t.Fatal(err)
	}
	if e.Book.Form != content.FormNonfiction {
		t.Errorf("Form = %q", e.Book.Form)
	}
}

func TestConvertPost(t *testing.T) {
	src := `---
layout: post
title: "What's in a type?"
date: '2014-05-17T17:53:27Z'
categories: code
tags: ["go", "code", "ruby"]
excerpt: Rambling about types.
published: false
---
Body
`
	e, err := convert("posts/whats-in-a-type.md", []byte(src), testTargets)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if e.Kind != content.KindPost || e.Title != "What's in a type?" || e.Date.String() != "2014-05-17" {
		t.Errorf("got %+v", e)
	}
	if e.Summary != "Rambling about types." || !e.Draft {
		t.Errorf("summary %q draft %v", e.Summary, e.Draft)
	}
	if !slices.Equal(e.Tags, []string{"code", "go", "ruby"}) {
		t.Errorf("Tags = %v, want category first then tags, deduped", e.Tags)
	}
}

func TestRewriteInlineLinks(t *testing.T) {
	body := "Like [*Annihilation*](/blog/2018-01-18-annihilation/) and [*The Great\nLeveler*](/blog/2018-02-26-the-great-leveler/),\n" +
		"not [*Bebe*](/blog/2018-06-29-bringing-up-bebe/) or [last year](/blog/2019-01-01-another-year-in-review/). See [here](/reading/).\n"
	got, err := rewriteBody(body, testTargets)
	if err != nil {
		t.Fatal(err)
	}
	want := "Like [[annihilation]] and [[the-great-leveler]],\n" +
		"not [*Bebe*](/books/bringing-up-bebe/) or [last year](/posts/another-year-in-review/). See [here](/reading/).\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestRewriteReferenceLinks(t *testing.T) {
	body := "Read [*Submergence*][sub-me], then more.\n\n[other]: https://example.com\n[sub-me]:             http://andrewledvina.com/blog/2015-12-14-submergence/\n"
	got, err := rewriteBody(body, testTargets)
	if err != nil {
		t.Fatal(err)
	}
	want := "Read [[submergence]], then more.\n\n[other]: https://example.com\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestRewriteUnknownTargetFails(t *testing.T) {
	_, err := rewriteBody("[x](/blog/2018-01-01-missing/)", testTargets)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("err = %v", err)
	}
}

func TestRewriteImagePaths(t *testing.T) {
	got, err := rewriteBody("![Flow Diagram](images/flow_simple.jpg)\n", testTargets)
	if err != nil {
		t.Fatal(err)
	}
	if got != "![Flow Diagram](/images/flow_simple.jpg)\n" {
		t.Errorf("got %q", got)
	}
}
