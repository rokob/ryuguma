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

	"go.yaml.in/yaml/v3"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/render"
	"github.com/rokob/ryuguma/internal/site"
)

const configFile = "site.yaml"

// project is the site repo on disk.
type project struct {
	root string
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

func (p *project) load(opts site.Options) (*site.Site, error) {
	entries, err := content.Load(os.DirFS(p.path("content")))
	if err != nil {
		return nil, err
	}
	return site.Build(entries, opts)
}

// build renders the whole site into out.
func (p *project) build(opts site.Options, out render.Output) (*site.Site, error) {
	cfg, err := p.config()
	if err != nil {
		return nil, err
	}
	s, err := p.load(opts)
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

func runBuild(p *project, fs *flag.FlagSet, args []string) error {
	outDir := fs.String("out", "dist", "output directory, relative to the repo root")
	drafts := fs.Bool("drafts", false, "include drafts")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := p.path(*outDir)
	if rel, err := filepath.Rel(p.root, dir); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing to build into %s: it must be a subdirectory of the repo", dir)
	}
	// Start clean so removed pages don't linger.
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	s, err := p.build(site.Options{Drafts: *drafts}, render.DirOutput(dir))
	if err != nil {
		return err
	}
	fmt.Printf("built %s into %s\n", summarize(s), *outDir)
	return nil
}

func runCheck(p *project, fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	// Building without drafts also catches published pages linking to drafts.
	s, err := p.build(site.Options{}, render.MemOutput{})
	if err != nil {
		return err
	}
	withDrafts, err := p.load(site.Options{Drafts: true})
	if err != nil {
		return err
	}
	drafts := len(withDrafts.All) + len(withDrafts.Standalone) - len(s.All) - len(s.Standalone)
	fmt.Printf("ok: %s, plus %d %s\n", summarize(s), drafts, plural(drafts, "draft", "drafts"))
	return nil
}

func runServe(p *project, fs *flag.FlagSet, args []string) error {
	addr := fs.String("addr", "localhost:8080", "address to listen on")
	drafts := fs.Bool("drafts", true, "include drafts")
	if err := fs.Parse(args); err != nil {
		return err
	}
	watch := &sourceWatcher{dirs: []string{p.path("content"), p.path("templates"), p.path("static"), p.path(configFile)}}
	build := func() (render.MemOutput, error) {
		out := render.MemOutput{}
		_, err := p.build(site.Options{Drafts: *drafts}, out)
		return out, err
	}
	fmt.Printf("serving http://%s (drafts: %v); reload the page after editing\n", *addr, *drafts)
	err := http.ListenAndServe(*addr, newDevServer(build, watch.changed))
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func summarize(s *site.Site) string {
	return fmt.Sprintf("%d books read (%d reading, %d to read), %d posts, %d atoms, %d pages",
		len(s.Books), len(s.Reading), len(s.ToRead), len(s.Posts), len(s.Atoms), len(s.Standalone))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
