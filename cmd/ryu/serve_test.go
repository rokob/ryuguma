package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rokob/ryuguma/internal/render"
)

// fakeProject counts builds and lets a test mark sources as changed.
type fakeProject struct {
	builds  int
	changed bool
	files   render.MemOutput
	err     error
}

func (f *fakeProject) build() (render.MemOutput, error) {
	f.builds++
	return f.files, f.err
}

func (f *fakeProject) sourcesChanged() bool {
	c := f.changed
	f.changed = false
	return c
}

func get(t *testing.T, h http.Handler, path string) (int, string, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, _ := io.ReadAll(rec.Body)
	return rec.Code, rec.Header().Get("Content-Type"), string(body)
}

func TestDevServerServesBuiltFiles(t *testing.T) {
	p := &fakeProject{files: render.MemOutput{
		"index.html":            []byte("home"),
		"books/dune/index.html": []byte("dune"),
		"style.css":             []byte("css"),
		"404.html":              []byte("lost"),
	}}
	srv := newDevServer(p.build, p.sourcesChanged)

	cases := []struct {
		path, body, ctype string
		code              int
	}{
		{"/", "home", "text/html; charset=utf-8", 200},
		{"/books/dune/", "dune", "text/html; charset=utf-8", 200},
		{"/style.css", "css", "text/css; charset=utf-8", 200},
		{"/missing/", "lost", "text/html; charset=utf-8", 404},
	}
	for _, c := range cases {
		code, ctype, body := get(t, srv, c.path)
		if code != c.code || body != c.body || ctype != c.ctype {
			t.Errorf("%s: got %d %q %q, want %d %q %q", c.path, code, ctype, body, c.code, c.ctype, c.body)
		}
	}
	if p.builds != 1 {
		t.Errorf("built %d times, want once while nothing changed", p.builds)
	}
}

func TestDevServerRedirectsMissingSlash(t *testing.T) {
	p := &fakeProject{files: render.MemOutput{"books/dune/index.html": []byte("dune")}}
	srv := newDevServer(p.build, p.sourcesChanged)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/books/dune", nil))
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/books/dune/" {
		t.Errorf("got %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestDevServerRebuildsWhenSourcesChange(t *testing.T) {
	p := &fakeProject{files: render.MemOutput{"index.html": []byte("v1")}}
	srv := newDevServer(p.build, p.sourcesChanged)
	get(t, srv, "/")

	p.files = render.MemOutput{"index.html": []byte("v2")}
	p.changed = true
	if _, _, body := get(t, srv, "/"); body != "v2" {
		t.Errorf("body = %q after change", body)
	}
	if p.builds != 2 {
		t.Errorf("builds = %d", p.builds)
	}
}

func TestDevServerShowsBuildErrors(t *testing.T) {
	p := &fakeProject{err: errors.New("books/dune.md: [[nope]]: no entry with that slug")}
	srv := newDevServer(p.build, p.sourcesChanged)
	code, _, body := get(t, srv, "/")
	if code != http.StatusInternalServerError || !strings.Contains(body, "[[nope]]") {
		t.Errorf("got %d %q", code, body)
	}
}
