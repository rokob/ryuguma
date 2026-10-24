package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/openlibrary"
)

var errNoInput = errors.New("no answer given (input ended)")

// ask prints a prompt and reads a line. A blank answer returns def.
func (e *env) ask(label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(e.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(e.out, "%s: ", label)
	}
	line, err := e.in.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		fmt.Fprintln(e.out)
		return "", errNoInput
	}
	if line = strings.TrimSpace(line); line == "" {
		return def, nil
	}
	return line, nil
}

func (e *env) askRequired(label string) (string, error) {
	for {
		s, err := e.ask(label, "")
		if err != nil || s != "" {
			return s, err
		}
	}
}

// choose asks until the answer is one of allowed.
func (e *env) choose(label string, allowed []string, def string) (string, error) {
	for {
		s, err := e.ask(fmt.Sprintf("%s (%s)", label, strings.Join(allowed, ", ")), def)
		if err != nil {
			return "", err
		}
		if slices.Contains(allowed, strings.ToLower(s)) {
			return strings.ToLower(s), nil
		}
		fmt.Fprintf(e.out, "Pick one of: %s.\n", strings.Join(allowed, ", "))
	}
}

// pick shows a numbered list and returns the chosen index. If none is set,
// 0 is offered with that label and returns -1.
func (e *env) pick(label string, options []string, none string) (int, error) {
	for i, o := range options {
		fmt.Fprintf(e.out, "  %d) %s\n", i+1, o)
	}
	lo := 1
	if none != "" {
		fmt.Fprintf(e.out, "  0) %s\n", none)
		lo = 0
	}
	for {
		s, err := e.ask(label, "")
		if err != nil {
			return 0, err
		}
		if n, err := strconv.Atoi(s); err == nil && n >= lo && n <= len(options) {
			return n - 1, nil
		}
		fmt.Fprintf(e.out, "Enter a number from %d to %d.\n", lo, len(options))
	}
}

func (e *env) entries() ([]content.Entry, error) {
	return content.Load(os.DirFS(e.path("content")))
}

func (e *env) find(slug string) (content.Entry, error) {
	entries, err := e.entries()
	if err != nil {
		return content.Entry{}, err
	}
	for _, en := range entries {
		if en.Slug == slug {
			return en, nil
		}
	}
	return content.Entry{}, fmt.Errorf("no entry with slug %q", slug)
}

func (e *env) save(en content.Entry) error {
	data, err := content.Encode(en)
	if err != nil {
		return err
	}
	return os.WriteFile(e.path("content", en.Path), data, 0o644)
}

