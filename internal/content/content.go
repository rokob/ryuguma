// Package content defines the on-disk content model: markdown files with YAML
// frontmatter under books/, posts/, and atoms/.
package content

import (
	"fmt"
	"time"

	"go.yaml.in/yaml/v3"
)

type Kind string

const (
	KindBook Kind = "book"
	KindPost Kind = "post"
	KindAtom Kind = "atom"
	// KindPage is a standalone page like /about/, outside the dated stream.
	KindPage Kind = "page"
)

// Dirs maps each content directory to the kind of entry it holds.
var Dirs = map[string]Kind{
	"books": KindBook,
	"posts": KindPost,
	"atoms": KindAtom,
	"pages": KindPage,
}

// DirFor returns the content directory for a kind.
func DirFor(k Kind) string {
	for d, kind := range Dirs {
		if kind == k {
			return d
		}
	}
	panic(fmt.Sprintf("unknown kind %q", k))
}

// Entry is one piece of content of any kind.
type Entry struct {
	Kind Kind
	Slug string
	// Path is relative to the content root, e.g. "books/dune.md".
	Path  string
	Title string
	// Date is when the entry happened: the publish date of a post or atom, or
	// the most recent lifecycle date of a book.
	Date    Date
	Tags    []string
	Draft   bool
	Summary string
	Body    string
	// Book is set only when Kind is KindBook.
	Book *Book
}

type Book struct {
	Authors []string
	// Series names the series the book belongs to, and Number is its place in
	// it. Number is optional, and can be fractional for in-between novellas.
	Series string
	Number float64
	// Link points at a canonical page for the book, like its Open Library
	// or store page.
	Link     string
	ISBN     string
	Added    Date
	Started  Date
	Finished Date
	Pages    int
	Rating   Rating
	Form     Form
}

type Status string

const (
	StatusToRead  Status = "to-read"
	StatusReading Status = "reading"
	StatusRead    Status = "read"
)

// Status is derived from which lifecycle dates are set.
func (b *Book) Status() Status {
	switch {
	case !b.Finished.IsZero():
		return StatusRead
	case !b.Started.IsZero():
		return StatusReading
	default:
		return StatusToRead
	}
}

func (b *Book) latestDate() Date {
	switch {
	case !b.Finished.IsZero():
		return b.Finished
	case !b.Started.IsZero():
		return b.Started
	default:
		return b.Added
	}
}

type Rating string

const (
	RatingBad   Rating = "bad"
	RatingOkay  Rating = "okay"
	RatingGood  Rating = "good"
	RatingGreat Rating = "great"
)

// Ratings lists every rating from worst to best.
var Ratings = []Rating{RatingBad, RatingOkay, RatingGood, RatingGreat}

type Form string

const (
	FormFiction    Form = "fiction"
	FormNonfiction Form = "nonfiction"
)

var Forms = []Form{FormFiction, FormNonfiction}

// Date is a calendar date, optionally with a time of day. It is written as
// 2006-01-02 or as RFC 3339 when the time matters.
type Date struct {
	t       time.Time
	hasTime bool
}

func ParseDate(s string) (Date, error) {
	if t, err := time.Parse(time.DateOnly, s); err == nil {
		return Date{t: t}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return Date{t: t.UTC(), hasTime: true}, nil
	}
	return Date{}, fmt.Errorf("invalid date %q (want 2006-01-02 or RFC 3339)", s)
}

// MustDate is ParseDate for literals; it panics on bad input.
func MustDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

// DateOf returns the calendar date of t, dropping the time of day.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{t: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

func (d Date) Time() time.Time        { return d.t }
func (d Date) IsZero() bool           { return d.t.IsZero() }
func (d Date) Before(o Date) bool     { return d.t.Before(o.t) }
func (d Date) Format(l string) string { return d.t.Format(l) }

func (d Date) String() string {
	if d.hasTime {
		return d.t.Format(time.RFC3339)
	}
	return d.t.Format(time.DateOnly)
}

func (d *Date) UnmarshalYAML(n *yaml.Node) error {
	parsed, err := ParseDate(n.Value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Date) MarshalYAML() (any, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: d.String()}, nil
}

// names accepts either a single string or a list in YAML.
type names []string

func (n *names) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		*n = names{node.Value}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err != nil {
		return err
	}
	*n = list
	return nil
}

func (n names) MarshalYAML() (any, error) {
	if len(n) == 1 {
		return n[0], nil
	}
	node := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	for _, s := range n {
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: s})
	}
	return node, nil
}
