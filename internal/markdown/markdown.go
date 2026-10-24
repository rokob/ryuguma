// Package markdown renders entry bodies to HTML and resolves references
// between entries: [[slug]], [[slug|label]], and ordinary links to
// /books/…, /posts/…, or /atoms/… paths.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"slices"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Target is what a reference points at.
type Target struct {
	Slug  string
	URL   string
	Title string
	// Cite renders the default label as a <cite>, for book titles.
	Cite bool
}

type Resolver interface {
	Resolve(slug string) (Target, bool)
}

type Result struct {
	HTML template.HTML
	// Refs lists the slugs this body references, in order, without repeats.
	Refs []string
}

type Renderer struct {
	md goldmark.Markdown
}

func New(r Resolver) *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			// Ahead of the standard link parser, which also triggers on '['.
			parser.WithInlineParsers(util.Prioritized(wikiLinkParser{}, 199)),
			parser.WithASTTransformers(util.Prioritized(&refTransformer{resolver: r}, 100)),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(wikiLinkRenderer{}, 100)),
		),
	)
	return &Renderer{md: md}
}

func (r *Renderer) Render(src []byte) (Result, error) {
	refs := &refState{}
	pc := parser.NewContext()
	pc.Set(refsKey, refs)
	var buf bytes.Buffer
	if err := r.md.Convert(src, &buf, parser.WithContext(pc)); err != nil {
		return Result{}, err
	}
	return Result{HTML: template.HTML(buf.String()), Refs: refs.slugs}, errors.Join(refs.errs...)
}

var refsKey = parser.NewContextKey()

type refState struct {
	slugs []string
	errs  []error
}

func (s *refState) add(slug string) {
	if !slices.Contains(s.slugs, slug) {
		s.slugs = append(s.slugs, slug)
	}
}

var KindWikiLink = ast.NewNodeKind("WikiLink")

// WikiLink is a [[slug]] or [[slug|label]] reference.
type WikiLink struct {
	ast.BaseInline
	Slug  string
	Label string
	// Set by the transformer once resolved.
	Target   Target
	Resolved bool
}

func (n *WikiLink) Kind() ast.NodeKind { return KindWikiLink }

func (n *WikiLink) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Slug": n.Slug, "Label": n.Label}, nil)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type wikiLinkParser struct{}

func (wikiLinkParser) Trigger() []byte { return []byte{'['} }

func (wikiLinkParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if !bytes.HasPrefix(line, []byte("[[")) {
		return nil
	}
	end := bytes.Index(line, []byte("]]"))
	if end < 0 {
		return nil
	}
	slug, label, _ := bytes.Cut(line[2:end], []byte("|"))
	// Anything that isn't a slug, like Objective-C's [[Foo alloc] init] in
	// prose, is left for the other parsers.
	if !slugPattern.Match(slug) {
		return nil
	}
	block.Advance(end + 2)
	return &WikiLink{Slug: string(slug), Label: string(bytes.TrimSpace(label))}
}

var internalPath = regexp.MustCompile(`^/(books|posts|atoms)/([a-z0-9-]+)/?(#.*)?$`)

type refTransformer struct {
	resolver Resolver
}

func (t *refTransformer) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	refs := pc.Get(refsKey).(*refState)
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *WikiLink:
			target, ok := t.resolver.Resolve(n.Slug)
			if !ok {
				refs.errs = append(refs.errs, fmt.Errorf("[[%s]]: no entry with that slug", n.Slug))
				return ast.WalkContinue, nil
			}
			n.Target, n.Resolved = target, true
			refs.add(target.Slug)
		case *ast.Link:
			dest := string(n.Destination)
			m := internalPath.FindStringSubmatch(dest)
			if m == nil {
				return ast.WalkContinue, nil
			}
			target, ok := t.resolver.Resolve(m[2])
			if !ok || target.URL != "/"+m[1]+"/"+m[2]+"/" {
				refs.errs = append(refs.errs, fmt.Errorf("link to %s: no such entry", dest))
				return ast.WalkContinue, nil
			}
			refs.add(target.Slug)
		}
		return ast.WalkContinue, nil
	})
}

type wikiLinkRenderer struct{}

func (wikiLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindWikiLink, renderWikiLink)
}

func renderWikiLink(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*WikiLink)
	if !n.Resolved {
		fmt.Fprintf(w, `<span class="broken-ref">[[%s]]</span>`, template.HTMLEscapeString(n.Slug))
		return ast.WalkSkipChildren, nil
	}
	label := template.HTMLEscapeString(n.Target.Title)
	switch {
	case n.Label != "":
		label = template.HTMLEscapeString(n.Label)
	case n.Target.Cite:
		label = "<cite>" + label + "</cite>"
	}
	fmt.Fprintf(w, `<a href="%s">%s</a>`, template.HTMLEscapeString(n.Target.URL), label)
	return ast.WalkSkipChildren, nil
}
