package content

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Encode renders an entry as a content file: frontmatter followed by the body.
func Encode(e Entry) ([]byte, error) {
	var fm any
	switch e.Kind {
	case KindBook:
		b := e.Book
		if b == nil {
			b = &Book{}
		}
		fm = bookFrontmatter{
			Title:    e.Title,
			Author:   b.Authors,
			ISBN:     b.ISBN,
			Added:    b.Added,
			Started:  b.Started,
			Finished: b.Finished,
			Pages:    b.Pages,
			Rating:   b.Rating,
			Form:     b.Form,
			Tags:     e.Tags,
			Draft:    e.Draft,
		}
	case KindPost:
		fm = postFrontmatter{Title: e.Title, Date: e.Date, Summary: e.Summary, Tags: e.Tags, Draft: e.Draft}
	case KindAtom:
		fm = atomFrontmatter{Title: e.Title, Date: e.Date, Tags: e.Tags, Draft: e.Draft}
	case KindPage:
		fm = pageFrontmatter{Title: e.Title}
	default:
		return nil, fmt.Errorf("unknown kind %q", e.Kind)
	}

	var buf bytes.Buffer
	buf.WriteString(delim + "\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fm); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString(delim + "\n")
	if e.Body != "" {
		buf.WriteString("\n" + e.Body)
	}
	return buf.Bytes(), nil
}