// create writes a new entry, refusing to reuse a slug from any kind.
func (e *env) create(en content.Entry) error {
	if _, err := e.find(en.Slug); err == nil {
		return fmt.Errorf("content/%s already exists, or its slug %q is taken; pick a different title", en.Path, en.Slug)
	}
	data, err := content.Encode(en)
	if err != nil {
		return err
	}
	p := e.path("content", en.Path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("content/%s already exists", en.Path)
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (e *env) today() content.Date { return content.DateOf(e.now()) }

// dateFlag adds -date, defaulting to today.
func dateFlag(fs *flag.FlagSet) *string {
	return fs.String("date", "", "date to record, as 2006-01-02 (default today)")
}

func (e *env) dateOr(s string) (content.Date, error) {
	if s == "" {
		return e.today(), nil
	}
	return content.ParseDate(s)
}

func runNewPost(e *env, fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.Join(fs.Args(), " ")
	var err error
	if title == "" {
		if title, err = e.askRequired("Title"); err != nil {
			return err
		}
	}
	entries, err := e.entries()
	if err != nil {
		return err
	}
	tags, err := e.askTags(entries)
	if err != nil {
		return err
	}
	slug := slugify(title)
	en := content.Entry{
		Kind: content.KindPost, Slug: slug, Path: "posts/" + slug + ".md",
		Title: title, Date: e.today(), Tags: tags, Draft: true,
	}
	if err := e.create(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Created content/%s as a draft.\nWrite it, preview with ryu serve, then: ryu publish %s\n", en.Path, slug)
	return nil
}

func (e *env) askTags(entries []content.Entry) ([]string, error) {
	var inUse []string
	for _, en := range entries {
		for _, t := range en.Tags {
			if !slices.Contains(inUse, t) {
				inUse = append(inUse, t)
			}
		}
	}
	slices.Sort(inUse)
	if len(inUse) > 0 {
		fmt.Fprintf(e.out, "Tags in use: %s\n", strings.Join(inUse, ", "))
	}
	answer, err := e.ask("Tags, separated by spaces (optional)", "")
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, t := range strings.FieldsFunc(answer, func(r rune) bool { return r == ' ' || r == ',' }) {
		if t = slugify(t); t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

func runBookAdd(e *env, fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.Join(fs.Args(), " ")
	var err error
	if title == "" {
		if title, err = e.askRequired("Title"); err != nil {
			return err
		}
	}
	b := &content.Book{Added: e.today()}

	docs, err := e.search.Search(context.Background(), title)
	if err != nil {
		fmt.Fprintf(e.out, "Open Library lookup failed (%v); enter the details by hand.\n", err)
	}
	var candidates []openlibrary.Doc
	var options []string
	for _, d := range docs {
		if len(d.Authors) == 0 {
			continue
		}
		if len(options) == 5 {
			break
		}
		candidates = append(candidates, d)
		detail := strings.Join(d.Authors, ", ")
		if d.FirstPublishYear != 0 {
			detail += fmt.Sprintf(", %d", d.FirstPublishYear)
		}
		if d.Pages != 0 {
			detail += fmt.Sprintf(", ~%d pages", d.Pages)
		}
		options = append(options, fmt.Sprintf("%s — %s", d.Title, detail))
	}
	chosen := -1
	if len(options) > 0 {
		fmt.Fprintln(e.out, "Open Library found:")
		if chosen, err = e.pick("Which one", options, "none of these"); err != nil {
			return err
		}
	}
	if chosen >= 0 {
		b.Authors, b.Pages = candidates[chosen].Authors, candidates[chosen].Pages
	} else {
		author, err := e.ask("Author (separate several with ;)", "")
		if err != nil {
			return err
		}
		for a := range strings.SplitSeq(author, ";") {
			if a = strings.TrimSpace(a); a != "" {
				b.Authors = append(b.Authors, a)
			}
		}
	}
	note, err := e.ask("Note, like where you heard of it (optional)", "")
	if err != nil {
		return err
	}

	slug := slugify(title)
	en := content.Entry{Kind: content.KindBook, Slug: slug, Path: "books/" + slug + ".md", Title: title, Book: b}
	if note != "" {
		en.Body = note + "\n"
	}
	if err := e.create(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Added %s to the to-read list (content/%s).\nWhen you start it: ryu book start %s\n", title, en.Path, slug)
	return nil
}

// pickBook resolves a slug argument, or offers the books in the given
// statuses, in that order and newest first within each.
func (e *env) pickBook(slug, label string, statuses ...content.Status) (content.Entry, error) {
	if slug != "" {
		en, err := e.find(slug)
		if err != nil {
			return en, err
		}
		if en.Kind != content.KindBook {
			return en, fmt.Errorf("%s is a %s, not a book", slug, en.Kind)
		}
		return en, nil
	}
	entries, err := e.entries()
	if err != nil {
		return content.Entry{}, err
	}
	var books []content.Entry
	for _, st := range statuses {
		var group []content.Entry
		for _, en := range entries {
			if en.Kind == content.KindBook && en.Book.Status() == st {
				group = append(group, en)
			}
		}
		slices.SortFunc(group, newestFirst)
		books = append(books, group...)
	}
	if len(books) == 0 {
		return content.Entry{}, errors.New("no books to choose from; add one with ryu book add")
	}
	options := make([]string, len(books))
	for i, b := range books {
		options[i] = describeBook(b)
	}
	i, err := e.pick(label, options, "")
	if err != nil {
		return content.Entry{}, err
	}
	return books[i], nil
}

func describeBook(b content.Entry) string {
	s := b.Title
	if len(b.Book.Authors) > 0 {
		s += " — " + strings.Join(b.Book.Authors, ", ")
	}
	return s
}

func newestFirst(a, b content.Entry) int {
	return cmp.Or(b.Date.Time().Compare(a.Date.Time()), cmp.Compare(a.Slug, b.Slug))
}

func runBookStart(e *env, fs *flag.FlagSet, args []string) error {
	date := dateFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	when, err := e.dateOr(*date)
	if err != nil {
		return err
	}
	en, err := e.pickBook(fs.Arg(0), "Which book", content.StatusToRead)
	if err != nil {
		return err
	}
	if st := en.Book.Status(); st != content.StatusToRead {
		return fmt.Errorf("%s is already %s", en.Title, st)
	}
	en.Book.Started = when
	if err := e.save(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Started %s on %s.\nWhen you're done: ryu book finish %s\n", en.Title, when, en.Slug)
	return nil
}

func runBookFinish(e *env, fs *flag.FlagSet, args []string) error {
	date := dateFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	when, err := e.dateOr(*date)
	if err != nil {
		return err
	}
	en, err := e.pickBook(fs.Arg(0), "Which book", content.StatusReading, content.StatusToRead)
	if err != nil {
		return err
	}
	b := en.Book
	if b.Status() == content.StatusRead {
		return fmt.Errorf("%s was already finished on %s", en.Title, b.Finished)
	}
	fmt.Fprintf(e.out, "Finishing %s.\n", describeBook(en))

	ratings := make([]string, len(content.Ratings))
	for i, r := range content.Ratings {
		ratings[i] = string(r)
	}
	rating, err := e.choose("Rating", ratings, string(b.Rating))
	if err != nil {
		return err
	}
	form, err := e.choose("Form", []string{string(content.FormFiction), string(content.FormNonfiction)}, string(b.Form))
	if err != nil {
		return err
	}
	pages := b.Pages
	for {
		def := ""
		if pages > 0 {
			def = strconv.Itoa(pages)
		}
		s, err := e.ask("Pages", def)
		if err != nil {
			return err
		}
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			pages = n
			break
		}
		fmt.Fprintln(e.out, "Enter a page count.")
	}

	b.Rating, b.Form, b.Pages, b.Finished = content.Rating(rating), content.Form(form), pages, when
	if err := e.save(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Finished %s on %s. Write about it in content/%s\n", en.Title, when, en.Path)
	return nil
}

func runPublish(e *env, fs *flag.FlagSet, args []string) error {
	keepDate := fs.Bool("keep-date", false, "keep the draft's date instead of using today")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var en content.Entry
	var err error
	if slug := fs.Arg(0); slug != "" {
		if en, err = e.find(slug); err != nil {
			return err
		}
	} else {
		drafts, err := e.drafts()
		if err != nil {
			return err
		}
		if len(drafts) == 0 {
			fmt.Fprintln(e.out, "No drafts.")
			return nil
		}
		options := make([]string, len(drafts))
		for i, d := range drafts {
			options[i] = fmt.Sprintf("%s (%s)", d.Title, d.Kind)
		}
		i, err := e.pick("Publish which", options, "")
		if err != nil {
			return err
		}
		en = drafts[i]
	}
	if !en.Draft {
		return fmt.Errorf("%s is not a draft", en.Slug)
	}
	en.Draft = false
	if !*keepDate && (en.Kind == content.KindPost || en.Kind == content.KindAtom) {
		en.Date = e.today()
	}
	if err := e.save(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Published %s.\n", en.Title)
	return nil
}

func (e *env) drafts() ([]content.Entry, error) {
	entries, err := e.entries()
	if err != nil {
		return nil, err
	}
	var drafts []content.Entry
	for _, en := range entries {
		if en.Draft {
			drafts = append(drafts, en)
		}
	}
	slices.SortFunc(drafts, newestFirst)
	return drafts, nil
}

func runStatus(e *env, fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	entries, err := e.entries()
	if err != nil {
		return err
	}
	slices.SortFunc(entries, newestFirst)
	var reading, toRead, drafts []content.Entry
	for _, en := range entries {
		switch {
		case en.Draft:
			drafts = append(drafts, en)
		case en.Kind != content.KindBook:
		case en.Book.Status() == content.StatusReading:
			reading = append(reading, en)
		case en.Book.Status() == content.StatusToRead:
			toRead = append(toRead, en)
		}
	}

	section := func(title string, items []content.Entry, line func(content.Entry) string) {
		fmt.Fprintf(e.out, "%s (%d)\n", title, len(items))
		if len(items) == 0 {
			fmt.Fprintln(e.out, "  none")
		}
		for _, en := range items {
			fmt.Fprintf(e.out, "  %s\n", line(en))
		}
		fmt.Fprintln(e.out)
	}
	section("Reading now", reading, func(en content.Entry) string {
		return fmt.Sprintf("%s (since %s)", describeBook(en), en.Book.Started.Format("Jan 2"))
	})
	section("Up next", toRead, describeBook)
	section("Drafts", drafts, func(en content.Entry) string {
		return fmt.Sprintf("%s (%s, content/%s)", en.Title, en.Kind, en.Path)
	})
	return nil
}

var folds = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ä': "a", 'ã': "a", 'å': "a", 'æ': "ae",
	'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ñ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ø': "o", 'œ': "oe",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ý': "y", 'ÿ': "y", 'ß': "ss",
}

// slugify makes a URL slug from a title: lowercase ASCII words joined by
// dashes, with apostrophes dropped and common accents folded.
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if f, ok := folds[r]; ok {
			b.WriteString(f)
			dash = false
			continue
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case r == '\'' || r == '’':
		default:
			if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
