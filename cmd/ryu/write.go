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

// choose asks until the answer is one of allowed, or the start of exactly
// one of them.
func (e *env) choose(label string, allowed []string, def string) (string, error) {
	for {
		s, err := e.ask(fmt.Sprintf("%s (%s)", label, strings.Join(allowed, ", ")), def)
		if err != nil {
			return "", err
		}
		s = strings.ToLower(s)
		var matches []string
		for _, a := range allowed {
			if a == s {
				return a, nil
			}
			if s != "" && strings.HasPrefix(a, s) {
				matches = append(matches, a)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
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

// lookup finds an entry by slug; an error means the content didn't load.
func (e *env) lookup(slug string) (content.Entry, bool, error) {
	entries, err := e.entries()
	if err != nil {
		return content.Entry{}, false, err
	}
	for _, en := range entries {
		if en.Slug == slug {
			return en, true, nil
		}
	}
	return content.Entry{}, false, nil
}

func (e *env) find(slug string) (content.Entry, error) {
	en, found, err := e.lookup(slug)
	if err == nil && !found {
		err = fmt.Errorf("no entry with slug %q", slug)
	}
	return en, err
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

// askLink asks for a book's canonical link. Enter accepts def, which may be
// empty; anything else must be a full URL.
func (e *env) askLink(def string) (string, error) {
	for {
		s, err := e.ask("Link to the book (Enter to accept, or paste a URL)", def)
		if err != nil {
			return "", err
		}
		if s == "" || content.IsWebURL(s) {
			return s, nil
		}
		fmt.Fprintln(e.out, "Enter a full URL starting with https://, or leave it blank.")
	}
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
	fmt.Fprintf(e.out, "Created content/%s as a draft.\nWrite it, preview with ./ryu serve, then: ./ryu publish %s\n", en.Path, slug)
	return nil
}

func runNewAtom(e *env, fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.Join(fs.Args(), " ")
	var err error
	if title == "" {
		if title, err = e.askRequired("Name (becomes the title and the [[slug]])"); err != nil {
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
	fmt.Fprintln(e.out, "Type the atom, then press Ctrl-D on a new line:")
	body, err := io.ReadAll(e.in)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return errors.New("an atom needs some text; nothing was written")
	}
	slug := slugify(title)
	en := content.Entry{
		Kind: content.KindAtom, Slug: slug, Path: "atoms/" + slug + ".md",
		Title: title, Date: e.today(), Tags: tags, Body: text + "\n",
	}
	if err := e.create(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Published content/%s; it goes live with your next push.\n", en.Path)
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
	slugFlag := fs.String("slug", "", "file name and URL to use instead of one made from the title")
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
	slug := slugify(title)
	if *slugFlag != "" {
		if slugify(*slugFlag) != *slugFlag {
			return fmt.Errorf("invalid slug %q: use lowercase letters, digits, and dashes", *slugFlag)
		}
		slug = *slugFlag
	}

	// Sort out a clash with an existing entry before asking anything else.
	sameTitle := false
	existing, found, err := e.lookup(slug)
	if err != nil {
		return err
	}
	if found {
		fmt.Fprintf(e.out, "You already have %s.\n", describeExisting(existing))
		label, choices := "A different book with the same title", []string{"different", "cancel"}
		if existing.Kind == content.KindBook && existing.Book.Status() == content.StatusRead {
			label, choices = "Re-reading it, or a different book with the same title", []string{"reread", "different", "cancel"}
		}
		choice, err := e.choose(label, choices, "")
		if err != nil {
			return err
		}
		switch choice {
		case "cancel":
			fmt.Fprintln(e.out, "Nothing added.")
			return nil
		case "reread":
			return e.addReread(existing)
		}
		sameTitle = true
	}

	b := &content.Book{Added: e.today()}
	docs, err := e.search.Search(context.Background(), title)
	if err != nil {
		fmt.Fprintf(e.out, "Open Library lookup failed (%v); enter the details by hand.\n", err)
	}
	var candidates []openlibrary.Doc
	var options []string
	for _, d := range openlibrary.Rank(docs, title) {
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
		// Open Library gives the median page count across editions.
		switch {
		case d.Pages != 0 && d.Editions > 1:
			detail += fmt.Sprintf(", %d editions (median %d pages)", d.Editions, d.Pages)
		case d.Pages != 0:
			detail += fmt.Sprintf(", %d pages", d.Pages)
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
	defaultLink := ""
	if chosen >= 0 {
		c := candidates[chosen]
		b.Authors, b.Pages = c.Authors, c.Pages
		if c.Key != "" {
			defaultLink = openlibrary.DefaultBaseURL + c.Key
		}
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
	if b.Link, err = e.askLink(defaultLink); err != nil {
		return err
	}
	note, err := e.ask("Note, like where you heard of it (optional)", "")
	if err != nil {
		return err
	}

	if sameTitle {
		// The title stays the same; the author's surname keeps the slug apart.
		slug = withSurname(slug, b.Authors)
		if _, taken, err := e.lookup(slug); err != nil || taken {
			return fmt.Errorf("%s is taken too; pick a slug with ./ryu book add -slug <slug> %q", slug, title)
		}
	}
	en := content.Entry{Kind: content.KindBook, Slug: slug, Path: "books/" + slug + ".md", Title: title, Book: b}
	if note != "" {
		en.Body = note + "\n"
	}
	if err := e.create(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Added %s to the to-read list (content/%s).\nWhen you start it: ./ryu book start %s\n", title, en.Path, slug)
	return nil
}

// addReread makes a new to-read entry for another reading of a finished
// book, carrying over its details and linking back to the first read.
func (e *env) addReread(first content.Entry) error {
	slug := fmt.Sprintf("%s-%d", first.Slug, e.today().Time().Year())
	if _, taken, err := e.lookup(slug); err != nil || taken {
		return fmt.Errorf("%s already exists; pick a slug with ./ryu book add -slug <slug> %q", slug, first.Title)
	}
	note, err := e.ask("Note, like why you're reading it again (optional)", "")
	if err != nil {
		return err
	}
	fb := first.Book
	en := content.Entry{
		Kind: content.KindBook, Slug: slug, Path: "books/" + slug + ".md", Title: first.Title,
		Book: &content.Book{Authors: fb.Authors, Link: fb.Link, Pages: fb.Pages, Form: fb.Form, Added: e.today()},
		Body: fmt.Sprintf("Re-reading [[%s]], which I first read in %d.\n", first.Slug, fb.Finished.Time().Year()),
	}
	if note != "" {
		en.Body += "\n" + note + "\n"
	}
	if err := e.create(en); err != nil {
		return err
	}
	fmt.Fprintf(e.out, "Added a re-read of %s (content/%s).\nWhen you start it: ./ryu book start %s\n", first.Title, en.Path, slug)
	return nil
}

// describeExisting says what an entry is, for messages about clashes.
func describeExisting(en content.Entry) string {
	if en.Kind != content.KindBook {
		return fmt.Sprintf("a %s called %s (content/%s)", en.Kind, en.Title, en.Path)
	}
	b := en.Book
	s := en.Title
	if len(b.Authors) > 0 {
		s += " by " + strings.Join(b.Authors, ", ")
	}
	switch b.Status() {
	case content.StatusRead:
		s += ", finished " + b.Finished.Format("Jan 2, 2006")
	case content.StatusReading:
		s += ", reading since " + b.Started.Format("Jan 2, 2006")
	default:
		s += ", on your to-read list"
		if !b.Added.IsZero() {
			s += " since " + b.Added.Format("Jan 2, 2006")
		}
	}
	return s + " (content/" + en.Path + ")"
}

// withSurname adds the first author's surname to a slug, or -2 without one.
func withSurname(slug string, authors []string) string {
	if len(authors) > 0 {
		if words := strings.Fields(authors[0]); len(words) > 0 {
			if s := slugify(words[len(words)-1]); s != "" {
				return slug + "-" + s
			}
		}
	}
	return slug + "-2"
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
		return content.Entry{}, errors.New("no books to choose from; add one with ./ryu book add")
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
	fmt.Fprintf(e.out, "Started %s on %s.\nWhen you're done: ./ryu book finish %s\n", en.Title, when, en.Slug)
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
	link, err := e.askLink(b.Link)
	if err != nil {
		return err
	}

	b.Rating, b.Form, b.Pages, b.Link, b.Finished = content.Rating(rating), content.Form(form), pages, link, when
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
