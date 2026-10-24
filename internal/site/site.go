// Package site turns loaded content into the in-memory model the templates
// render: URLs, rendered bodies, references in both directions, and the
// groupings each index page needs.
package site

import (
	"cmp"
	"errors"
	"fmt"
	"html/template"
	"slices"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/markdown"
)

type Options struct {
	// Drafts includes draft entries, for previewing locally.
	Drafts bool
}

// Page is an entry ready to render.
type Page struct {
	content.Entry
	URL       string
	HTML      template.HTML
	Refs      []*Page
	Backlinks []*Page
}

type Tag struct {
	Name  string
	URL   string
	Pages []*Page
}

type Site struct {
	// Pages holds every published entry, newest first.
	Pages []*Page
	// Books holds read books, most recently finished first.
	Books   []*Page
	Reading []*Page
	ToRead  []*Page
	Posts   []*Page
	Atoms   []*Page
	Tags    []Tag

	bySlug map[string]*Page
}

func (s *Site) Lookup(slug string) *Page { return s.bySlug[slug] }

func URLFor(e content.Entry) string {
	return "/" + content.DirFor(e.Kind) + "/" + e.Slug + "/"
}

func Build(entries []content.Entry, opts Options) (*Site, error) {
	s := &Site{bySlug: map[string]*Page{}}
	for _, e := range entries {
		if e.Draft && !opts.Drafts {
			continue
		}
		p := &Page{Entry: e, URL: URLFor(e)}
		s.Pages = append(s.Pages, p)
		s.bySlug[e.Slug] = p
	}
	sortNewestFirst(s.Pages)

	md := markdown.New(resolver{s})
	var errs []error
	for _, p := range s.Pages {
		res, err := md.Render([]byte(p.Body))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Path, err))
		}
		p.HTML = res.HTML
		for _, slug := range res.Refs {
			ref := s.bySlug[slug]
			p.Refs = append(p.Refs, ref)
			ref.Backlinks = append(ref.Backlinks, p)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	tags := map[string]*Tag{}
	for _, p := range s.Pages {
		switch {
		case p.Kind == content.KindPost:
			s.Posts = append(s.Posts, p)
		case p.Kind == content.KindAtom:
			s.Atoms = append(s.Atoms, p)
		case p.Book.Status() == content.StatusRead:
			s.Books = append(s.Books, p)
		case p.Book.Status() == content.StatusReading:
			s.Reading = append(s.Reading, p)
		default:
			s.ToRead = append(s.ToRead, p)
		}
		for _, name := range p.Tags {
			t, ok := tags[name]
			if !ok {
				t = &Tag{Name: name, URL: "/tags/" + name + "/"}
				tags[name] = t
			}
			t.Pages = append(t.Pages, p)
		}
	}
	for _, t := range tags {
		s.Tags = append(s.Tags, *t)
	}
	slices.SortFunc(s.Tags, func(a, b Tag) int { return cmp.Compare(a.Name, b.Name) })
	return s, nil
}

// Pages are processed newest first, so refs and backlinks come out in that
// order too.
func sortNewestFirst(pages []*Page) {
	slices.SortStableFunc(pages, func(a, b *Page) int {
		return cmp.Or(b.Date.Time().Compare(a.Date.Time()), cmp.Compare(a.Slug, b.Slug))
	})
}

type resolver struct{ s *Site }

func (r resolver) Resolve(slug string) (markdown.Target, bool) {
	p, ok := r.s.bySlug[slug]
	if !ok {
		return markdown.Target{}, false
	}
	return markdown.Target{Slug: p.Slug, URL: p.URL, Title: p.DisplayTitle(), Cite: p.Kind == content.KindBook}, true
}

// DisplayTitle falls back to the date for untitled atoms.
func (p *Page) DisplayTitle() string {
	if p.Title != "" {
		return p.Title
	}
	return p.Date.Format("January 2, 2006")
}
