package render

import (
	"html/template"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/history"
	"github.com/rokob/ryuguma/internal/site"
)

func fixtureSite(t *testing.T) *site.Site {
	t.Helper()
	entries := []content.Entry{
		{Kind: content.KindBook, Slug: "dune", Path: "books/dune.md", Title: "Dune", Tags: []string{"scifi"},
			Date: content.MustDate("2015-08-03"),
			Book: &content.Book{Authors: []string{"Frank Herbert"}, Link: "https://www.amazon.com/dp/0441013597", Finished: content.MustDate("2015-08-03"), Pages: 896, Rating: content.RatingGreat}},
		{Kind: content.KindBook, Slug: "james", Path: "books/james.md", Title: "James",
			Date: content.MustDate("2026-05-01"),
			Book: &content.Book{Authors: []string{"Percival Everett"}, Started: content.MustDate("2026-05-01")}},
		{Kind: content.KindBook, Slug: "someday", Path: "books/someday.md", Title: "Someday",
			Date: content.MustDate("2026-02-01"), Book: &content.Book{Added: content.MustDate("2026-02-01")}},
		{Kind: content.KindBook, Slug: "red-rising", Path: "books/red-rising.md", Title: "Red Rising", Date: content.MustDate("2019-01-01"),
			Book: &content.Book{Series: "Red Rising", Number: 1, Finished: content.MustDate("2019-01-01")}},
		{Kind: content.KindBook, Slug: "golden-son", Path: "books/golden-son.md", Title: "Golden Son", Date: content.MustDate("2019-02-01"),
			Book: &content.Book{Series: "Red Rising", Number: 2, Finished: content.MustDate("2019-02-01")}},
		{Kind: content.KindPost, Slug: "hello", Path: "posts/hello.md", Title: "Hello", Date: content.MustDate("2014-01-01"),
			Tags: []string{"life"}, Body: "First I read [[dune]].\n"},
		{Kind: content.KindAtom, Slug: "idea", Path: "atoms/idea.md", Title: "An idea", Date: content.MustDate("2026-06-01"), Body: "A small idea.\n"},
		{Kind: content.KindPage, Slug: "home", Path: "pages/home.md", Title: "Home", Body: "Hi, I write here.\n"},
		{Kind: content.KindPage, Slug: "about", Path: "pages/about.md", Title: "About", Body: "About me.\n"},
		{Kind: content.KindPage, Slug: "reading", Path: "pages/reading.md", Title: "Reading", Body: "Why I track books.\n"},
	}
	revs := map[string][]history.Revision{"books/dune.md": {
		{Commit: "c0ffee1234567890", Date: time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), Subject: "chore(content): migrate", Path: "content/books/dune.md"},
		{Commit: "aaaa1111bbbb2222", Date: time.Date(2015, 8, 3, 0, 0, 0, 0, time.UTC), Subject: "post about dune", Path: "_posts/2015-08-03-dune.md"},
	}}
	s, err := site.Build(entries, site.Options{History: func(p string) []history.Revision { return revs[p] }})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var cfg = Config{Title: "ryuguma", URL: "https://ryuguma.com", Repo: "https://github.com/rokob/ryuguma"}

func TestSiteWritesEveryPage(t *testing.T) {
	out := MemOutput{}
	if err := Site(fixtureSite(t), cfg, os.DirFS("../../templates"), out); err != nil {
		t.Fatalf("Site: %v", err)
	}
	want := []string{
		"404.html", "about/index.html", "atoms/idea/index.html", "atoms/index.html",
		"books/dune/history/index.html", "books/dune/index.html", "books/golden-son/index.html", "books/index.html", "books/james/index.html", "books/red-rising/index.html", "books/someday/index.html",
		"index.html", "posts/hello/index.html", "posts/index.html", "reading/index.html",
		"tags/index.html", "tags/life/index.html", "tags/scifi/index.html",
	}
	var got []string
	for name := range out {
		got = append(got, name)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("files =\n%v\nwant\n%v", got, want)
	}
}

