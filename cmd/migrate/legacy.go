package main

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/rokob/ryuguma/internal/content"
)

// legacyFrontmatter is what the Jekyll/Gatsby site used.
type legacyFrontmatter struct {
	Layout     string   `yaml:"layout"`
	Title      string   `yaml:"title"`
	Date       string   `yaml:"date"`
	Categories string   `yaml:"categories"`
	Tags       []string `yaml:"tags"`
	Pagecount  int      `yaml:"pagecount"`
	Published  *bool    `yaml:"published"`
	Excerpt    string   `yaml:"excerpt"`
}

// formOverrides fills in books that were never tagged fiction or nonfiction.
var formOverrides = map[string]content.Form{
	"say-nothing": content.FormNonfiction,
}

func parseLegacy(data []byte) (legacyFrontmatter, string, error) {
	var fm legacyFrontmatter
	head, body, err := content.SplitFrontmatter(data)
	if err != nil {
		return fm, "", err
	}
	dec := yaml.NewDecoder(bytes.NewReader(head))
	dec.KnownFields(true)
	if err := dec.Decode(&fm); err != nil {
		return fm, "", err
	}
	return fm, body, nil
}

// convert turns a legacy file into a new-format entry. titles maps every
// slug to its title so internal links can be checked and simplified.
func convert(p string, data []byte, targets map[string]target) (content.Entry, error) {
	fm, body, err := parseLegacy(data)
	if err != nil {
		return content.Entry{}, fmt.Errorf("%s: %w", p, err)
	}
	date, err := legacyDate(fm.Date)
	if err != nil {
		return content.Entry{}, fmt.Errorf("%s: %w", p, err)
	}
	body, err = rewriteBody(body, targets)
	if err != nil {
		return content.Entry{}, fmt.Errorf("%s: %w", p, err)
	}

	dir, file := path.Split(p)
	slug := strings.TrimSuffix(file, ".md")
	e := content.Entry{
		Slug:  slug,
		Path:  p,
		Title: fm.Title,
		Draft: fm.Published != nil && !*fm.Published,
		Body:  body,
	}

	tags := cleanTags(fm.Tags)
	switch dir {
	case "books/":
		e.Kind = content.KindBook
		b := &content.Book{Finished: date, Pages: fm.Pagecount, Form: formOverrides[slug]}
		for _, t := range tags {
			switch {
			case slices.Contains(content.Ratings, content.Rating(t)):
				b.Rating = content.Rating(t)
			case slices.Contains(content.Forms, content.Form(t)):
				b.Form = content.Form(t)
			case t == "book":
			default:
				e.Tags = append(e.Tags, t)
			}
		}
		e.Book = b
		e.Date = date
	case "posts/":
		e.Kind = content.KindPost
		e.Date = date
		e.Summary = fm.Excerpt
		e.Tags = cleanTags(append(strings.Fields(fm.Categories), tags...))
	default:
		return content.Entry{}, fmt.Errorf("%s: unexpected directory", p)
	}
	return e, nil
}

// Legacy dates came in several formats; only the calendar date matters now.
func legacyDate(s string) (content.Date, error) {
	s = strings.TrimSpace(s)
	if len(s) < len("2006-01-02") {
		return content.Date{}, fmt.Errorf("bad date %q", s)
	}
	return content.ParseDate(s[:len("2006-01-02")])
}

func cleanTags(tags []string) []string {
	var out []string
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

var (
	oldPostPath = `/blog/\d{4}-\d{2}-\d{2}-([a-z0-9-]+)/?`
	inlineLink  = regexp.MustCompile(`\[([^\]]*)\]\(` + oldPostPath + `\)`)
	refDef      = regexp.MustCompile(`(?m)^\[([^\]]+)\]:[ \t]*(?:https?://[^/\s]+)?` + oldPostPath + `[ \t]*\n?`)
)

type target struct {
	Title string
	Dir   string
}

// rewriteBody converts links to old /blog/ URLs and makes image paths
// absolute. A link whose label is just the target's title becomes [[slug]];
// any other label is kept as a normal link to the new URL.
func rewriteBody(body string, targets map[string]target) (string, error) {
	var errs []error
	wiki := func(label, slug string) string {
		t, ok := targets[slug]
		if !ok {
			errs = append(errs, fmt.Errorf("link to unknown post %q", slug))
			return label
		}
		plain := strings.Join(strings.Fields(strings.Trim(label, "*_")), " ")
		if strings.EqualFold(plain, t.Title) {
			return "[[" + slug + "]]"
		}
		return "[" + label + "](/" + t.Dir + "/" + slug + "/)"
	}

	body = inlineLink.ReplaceAllStringFunc(body, func(m string) string {
		sub := inlineLink.FindStringSubmatch(m)
		return wiki(sub[1], sub[2])
	})

	refs := map[string]string{}
	body = refDef.ReplaceAllStringFunc(body, func(m string) string {
		sub := refDef.FindStringSubmatch(m)
		refs[strings.ToLower(sub[1])] = sub[2]
		return ""
	})
	for ref, slug := range refs {
		usage := regexp.MustCompile(`\[([^\]]*)\]\[(?i:` + regexp.QuoteMeta(ref) + `)\]`)
		body = usage.ReplaceAllStringFunc(body, func(m string) string {
			return wiki(usage.FindStringSubmatch(m)[1], slug)
		})
	}

	body = strings.ReplaceAll(body, "](images/", "](/images/")
	return body, errors.Join(errs...)
}
