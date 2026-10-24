package content

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// Load parses every entry under the content root. It reports all broken
// files at once rather than stopping at the first.
func Load(fsys fs.FS) ([]Entry, error) {
	var (
		entries []Entry
		errs    []error
	)
	dirs := make([]string, 0, len(Dirs))
	for d := range Dirs {
		dirs = append(dirs, d)
	}
	slices.Sort(dirs)

	for _, dir := range dirs {
		files, err := fs.ReadDir(fsys, dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, f := range files {
			name := f.Name()
			if f.IsDir() || strings.HasPrefix(name, ".") || path.Ext(name) != ".md" {
				continue
			}
			p := path.Join(dir, name)
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			e, err := Parse(p, data)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			entries = append(entries, e)
		}
	}

	seen := map[string]string{}
	for _, e := range entries {
		if other, ok := seen[e.Slug]; ok {
			errs = append(errs, fmt.Errorf("%s: slug %q is already used by %s", e.Path, e.Slug, other))
			continue
		}
		seen[e.Slug] = e.Path
	}
	return entries, errors.Join(errs...)
}
