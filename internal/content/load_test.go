package content_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rokob/ryuguma/internal/content"
)

func TestLoadReadsAllKinds(t *testing.T) {
	fsys := fstest.MapFS{
		"books/dune.md":        {Data: []byte("---\ntitle: Dune\nfinished: 2015-08-03\n---\n")},
		"posts/hello.md":       {Data: []byte("---\ntitle: Hello\ndate: 2014-01-01\n---\n")},
		"atoms/idea.md":        {Data: []byte("---\ntitle: Idea\ndate: 2026-01-01\n---\nidea\n")},
		"images/flow.png":      {Data: []byte("png")},
		"books/notes.txt":      {Data: []byte("ignored")},
		"README.md":            {Data: []byte("not content")},
		"books/.hidden.md":     {Data: []byte("ignored")},
		"posts/drafty.md":      {Data: []byte("---\ntitle: Drafty\ndate: 2014-01-02\ndraft: true\n---\n")},
		"atoms/nested/deep.md": {Data: []byte("---\ndate: 2026-01-01\n---\n")},
	}
	entries, err := content.Load(fsys)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := map[string]content.Kind{}
	for _, e := range entries {
		got[e.Path] = e.Kind
	}
	want := map[string]content.Kind{
		"books/dune.md":   content.KindBook,
		"posts/hello.md":  content.KindPost,
		"posts/drafty.md": content.KindPost,
		"atoms/idea.md":   content.KindAtom,
	}
	if len(got) != len(want) {
		t.Errorf("loaded %v, want %v", got, want)
	}
	for p, k := range want {
		if got[p] != k {
			t.Errorf("%s: kind %q, want %q", p, got[p], k)
		}
	}
}

func TestLoadReportsEveryBrokenFile(t *testing.T) {
	fsys := fstest.MapFS{
		"books/ok.md":   {Data: []byte("---\ntitle: OK\n---\n")},
		"books/bad.md":  {Data: []byte("---\nrating: great\n---\n")},
		"posts/bad2.md": {Data: []byte("no frontmatter")},
	}
	_, err := content.Load(fsys)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, p := range []string{"books/bad.md", "posts/bad2.md"} {
		if !strings.Contains(err.Error(), p) {
			t.Errorf("error does not mention %s:\n%v", p, err)
		}
	}
}

func TestLoadRejectsSlugCollisionsAcrossKinds(t *testing.T) {
	fsys := fstest.MapFS{
		"books/flow.md": {Data: []byte("---\ntitle: Flow\n---\n")},
		"posts/flow.md": {Data: []byte("---\ntitle: Flow\ndate: 2020-01-01\n---\n")},
	}
	_, err := content.Load(fsys)
	if err == nil || !strings.Contains(err.Error(), "flow") {
		t.Fatalf("expected slug collision error, got %v", err)
	}
}
