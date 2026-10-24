# ryuguma

Source for [ryuguma.com](https://ryuguma.com): books I've read (and want to read), posts, and atoms
(small ideas).

A small Go program, `ryu`, reads everything under `content/`, builds an in-memory model of the site
(pages, references in both directions, tags, reading stats, git history), and renders static HTML with
`html/template`. No framework and no client-side JavaScript.

This repo continues the history of [rokob.github.com](https://github.com/rokob/rokob.github.com)
(Jekyll, then Gatsby), so every post's history reaches back to when it was first written.

## Writing

Run `ryu` with no arguments to see every command. Anything you leave off is asked for.

```sh
go install ./cmd/ryu      # or: go run ./cmd/ryu <command>

ryu status                # reading now, up next, open drafts
ryu book add "Piranesi"   # look it up on Open Library, add to the to-read list
ryu book start            # pick from the to-read list
ryu book finish           # pick from what you're reading; asks rating, form, pages
ryu new post "Title"      # start a draft post
ryu publish               # pick a draft, publish it dated today
ryu serve                 # preview at localhost:8080 with drafts; reload after editing
ryu check                 # validate content, references, and templates
```

After `ryu` changes a file, commit it with jj as usual. Pushing to `main` deploys.

## Content

Everything is markdown with YAML frontmatter. The directory decides the kind; the filename is the slug
and the URL (`content/books/dune.md` → `/books/dune/`). Slugs are unique across all kinds.

**Books** (`content/books/`). One file follows a book from the to-read list to the write-up. Status
comes from the dates: no `started` means to read, `started` without `finished` means reading,
`finished` means read.

```yaml
---
title: The Wind-Up Bird Chronicle
author: Haruki Murakami          # or a list: [Daniel J. Siegel, Tina Payne Bryson]
added: 2026-01-10                # put on the to-read list
started: 2026-08-01
finished: 2026-10-19
pages: 626
rating: great                    # bad | okay | good | great
form: fiction                    # fiction | nonfiction
tags: [literary]
---
```

**Posts** (`content/posts/`) take `title`, `date`, optional `summary`, `tags`, and `draft`.
**Atoms** (`content/atoms/`) take `date`, optional `title`, `tags`, and `draft`.
**Pages** (`content/pages/`) take only `title` and render at `/<slug>/`. `home.md` is the home page
intro, and `reading.md` is the intro to the reading stats page.

Unknown keys, bad dates, and bad ratings fail the build with the file and line.

### Cross-references

- `[[dune]]` links to any entry by slug, labelled with its title (book titles in italics).
- `[[dune|that spice book]]` uses your own label.
- Ordinary links to `/books/…`, `/posts/…`, or `/atoms/…` work too and are checked the same way.

Every page lists what links to it under "Referenced by". A reference to a missing entry, or from a
published page to a draft, fails the build.

## Design

- `templates/` holds the HTML: `_layout.html` and `_partials.html` are shared, and the rest are one file
  per page type. A standalone page can have its own template, like `page-reading.html` for `/reading/`.
- `static/style.css` starts with design tokens (fonts, sizes, spacing, colors for light and dark). Most
  visual changes only touch those. The chart ramp is validated for contrast in both schemes, so keep
  that true when changing it.
- `ryu serve` reads templates and CSS from disk on every change, so design edits need no rebuild. New
  template functions (in `internal/render/funcs.go`) do need a restart.

## History

Each entry's footer links its latest commit, and `/…/history/` lists every revision with links to
the commit and that version of the file on GitHub. It comes from one `git log` pass that follows
renames, so it needs full history. `ryu` refuses to build from a shallow clone.

## Deploying

Cloudflare Workers Builds deploys on every push to `main`, serving `dist/` as static assets only, with
no Worker script (`wrangler.jsonc`). One-time setup in the Cloudflare dashboard:

1. **Workers & Pages → Create → Import a repository**, and pick `rokob/ryuguma`.
2. Build command: `sh scripts/ci-build.sh`. Deploy command: `npx wrangler deploy` (the default).
3. The `ryuguma.com` custom domain in `wrangler.jsonc` is attached on the first deploy. Remove any
   existing apex DNS record for it first.

`scripts/ci-build.sh` unshallows the checkout, runs the tests, and builds. It runs the same locally.

## Layout

```
cmd/ryu/             the CLI: commands, dev server, project wiring
cmd/migrate/         one-time conversion from the Gatsby site (kept for reference)
internal/content/    file format: parse, validate, encode, load
internal/markdown/   goldmark setup and [[ref]] resolution
internal/site/       in-memory model: pages, refs, backlinks, tags, reading stats
internal/history/    per-file git history, following renames
internal/render/     templates to files
internal/openlibrary/ book lookup
content/ templates/ static/
```

## Roadmap

- [x] Content model, migration from Gatsby, author backfill
- [x] Renderer, templates, dev server, refs and backlinks
- [x] Git history pages and GitHub links
- [x] Reading stats page
- [x] `ryu` writing commands for books, posts, and drafts
- [x] Deploy config for Cloudflare
- [ ] Atoms: format and `ryu new atom`
- [ ] Feeds (Atom) and sitemap
- [ ] Search
- [ ] Syntax highlighting at build time
- [ ] Redirect old rokob.github.io URLs
- [ ] Design pass
