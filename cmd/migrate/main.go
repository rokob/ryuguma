// Command migrate holds one-time conversions from the old Gatsby site.
//
//	migrate frontmatter              rewrite legacy frontmatter in place
//	migrate authors-lookup > a.tsv   look up missing book authors on Open Library
//	migrate authors-apply a.tsv      write reviewed authors into book files
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rokob/ryuguma/internal/content"
	"github.com/rokob/ryuguma/internal/openlibrary"
)

func main() {
	root := flag.String("content", "content", "content directory")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: migrate [-content dir] frontmatter | authors-lookup | authors-apply FILE")
		flag.PrintDefaults()
	}
	flag.Parse()

	var err error
	switch flag.Arg(0) {
	case "frontmatter":
		err = migrateFrontmatter(*root)
	case "authors-lookup":
		err = runAuthorsLookup(*root)
	case "authors-apply":
		if flag.NArg() != 2 {
			flag.Usage()
			os.Exit(2)
		}
		err = runAuthorsApply(*root, flag.Arg(1))
	default:
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// readFiles returns the markdown files in the given content dirs, keyed by
// path relative to root.
func readFiles(root string, dirs ...string) (map[string][]byte, error) {
	files := map[string][]byte{}
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(root, dir, "*.md"))
		if err != nil {
			return nil, err
		}
		for _, p := range matches {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(root, p)
			files[filepath.ToSlash(rel)] = data
		}
	}
	return files, nil
}

func writeFiles(root string, files map[string][]byte) error {
	for rel, data := range files {
		if err := os.WriteFile(filepath.Join(root, rel), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func migrateFrontmatter(root string) error {
	files, err := readFiles(root, "books", "posts")
	if err != nil {
		return err
	}
	targets := map[string]target{}
	for rel, data := range files {
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
	if err := writeFiles(root, out); err != nil {
		return err
	}
	return checkLoads(root)
}

func runAuthorsLookup(root string) error {
	entries, err := content.Load(os.DirFS(root))
	if err != nil {
		return err
	}
	client := &openlibrary.Client{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		UserAgent: "ryuguma-migrate (+https://github.com/rokob/ryuguma)",
	}
	// Open Library asks clients to stay around one request per second.
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	n := 0
	wait := func() {
		<-tick.C
		n++
		fmt.Fprintf(os.Stderr, "\rlooked up %d", n)
	}
	rows, err := lookupAuthors(context.Background(), client, entries, wait)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	return writeRows(os.Stdout, rows)
}

func runAuthorsApply(root, review string) error {
	f, err := os.Open(review)
	if err != nil {
		return err
	}
	defer f.Close()
	rows, err := readRows(f)
	if err != nil {
		return err
	}
	files, err := readFiles(root, "books")
	if err != nil {
		return err
	}
	out, err := applyAuthors(files, rows)
	if err != nil {
		return err
	}
	if err := writeFiles(root, out); err != nil {
		return err
	}
	fmt.Printf("set authors on %d books\n", len(out))
	return checkLoads(root)
}

func checkLoads(root string) error {
	entries, err := content.Load(os.DirFS(root))
	if err != nil {
		return fmt.Errorf("content does not load after migration: %w", err)
	}
	fmt.Printf("%d entries load cleanly\n", len(entries))
	return nil
}

func slugOf(rel string) string {
	base := filepath.Base(rel)
	return base[:len(base)-len(filepath.Ext(base))]
}
