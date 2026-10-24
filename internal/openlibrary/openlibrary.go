// Package openlibrary looks up book metadata from openlibrary.org.
package openlibrary

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
)

const DefaultBaseURL = "https://openlibrary.org"

type Client struct {
	BaseURL   string
	HTTP      *http.Client
	UserAgent string
}

type Doc struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	Authors          []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	// Pages is the median across editions, so a given edition may differ.
	Pages    int `json:"number_of_pages_median"`
	Editions int `json:"edition_count"`
}

// Search returns works matching a title, most relevant first.
func (c *Client) Search(ctx context.Context, title string) ([]Doc, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	q := url.Values{
		"title":  {title},
		"fields": {"key,title,author_name,first_publish_year,number_of_pages_median,edition_count"},
		"limit":  {"20"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/search.json?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open library search %q: %s", title, resp.Status)
	}
	var body struct {
		Docs []Doc `json:"docs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("open library search %q: %w", title, err)
	}
	return body.Docs, nil
}

type Match int

const (
	MatchNone     Match = iota // no candidate with authors
	MatchGuess                 // most relevant result, title differs
	MatchSubtitle              // same title once any ": subtitle" is dropped
	MatchExact                 // titles are equal after normalizing
)

func (m Match) String() string {
	return [...]string{"none", "guess", "subtitle", "exact"}[m]
}

// titleMatch says how well a result's title matches the one searched for.
func titleMatch(got, want string) Match {
	switch {
	case normalize(got) == normalize(want):
		return MatchExact
	case normalize(mainTitle(got)) == normalize(mainTitle(want)):
		return MatchSubtitle
	default:
		return MatchGuess
	}
}

// Rank orders results by how well their titles match, keeping Open
// Library's relevance order among equally good matches.
func Rank(docs []Doc, title string) []Doc {
	ranked := slices.Clone(docs)
	slices.SortStableFunc(ranked, func(a, b Doc) int {
		return cmp.Compare(titleMatch(b.Title, title), titleMatch(a.Title, title))
	})
	return ranked
}

// Best picks the candidate that most likely is the book with this title.
// Among equally good title matches, the one closest to pages wins.
func Best(docs []Doc, title string, pages int) (Doc, Match) {
	var (
		best      Doc
		bestMatch = MatchNone
		bestDiff  int
	)
	for _, d := range docs {
		if len(d.Authors) == 0 {
			continue
		}
		m := titleMatch(d.Title, title)
		diff := pageDiff(d.Pages, pages)
		if m > bestMatch || (m == bestMatch && m > MatchGuess && diff < bestDiff) {
			best, bestMatch, bestDiff = d, m, diff
		}
	}
	return best, bestMatch
}

func pageDiff(a, b int) int {
	if a == 0 || b == 0 {
		return 1 << 30
	}
	if a > b {
		return a - b
	}
	return b - a
}

func mainTitle(s string) string {
	main, _, _ := strings.Cut(s, ":")
	return main
}

// normalize lowercases, drops punctuation, and collapses whitespace.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '\'' || r == '’':
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
