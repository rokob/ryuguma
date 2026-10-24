package render

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
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
	// host names the site a link goes to, like "openlibrary.org".
	"host": func(link string) string {
		u, err := url.Parse(link)
		if err != nil {
			return link
		}
		return strings.TrimPrefix(u.Hostname(), "www.")
	},
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
	"readingStats": site.Reading,
	"comma":        comma,
	"frac":         frac,
	"percent": func(n, total int) string {
		if total == 0 {
			return "–"
		}
		return fmt.Sprintf("%d%%", (100*n+total/2)/total)
	},
	"bin":         heatBin,
	"ratingNames": func() []content.Rating { return content.Ratings },
	// pips marks how many of the rating levels a rating fills: great fills all.
	"pips": func(r content.Rating) []bool {
		n := slices.Index(content.Ratings, r) + 1
		pips := make([]bool, len(content.Ratings))
		for i := range n {
			pips[i] = true
		}
		return pips
	},
	"monthNames": func() []string {
		var names []string
		for m := time.January; m <= time.December; m++ {
			names = append(names, m.String())
		}
		return names
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

// comma formats 63481 as "63,481".
func comma(n int) string {
	s := fmt.Sprint(n)
	if n < 0 {
		return "-" + comma(-n)
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// frac is n/max for CSS, as a string so templates can put it in a style.
func frac(n, max int) string {
	if max == 0 {
		return "0"
	}
	return fmt.Sprintf("%.3f", float64(n)/float64(max))
}

// heatBin buckets a month's book count for the heatmap: 0, 1, 2, 3–4, 5+.
func heatBin(n int) int {
	switch {
	case n >= 5:
		return 4
	case n >= 3:
		return 3
	default:
		return n
	}
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
