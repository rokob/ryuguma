// Command migrate converts content from the old Gatsby site's frontmatter to
// the ryuguma format, rewriting files in place. It is meant to run once.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rokob/ryuguma/internal/content"
)

func main() {
	root := flag.String("content", "content", "content directory")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	var paths []string
	for _, dir := range []string{"books", "posts"} {
		matches, err := filepath.Glob(filepath.Join(root, dir, "*.md"))
		if err != nil {
			return err
		}
		paths = append(paths, matches...)
	}

	files := map[string][]byte{}
	targets := map[string]target{}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		files[filepath.ToSlash(rel)] = data
		fm, _, err := parseLegacy(data)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		targets[slugOf(rel)] = target{Title: fm.Title, Dir: filepath.Dir(rel)}
	}

	var errs []error
	out := map[string][]byte{}
	for rel, data := range files {
		e, err := convert(rel, data, targets)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		encoded, err := content.Encode(e)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", rel, err))
			continue
		}
		out[rel] = encoded
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	for rel, data := range out {
		if err := os.WriteFile(filepath.Join(root, rel), data, 0o644); err != nil {
			return err
		}
	}

	entries, err := content.Load(os.DirFS(root))
	if err != nil {
		return fmt.Errorf("migrated content does not load: %w", err)
	}
	fmt.Printf("migrated %d entries\n", len(entries))
	return nil
}

func slugOf(rel string) string {
	base := filepath.Base(rel)
	return base[:len(base)-len(filepath.Ext(base))]
}
