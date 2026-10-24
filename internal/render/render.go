// Package render writes the site model out as static files using the HTML
// templates in the theme directory.
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/rokob/ryuguma/internal/site"
)

type Config struct {
	Title  string `yaml:"title"`
	URL    string `yaml:"url"`
	Author string `yaml:"author"`
	Repo   string `yaml:"repo"`
}

// Output receives generated files. Names are slash-separated paths relative
// to the site root, like "books/dune/index.html".
type Output interface {
	WriteFile(name string, data []byte) error
}

// MemOutput keeps files in memory, for tests and the dev server.
type MemOutput map[string][]byte

func (m MemOutput) WriteFile(name string, data []byte) error {
	m[name] = data
	return nil
}

// DirOutput writes files under a directory.
type DirOutput string

func (d DirOutput) WriteFile(name string, data []byte) error {
	p := filepath.Join(string(d), filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Data is what every template receives. Page and Tag are set on the pages
// that are about one entry or one tag.
type Data struct {
	Config Config
	Site   *site.Site
	// Path is the URL path of the page being rendered.
	Path  string
	Title string
	Page  *site.Page
	Tag   *site.Tag
}

// homeSlug names the standalone page whose body introduces the home page.
const homeSlug = "home"

// Site renders every page of s with the templates in tmpl.
func Site(s *site.Site, cfg Config, tmpl fs.FS, out Output) error {
	r := &renderer{cfg: cfg, site: s, tmpl: tmpl, out: out, parsed: map[string]*template.Template{}}

	r.page("index.html", "home.html", Data{Path: "/", Page: s.Lookup(homeSlug)})
	r.page("404.html", "404.html", Data{Title: "Not found"})
	r.page("books/index.html", "books.html", Data{Path: "/books/", Title: "Books"})
	r.page("posts/index.html", "posts.html", Data{Path: "/posts/", Title: "Posts"})
	r.page("atoms/index.html", "atoms.html", Data{Path: "/atoms/", Title: "Atoms"})
	r.page("tags/index.html", "tags.html", Data{Path: "/tags/", Title: "Tags"})

	for _, p := range s.All {
		r.page(indexFor(p.URL), string(p.Kind)+".html", Data{Path: p.URL, Title: p.DisplayTitle(), Page: p})
		if len(p.History) > 0 {
			r.page(indexFor(p.URL+"history/"), "history.html", Data{Path: p.URL + "history/", Title: "History of " + p.DisplayTitle(), Page: p})
		}
	}
	for _, p := range s.Standalone {
		if p.Slug == homeSlug {
			continue
		}
		r.page(indexFor(p.URL), "page.html", Data{Path: p.URL, Title: p.Title, Page: p})
	}
	for i := range s.Tags {
		t := &s.Tags[i]
		r.page(indexFor(t.URL), "tag.html", Data{Path: t.URL, Title: "#" + t.Name, Tag: t})
	}
	return r.err
}

// Copy writes every file in fsys to out under prefix.
func Copy(out Output, prefix string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Base(p)[0] == '.' {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		return out.WriteFile(path.Join(prefix, p), data)
	})
}

func indexFor(url string) string {
	return path.Join(url[1:], "index.html")
}

type renderer struct {
	cfg    Config
	site   *site.Site
	tmpl   fs.FS
	out    Output
	parsed map[string]*template.Template
	err    error
}

func (r *renderer) page(name, tmplName string, d Data) {
	if r.err != nil {
		return
	}
	t, err := r.template(tmplName)
	if err != nil {
		r.err = err
		return
	}
	d.Config, d.Site = r.cfg, r.site
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", d); err != nil {
		r.err = fmt.Errorf("rendering %s: %w", name, err)
		return
	}
	r.err = r.out.WriteFile(name, buf.Bytes())
}

// template parses a page template together with the shared _*.html files.
func (r *renderer) template(name string) (*template.Template, error) {
	if t, ok := r.parsed[name]; ok {
		return t, nil
	}
	t, err := template.New(name).Funcs(funcs).ParseFS(r.tmpl, "_*.html", name)
	if err != nil {
		return nil, err
	}
	r.parsed[name] = t
	return t, nil
}
