package content

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const delim = "---"

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// The frontmatter structs define the allowed keys for each kind and the order
// they are written in.

type bookFrontmatter struct {
	Title    string   `yaml:"title"`
	Author   names    `yaml:"author,omitempty"`
	ISBN     string   `yaml:"isbn,omitempty"`
	Added    Date     `yaml:"added,omitempty"`
	Started  Date     `yaml:"started,omitempty"`
	Finished Date     `yaml:"finished,omitempty"`
	Pages    int      `yaml:"pages,omitempty"`
	Rating   Rating   `yaml:"rating,omitempty"`
	Form     Form     `yaml:"form,omitempty"`
	Tags     []string `yaml:"tags,omitempty,flow"`
	Draft    bool     `yaml:"draft,omitempty"`
}

type postFrontmatter struct {
	Title   string   `yaml:"title"`
	Date    Date     `yaml:"date"`
	Summary string   `yaml:"summary,omitempty"`
	Tags    []string `yaml:"tags,omitempty,flow"`
	Draft   bool     `yaml:"draft,omitempty"`
}

type atomFrontmatter struct {
	Title string   `yaml:"title"`
	Date  Date     `yaml:"date"`
	Tags  []string `yaml:"tags,omitempty,flow"`
	Draft bool     `yaml:"draft,omitempty"`
}

type pageFrontmatter struct {
	Title string `yaml:"title"`
}

// Parse reads one content file. The path, relative to the content root,
// determines the entry's kind and slug.
func Parse(p string, data []byte) (Entry, error) {
	e, err := parse(p, data)
	if err != nil {
		return Entry{}, fmt.Errorf("%s: %w", p, err)
	}
	return e, nil
}

func parse(p string, data []byte) (Entry, error) {
	dir, file := path.Split(p)
	kind, ok := Dirs[strings.TrimSuffix(dir, "/")]
	if !ok {
		return Entry{}, fmt.Errorf("unknown content directory %q", dir)
	}
	slug := strings.TrimSuffix(file, ".md")
	if !slugPattern.MatchString(slug) {
		return Entry{}, fmt.Errorf("invalid slug %q (want lowercase letters, digits, and dashes)", slug)
	}
	fm, body, err := SplitFrontmatter(data)
	if err != nil {
		return Entry{}, err
	}
	e := Entry{Kind: kind, Slug: slug, Path: p, Body: body}

	switch kind {
	case KindBook:
		var f bookFrontmatter
		if err := decodeStrict(fm, &f); err != nil {
			return Entry{}, err
		}
		e.Title, e.Tags, e.Draft = f.Title, f.Tags, f.Draft
		e.Book = &Book{
			Authors:  f.Author,
			ISBN:     f.ISBN,
			Added:    f.Added,
			Started:  f.Started,
			Finished: f.Finished,
			Pages:    f.Pages,
			Rating:   f.Rating,
			Form:     f.Form,
		}
		e.Date = e.Book.latestDate()
	case KindPost:
		var f postFrontmatter
		if err := decodeStrict(fm, &f); err != nil {
			return Entry{}, err
		}
		e.Title, e.Date, e.Summary, e.Tags, e.Draft = f.Title, f.Date, f.Summary, f.Tags, f.Draft
	case KindAtom:
		var f atomFrontmatter
		if err := decodeStrict(fm, &f); err != nil {
			return Entry{}, err
		}
		e.Title, e.Date, e.Tags, e.Draft = f.Title, f.Date, f.Tags, f.Draft
	case KindPage:
		var f pageFrontmatter
		if err := decodeStrict(fm, &f); err != nil {
			return Entry{}, err
		}
		e.Title = f.Title
	}
	return e, validate(e)
}

// SplitFrontmatter separates the YAML between the leading --- lines from the
// markdown body. Whitespace around the delimiters is ignored.
func SplitFrontmatter(data []byte) (frontmatter []byte, body string, err error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	first, rest, _ := strings.Cut(s, "\n")
	if strings.TrimSpace(first) != delim {
		return nil, "", errors.New("missing frontmatter: file must start with ---")
	}
	var fm strings.Builder
	for rest != "" {
		var line string
		line, rest, _ = strings.Cut(rest, "\n")
		if strings.TrimSpace(line) == delim {
			return []byte(fm.String()), strings.TrimLeft(rest, "\n"), nil
		}
		fm.WriteString(line + "\n")
	}
	return nil, "", errors.New("unterminated frontmatter: no closing ---")
}

func decodeStrict(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func validate(e Entry) error {
	var errs []error
	if e.Title == "" {
		errs = append(errs, errors.New("title is required"))
	}
	if e.Kind == KindAtom && strings.TrimSpace(e.Body) == "" {
		errs = append(errs, errors.New("an atom needs some text below the frontmatter"))
	}
	if e.Date.IsZero() && (e.Kind == KindPost || e.Kind == KindAtom) {
		errs = append(errs, errors.New("date is required"))
	}
	for _, t := range e.Tags {
		if !slugPattern.MatchString(t) {
			errs = append(errs, fmt.Errorf("invalid tag %q (want lowercase letters, digits, and dashes)", t))
		}
	}
	if b := e.Book; b != nil {
		if b.Rating != "" && !slices.Contains(Ratings, b.Rating) {
			errs = append(errs, fmt.Errorf("invalid rating %q (want bad, okay, good, or great)", b.Rating))
		}
		if b.Form != "" && !slices.Contains(Forms, b.Form) {
			errs = append(errs, fmt.Errorf("invalid form %q (want fiction or nonfiction)", b.Form))
		}
		if !b.Started.IsZero() && !b.Finished.IsZero() && b.Finished.Before(b.Started) {
			errs = append(errs, fmt.Errorf("finished %s is before started %s", b.Finished, b.Started))
		}
	}
	return errors.Join(errs...)
}
