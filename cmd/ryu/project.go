package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/history"
	"github.com/rokob/ryuguma/internal/render"
	"github.com/rokob/ryuguma/internal/site"
)

const configFile = "site.yaml"

// project is the site repo on disk.
type project struct {
	root string
	// readHistory reads the git history of paths in the repo at root. It
	// defaults to history.FromGit.
	readHistory func(root string, paths []string) (*history.Log, error)
}

func (p *project) path(parts ...string) string {
	return filepath.Join(append([]string{p.root}, parts...)...)
}

func (p *project) config() (render.Config, error) {
	var cfg render.Config
	data, err := os.ReadFile(p.path(configFile))
	if err != nil {
		return cfg, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", configFile, err)
	}
	return cfg, nil
}

// source is the content and its git history, read once so more than one site
// can be built from it.
type source struct {
	entries []content.Entry
	log     *history.Log
}

func (p *project) read() (source, error) {
	entries, err := content.Load(os.DirFS(p.path("content")))
	if err != nil {
		return source{}, err
	}
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = "content/" + e.Path
	}
	readHistory := p.readHistory
	if readHistory == nil {
		readHistory = history.FromGit
	}
	log, err := readHistory(p.root, paths)
	if err != nil {
		return source{}, err
	}
	return source{entries: entries, log: log}, nil
}

func (src source) site(opts site.Options) (*site.Site, error) {
	opts.History = func(path string) []history.Revision { return src.log.For("content/" + path) }
	return site.Build(src.entries, opts)
}

// build renders the whole site into out.
func (p *project) build(opts site.Options, out render.Output, now time.Time) (*site.Site, error) {
	src, err := p.read()
	if err != nil {
		return nil, err
	}
	return p.renderSite(src, opts, out, now)
}

// renderSite builds a site from src and renders it into out.
func (p *project) renderSite(src source, opts site.Options, out render.Output, now time.Time) (*site.Site, error) {
	cfg, err := p.config()
	if err != nil {
		return nil, err
	}
	cfg.BuiltAt = now
	s, err := src.site(opts)
	if err != nil {
		return nil, err
	}
	if err := render.Site(s, cfg, os.DirFS(p.path("templates")), out); err != nil {
		return nil, err
	}
	if err := render.Copy(out, "", os.DirFS(p.path("static"))); err != nil {
		return nil, err
	}
	if err := render.Copy(out, "images", os.DirFS(p.path("content", "images"))); err != nil {
		return nil, err
	}
	return s, nil
}

func runBuild(e *env, fs *flag.FlagSet, args []string) error {
	outDir := fs.String("out", "dist", "output directory inside the repo, relative to the repo root")
	drafts := fs.Bool("drafts", false, "include drafts")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := *outDir
	if !filepath.IsAbs(dir) {
		dir = e.path(dir)
	}
	if rel, err := filepath.Rel(e.root, dir); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing to build into %s: it must be a subdirectory of the repo", dir)
	}
	// Start clean so removed pages don't linger.
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	s, err := e.build(site.Options{Drafts: *drafts}, render.DirOutput(dir), e.now())
	if err != nil {
		return err
	}
	fmt.Fprintf(e.out, "built %s into %s\n", summarize(s), *outDir)
	return nil
}

func runCheck(e *env, fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	// Content and history are read once and built twice. Building without
	// drafts catches published pages linking to drafts, and building with
	// them checks the drafts themselves.
	src, err := e.read()
	if err != nil {
		return err
	}
	s, err := e.renderSite(src, site.Options{}, render.MemOutput{}, e.now())
	if err != nil {
		return err
	}
	withDrafts, err := src.site(site.Options{Drafts: true})
	if err != nil {
		return err
	}
	drafts := len(withDrafts.All) + len(withDrafts.Standalone) - len(s.All) - len(s.Standalone)
	fmt.Fprintf(e.out, "ok: %s, plus %d %s\n", summarize(s), drafts, plural(drafts, "draft", "drafts"))
	return nil
}

func runServe(e *env, fs *flag.FlagSet, args []string) error {
	addr := fs.String("addr", "localhost:8080", "address to listen on")
	drafts := fs.Bool("drafts", true, "include drafts")
	if err := fs.Parse(args); err != nil {
		return err
	}
	watch := &sourceWatcher{dirs: []string{e.path("content"), e.path("templates"), e.path("static"), e.path(configFile)}}
	build := func() (render.MemOutput, error) {
		out := render.MemOutput{}
		_, err := e.build(site.Options{Drafts: *drafts}, out, e.now())
		return out, err
	}
	fmt.Fprintf(e.out, "serving http://%s (drafts: %v); reload the page after editing\n", *addr, *drafts)
	err := http.ListenAndServe(*addr, newDevServer(build, watch.changed))
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func summarize(s *site.Site) string {
	count := func(n int, one, many string) string { return fmt.Sprintf("%d %s", n, plural(n, one, many)) }
	return fmt.Sprintf("%s read (%d reading, %d to read), %s, %s, %s",
		count(len(s.Books), "book", "books"), len(s.Reading), len(s.ToRead),
		count(len(s.Posts), "post", "posts"), count(len(s.Atoms), "atom", "atoms"), count(len(s.Standalone), "page", "pages"))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
