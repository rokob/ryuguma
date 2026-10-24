package markdown_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/markdown"
)

type fakeResolver map[string]markdown.Target

func (f fakeResolver) Resolve(slug string) (markdown.Target, bool) {
	t, ok := f[slug]
	return t, ok
}

var targets = fakeResolver{
	"dune":           {Slug: "dune", URL: "/books/dune/", Title: "Dune", Cite: true},
	"flow":           {Slug: "flow", URL: "/books/flow/", Title: "Flow", Cite: true},
	"year-in-review": {Slug: "year-in-review", URL: "/posts/year-in-review/", Title: "Year in Review"},
}

func render(t *testing.T, src string) markdown.Result {
	t.Helper()
	r := markdown.New(targets)
	res, err := r.Render([]byte(src))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return res
}

func TestWikiLinkUsesTargetTitle(t *testing.T) {
	res := render(t, "I loved [[dune]] and [[year-in-review]].")
	html := string(res.HTML)
	if !strings.Contains(html, `<a href="/books/dune/"><cite>Dune</cite></a>`) {
		t.Errorf("book link missing cite: %s", html)
	}
	if !strings.Contains(html, `<a href="/posts/year-in-review/">Year in Review</a>`) {
		t.Errorf("post link: %s", html)
	}
	if !slices.Equal(res.Refs, []string{"dune", "year-in-review"}) {
		t.Errorf("Refs = %v", res.Refs)
	}
}

func TestWikiLinkCustomLabel(t *testing.T) {
	res := render(t, "Read [[dune|that spice book]].")
	if !strings.Contains(string(res.HTML), `<a href="/books/dune/">that spice book</a>`) {
		t.Errorf("got %s", res.HTML)
	}
}

func TestInternalMarkdownLinksAreRefs(t *testing.T) {
	res := render(t, "See [*the first time*](/books/flow/) and [twice](/books/flow) and [x](https://example.com/books/dune/).")
	if !slices.Equal(res.Refs, []string{"flow"}) {
		t.Errorf("Refs = %v, want deduped internal refs only", res.Refs)
	}
}

func TestBrokenRefsAreErrors(t *testing.T) {
	r := markdown.New(targets)
	_, err := r.Render([]byte("[[nope]] and [x](/posts/missing/)"))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"nope", "/posts/missing/"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestBracketsInCodeAndNonSlugsAreLeftAlone(t *testing.T) {
	src := "```haskell\nf :: [[Integer]]\n```\n\nInline `[[SomeClass alloc] init]` and prose [[Not A Slug]] stay.\n"
	res := render(t, src)
	html := string(res.HTML)
	for _, want := range []string{"[[Integer]]", "[[SomeClass alloc] init]", "[[Not A Slug]]"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in %s", want, html)
		}
	}
	if len(res.Refs) != 0 {
		t.Errorf("Refs = %v", res.Refs)
	}
}

func TestReferenceStyleLinksAndRawHTMLStillWork(t *testing.T) {
	src := "[*Dune*][amz] by Herbert.\n\n<div class=\"x\">raw</div>\n\n[amz]: https://example.com/dune\n"
	html := string(render(t, src).HTML)
	if !strings.Contains(html, `<a href="https://example.com/dune"><em>Dune</em></a>`) {
		t.Errorf("reference link: %s", html)
	}
	if !strings.Contains(html, `<div class="x">raw</div>`) {
		t.Errorf("raw html: %s", html)
	}
}
