package render

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/site"
)

var funcs = template.FuncMap{
	"date":      func(d content.Date) string { return d.Format("Jan 2, 2006") },
	"monthDay":  func(d content.Date) string { return d.Format("Jan 2") },
	"isodate":   func(d content.Date) string { return d.Format("2006-01-02") },
	"byYear":    byYear,
	"authors":   authors,
	"take":      take,
	"excerpt":   excerpt,
	"hasPrefix": strings.HasPrefix,
	"short":     func(sha string) string { return sha[:min(8, len(sha))] },
	"capitalize": func(v any) string {
		s := fmt.Sprint(v)
		if s == "" {
			return s
		}
		r, n := utf8.DecodeRuneInString(s)
		return strings.ToUpper(string(r)) + s[n:]
	},
	"plural": func(n int, one, many string) string {
		if n == 1 {
			return one
		}
		return many
	},
	"pageCount": func(pages []*site.Page) int {
		total := 0
		for _, p := range pages {
			if p.Book != nil {
				total += p.Book.Pages
			}
		}
		return total
	},
}

type YearGroup struct {
	Year  int
	Pages []*site.Page
}

// byYear groups pages, which are already sorted newest first, by year.
func byYear(pages []*site.Page) []YearGroup {
	var groups []YearGroup
	for _, p := range pages {
		y := p.Date.Time().Year()
		if len(groups) == 0 || groups[len(groups)-1].Year != y {
			groups = append(groups, YearGroup{Year: y})
		}
		g := &groups[len(groups)-1]
		g.Pages = append(g.Pages, p)
	}
	return groups
}

// authors joins names as "A", "A and B", or "A, B, and C".
func authors(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}

func take(n int, pages []*site.Page) []*site.Page {
	if len(pages) > n {
		return pages[:n]
	}
	return pages
}

var (
	firstParagraph = regexp.MustCompile(`(?s)<p>(.*?)</p>`)
	tags           = regexp.MustCompile(`<[^>]*>`)
)

// excerpt shortens plain text, or the first paragraph of rendered HTML, to
// a word boundary near max characters.
func excerpt(max int, v any) string {
	var text string
	switch v := v.(type) {
	case template.HTML:
		m := firstParagraph.FindStringSubmatch(string(v))
		if m == nil {
			return ""
		}
		text = html.UnescapeString(tags.ReplaceAllString(m[1], ""))
	default:
		text = fmt.Sprint(v)
	}
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	cut := string([]rune(text)[:max])
	if i := strings.LastIndexByte(cut, ' '); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, ",.;:") + "…"
}
