package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/content"
)

type fakeResolver map[string]string

func (f fakeResolver) Resolve(_ context.Context, short string) (string, error) {
	if long, ok := f[short]; ok {
		return long, nil
	}
	return "", errors.New("no redirect")
}

func TestFindStoreLink(t *testing.T) {
	body := "Read [*Ice*][ice] and [this review](http://www.slate.com/review).\n\n" +
		"[slate]: http://www.slate.com/x\n[ice]:     http://amzn.com/0143105981\n[other]: https://www.amazon.com/dp/B000000000\n"
	if got := findStoreLink(body); got != "http://amzn.com/0143105981" {
		t.Errorf("findStoreLink = %q, want the first Amazon link", got)
	}
	// A link to an Amazon page that isn't a product is passed over.
	body = "[kf]: http://www.amazon.com/gp/digital/kindle/first\n[moon]: http://amzn.com/B00ZX99WE8\n"
	if got := findStoreLink(body); got != "http://amzn.com/B00ZX99WE8" {
		t.Errorf("findStoreLink = %q, want the product link", got)
	}
	if got := findStoreLink("no links here"); got != "" {
		t.Errorf("findStoreLink = %q", got)
	}
}

func TestNormalizeStoreLink(t *testing.T) {
	r := fakeResolver{
		"http://a.co/9Ua6Xkj":     "https://www.amazon.com/dp/B004YW4L5K/ref=as_li_ss_tl?_encoding=UTF8&tag=rokob-20",
		"https://amzn.to/2JCdSjE": "https://www.amazon.com/gp/product/0062316117/ref=x?tag=rokob-20",
	}
	cases := map[string]string{
		"http://amzn.com/0143105981":                                                       "https://www.amazon.com/dp/0143105981",
		"https://amzn.com/B00ICN066A":                                                      "https://www.amazon.com/dp/B00ICN066A",
		"http://smile.amazon.com/gp/product/B00ABC1234/ref=x?tag=y":                        "https://www.amazon.com/dp/B00ABC1234",
		"https://www.amazon.com/Stand-Stephen-King-ebook/dp/B008J1KZ3Q/ref=sr_1_1?s=books": "https://www.amazon.com/dp/B008J1KZ3Q",
		"https://www.amazon.com/dp/B003XT605Y":                                             "https://www.amazon.com/dp/B003XT605Y",
		"http://a.co/9Ua6Xkj":                                                              "https://www.amazon.com/dp/B004YW4L5K",
		"https://amzn.to/2JCdSjE":                                                          "https://www.amazon.com/dp/0062316117",
		"https://www.amazon.com/some-list-page?x=1":                                        "https://www.amazon.com/some-list-page",
	}
	for in, want := range cases {
		got, err := normalizeStoreLink(context.Background(), r, in)
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("normalize(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestBackfillLinks(t *testing.T) {
	files := map[string][]byte{
		"books/ice.md":     []byte("---\ntitle: Ice\nfinished: 2019-02-10\n---\n\n[*Ice*][ice] is odd.\n\n[ice]: http://amzn.com/0143105981\n"),
		"books/linked.md":  []byte("---\ntitle: Linked\nlink: https://openlibrary.org/works/OL1W\nfinished: 2019-02-10\n---\n\n[x]: http://amzn.com/0000000000\n"),
		"books/short.md":   []byte("---\ntitle: Short\nfinished: 2019-02-10\n---\n\n[s]: http://a.co/broken\n"),
		"books/nothing.md": []byte("---\ntitle: Nothing\nfinished: 2019-02-10\n---\n\nNo link.\n"),
	}
	out, report, err := backfillLinks(context.Background(), fakeResolver{}, files)
	if err != nil {
		t.Fatal(err)
	}
	ice, err := content.Parse("books/ice.md", out["books/ice.md"])
	if err != nil {
		t.Fatal(err)
	}
	if ice.Book.Link != "https://www.amazon.com/dp/0143105981" || !strings.Contains(ice.Body, "[ice]: http://amzn.com/0143105981") {
		t.Errorf("ice = %+v", ice)
	}
	if _, changed := out["books/linked.md"]; changed {
		t.Error("a book that already has a link was changed")
	}
	// An unresolvable short link is still a working link, so keep it as https.
	short, _ := content.Parse("books/short.md", out["books/short.md"])
	if short.Book.Link != "https://a.co/broken" {
		t.Errorf("short link = %q", short.Book.Link)
	}
	joined := strings.Join(report, "\n")
	if !strings.Contains(joined, "short") || !strings.Contains(joined, "nothing") {
		t.Errorf("report should mention the unresolved and missing links:\n%s", joined)
	}
}