func TestPagesShowTheirContent(t *testing.T) {
	out := MemOutput{}
	if err := Site(fixtureSite(t), cfg, os.DirFS("../../templates"), out); err != nil {
		t.Fatalf("Site: %v", err)
	}
	checks := map[string][]string{
		"books/dune/index.html": {"<title>Dune · ryuguma</title>", "<cite>Dune</cite>", "Frank Herbert", "896", "Referenced by", `href="/posts/hello/"`,
			`href="https://github.com/rokob/ryuguma/commit/c0ffee1234567890"><code>c0ffee12</code>`, `href="/books/dune/history/"`, `<a href="https://www.amazon.com/dp/0441013597">amazon.com</a>`},
		"books/dune/history/index.html": {"post about dune", `href="https://github.com/rokob/ryuguma/blob/aaaa1111bbbb2222/_posts/2015-08-03-dune.md"`, "Aug 3, 2015"},
		"books/james/index.html":        {"reading now", "Started", `<link rel="canonical" href="https://ryuguma.com/books/james/">`},
		"index.html":                    {"Hi, I write here.", "Reading now", "James", `href="/books/dune/"`, "A small idea."},
		"books/index.html":              {"Up next", "Someday", "2015", "1 book"},
		"posts/hello/index.html":        {`<a href="/books/dune/"><cite>Dune</cite></a>`, "#life"},
		"tags/scifi/index.html":         {"#scifi", "Dune"},
	}
	for name, wants := range checks {
		html := string(out[name])
		for _, w := range wants {
			if !strings.Contains(html, w) {
				t.Errorf("%s: missing %q", name, w)
			}
		}
	}
}

func TestTemplateErrorsNameTheFile(t *testing.T) {
	tmpl := fstest.MapFS{
		"_layout.html": {Data: []byte(`{{define "base"}}{{block "main" .}}{{end}}{{end}}`)},
		"home.html":    {Data: []byte(`{{define "main"}}{{.Nope}}{{end}}`)},
	}
	err := Site(fixtureSite(t), cfg, tmpl, MemOutput{})
	if err == nil || !strings.Contains(err.Error(), "index.html") {
		t.Errorf("err = %v", err)
	}
}

func TestCopy(t *testing.T) {
	src := fstest.MapFS{"style.css": {Data: []byte("a{}")}, "img/x.png": {Data: []byte("png")}, ".DS_Store": {Data: []byte("junk")}}
	out := MemOutput{}
	if err := Copy(out, "static", src); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || string(out["static/style.css"]) != "a{}" || out["static/img/x.png"] == nil {
		t.Errorf("out = %v", out)
	}
}

func TestExcerpt(t *testing.T) {
	body := template.HTML("<h2>Skip me</h2><p>First <em>paragraph</em> &amp; more words here.</p><p>Second.</p>")
	if got := excerpt(100, body); got != "First paragraph & more words here." {
		t.Errorf("excerpt = %q", got)
	}
	if got := excerpt(18, body); got != "First paragraph &…" {
		t.Errorf("short excerpt = %q", got)
	}
	if got := excerpt(10, "plain summary text"); got != "plain…" {
		t.Errorf("string excerpt = %q", got)
	}
}

func TestAuthors(t *testing.T) {
	cases := map[string][]string{
		"":            nil,
		"A":           {"A"},
		"A and B":     {"A", "B"},
		"A, B, and C": {"A", "B", "C"},
	}
	for want, names := range cases {
		if got := authors(names); got != want {
			t.Errorf("authors(%v) = %q, want %q", names, got, want)
		}
	}
}

func TestComma(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 75191: "75,191", 1234567: "1,234,567", -4200: "-4,200"} {
		if got := comma(n); got != want {
			t.Errorf("comma(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestHeatBin(t *testing.T) {
	for n, want := range map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 3, 5: 4, 9: 4} {
		if got := heatBin(n); got != want {
			t.Errorf("heatBin(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestPips(t *testing.T) {
	pips := funcs["pips"].(func(content.Rating) []bool)
	for r, want := range map[content.Rating][]bool{
		content.RatingBad:   {true, false, false, false},
		content.RatingGood:  {true, true, true, false},
		content.RatingGreat: {true, true, true, true},
	} {
		if got := pips(r); !slices.Equal(got, want) {
			t.Errorf("pips(%s) = %v, want %v", r, got, want)
		}
	}
}

func TestBookPageShowsItsSeries(t *testing.T) {
	out := MemOutput{}
	if err := Site(fixtureSite(t), cfg, os.DirFS("../../templates"), out); err != nil {
		t.Fatalf("Site: %v", err)
	}
	html := string(out["books/golden-son/index.html"])
	for _, want := range []string{
		"<dt>Series</dt><dd>Red Rising, book 2</dd>",
		"Red Rising series",
		`<a href="/books/red-rising/"><cite>Red Rising</cite></a>`,
		"read Jan 2019",
		`aria-current="page"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("golden-son: missing %q", want)
		}
	}
	if strings.Contains(html, `href="/books/golden-son/"`) {
		t.Error("golden-son links to itself in its series")
	}
	if dune := string(out["books/dune/index.html"]); strings.Contains(dune, "series") {
		t.Error("dune is not in a series but mentions one")
	}
}
