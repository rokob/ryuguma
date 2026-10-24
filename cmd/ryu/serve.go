package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rokob/ryuguma/internal/render"
)

// devServer serves the site from memory and rebuilds it whenever a request
// arrives after the sources changed. Build errors show in the browser.
type devServer struct {
	build   func() (render.MemOutput, error)
	changed func() bool

	mu    sync.Mutex
	files render.MemOutput
	err   error
	built bool
}

func newDevServer(build func() (render.MemOutput, error), changed func() bool) *devServer {
	return &devServer{build: build, changed: changed}
}

func (s *devServer) current() (render.MemOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.built || s.changed() {
		start := time.Now()
		s.files, s.err = s.build()
		s.built = true
		if s.err != nil {
			fmt.Fprintf(os.Stderr, "build failed:\n%v\n", s.err)
		} else {
			fmt.Fprintf(os.Stderr, "built %d files in %s\n", len(s.files), time.Since(start).Round(time.Millisecond))
		}
	}
	return s.files, s.err
}

var errorPage = template.Must(template.New("").Parse(`<!doctype html>
<meta charset="utf-8"><title>Build failed</title>
<body style="font: 15px/1.5 ui-monospace, monospace; padding: 2rem; max-width: 60rem">
<h1 style="color: #9a3b2f">Build failed</h1>
<pre style="white-space: pre-wrap">{{.}}</pre>
<p>Fix the problem and reload.</p>`))

func (s *devServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	files, err := s.current()
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		errorPage.Execute(w, err.Error())
		return
	}

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if strings.HasSuffix(r.URL.Path, "/") {
		name = path.Join(name, "index.html")
	}
	data, ok := files[name]
	if !ok && path.Ext(name) == "" {
		if _, ok := files[path.Join(name, "index.html")]; ok {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
	}
	status := http.StatusOK
	if !ok {
		name, data, status = "404.html", files["404.html"], http.StatusNotFound
	}
	ctype := mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(data)
}

// sourceWatcher reports whether any file under dirs was added, removed, or
// modified since the last call.
type sourceWatcher struct {
	dirs []string
	last string
}

func (w *sourceWatcher) changed() bool {
	var latest time.Time
	count := 0
	for _, dir := range w.dirs {
		filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			count++
			if info, err := d.Info(); err == nil && info.ModTime().After(latest) {
				latest = info.ModTime()
			}
			return nil
		})
	}
	stamp := fmt.Sprintf("%d@%d", count, latest.UnixNano())
	if stamp == w.last {
		return false
	}
	w.last = stamp
	return true
}
