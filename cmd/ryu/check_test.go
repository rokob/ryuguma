package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newSiteHarness is a harness that can render the site, using the real
// templates and stylesheet.
func newSiteHarness(t *testing.T, files map[string]string) *harness {
	t.Helper()
	h := newHarness(t, files)
	for _, dir := range []string{"templates", "static"} {
		abs, err := filepath.Abs(filepath.Join("..", "..", dir))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(abs, filepath.Join(h.root, dir)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(h.root, "content", "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	return h
}

const (
	readDune  = "---\ntitle: Dune\nfinished: 2015-08-03\n---\n"
	draftPost = "---\ntitle: WIP\ndate: 2026-01-01\ndraft: true\n---\nAbout [[dune]].\n"
)

func TestCheckCountsDraftsAndReadsHistoryOnce(t *testing.T) {
	h := newSiteHarness(t, map[string]string{"books/dune.md": readDune, "posts/wip.md": draftPost})
	if err := h.run("", "check"); err != nil {
		t.Fatal(err)
	}
	if out := h.out.String(); !strings.Contains(out, "plus 1 draft\n") {
		t.Errorf("output = %q", out)
	}
	if h.historyReads != 1 {
		t.Errorf("read history %d times, want once", h.historyReads)
	}
}

func TestCheckValidatesDrafts(t *testing.T) {
	h := newSiteHarness(t, map[string]string{
		"books/dune.md": readDune,
		"posts/wip.md":  "---\ntitle: WIP\ndate: 2026-01-01\ndraft: true\n---\nAbout [[gone]].\n",
	})
	err := h.run("", "check")
	if err == nil || !strings.Contains(err.Error(), "gone") {
		t.Errorf("err = %v, want the draft's broken ref", err)
	}
}

func TestCheckCatchesPublishedLinksToDrafts(t *testing.T) {
	h := newSiteHarness(t, map[string]string{
		"posts/wip.md":  draftPost,
		"posts/pub.md":  "---\ntitle: Pub\ndate: 2026-01-02\n---\nSee [[wip]].\n",
		"books/dune.md": readDune,
	})
	err := h.run("", "check")
	if err == nil || !strings.Contains(err.Error(), "wip") {
		t.Errorf("err = %v, want the link to the draft", err)
	}
}
