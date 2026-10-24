// Command ryu builds, previews, and helps write the ryuguma site.
// Run it with no arguments for a list of commands.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rokob/ryuguma/internal/openlibrary"
)

// env is everything a command touches outside its arguments, so tests can
// supply a temp repo, a fixed clock, scripted input, and a fake lookup.
type env struct {
	*project
	now    func() time.Time
	in     *bufio.Reader
	out    io.Writer
	search searcher
}

type searcher interface {
	Search(ctx context.Context, title string) ([]openlibrary.Doc, error)
}

type command struct {
	name    string
	group   string
	summary string
	usage   string
	run     func(e *env, fs *flag.FlagSet, args []string) error
}

// Commands are listed in help in this order.
var commands = []command{
	{name: "status", group: "Writing", summary: "what you're reading, what's next, and open drafts", usage: "ryu status", run: runStatus},
	{name: "book add", group: "Writing", summary: "put a book on the to-read list, looked up on Open Library", usage: `ryu book add ["Title"]`, run: runBookAdd},
	{name: "book start", group: "Writing", summary: "start reading a book from the list", usage: "ryu book start [-date d] [slug]", run: runBookStart},
	{name: "book finish", group: "Writing", summary: "finish a book: rating, form, and pages", usage: "ryu book finish [-date d] [slug]", run: runBookFinish},
	{name: "new post", group: "Writing", summary: "start a draft post", usage: `ryu new post ["Title"]`, run: runNewPost},
	{name: "publish", group: "Writing", summary: "publish a draft, dated today", usage: "ryu publish [-keep-date] [slug]", run: runPublish},
	{name: "serve", group: "Site", summary: "preview locally with drafts, rebuilding on change", usage: "ryu serve [-addr :8080] [-drafts=true]", run: runServe},
	{name: "build", group: "Site", summary: "write the site to dist/", usage: "ryu build [-out dist] [-drafts]", run: runBuild},
	{name: "check", group: "Site", summary: "validate content, references, and templates", usage: "ryu check", run: runCheck},
}

func main() {
	e := &env{
		now: time.Now,
		in:  bufio.NewReader(os.Stdin),
		out: os.Stdout,
		search: &openlibrary.Client{
			HTTP:      &http.Client{Timeout: 15 * time.Second},
			UserAgent: "ryuguma (+https://github.com/rokob/ryuguma)",
		},
	}
	if err := run(os.Args[1:], e); err != nil {
		var usage usageError
		if errors.As(err, &usage) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

type usageError string

func (u usageError) Error() string { return string(u) }

func run(args []string, e *env) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printHelp(e.out)
		return nil
	}
	c, rest, ok := lookup(args)
	if !ok {
		return usageError(fmt.Sprintf("ryu: unknown command %q (run ryu with no arguments for help)", strings.Join(args[:min(2, len(args))], " ")))
	}
	if e.project == nil {
		p, err := findProject()
		if err != nil {
			return err
		}
		e.project = p
	}
	fs := newFlags(c)
	fs.SetOutput(e.out)
	if err := c.run(e, fs, rest); !errors.Is(err, flag.ErrHelp) {
		return err
	}
	return nil
}

// lookup matches two-word commands like "book add" before one-word ones.
func lookup(args []string) (command, []string, bool) {
	for _, n := range []int{2, 1} {
		if len(args) < n {
			continue
		}
		name := strings.Join(args[:n], " ")
		for _, c := range commands {
			if c.name == name {
				return c, args[n:], true
			}
		}
	}
	return command{}, nil, false
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "ryu builds and helps write the ryuguma site.")
	group := ""
	for _, c := range commands {
		if c.group != group {
			group = c.group
			fmt.Fprintf(w, "\n%s:\n", group)
		}
		fmt.Fprintf(w, "  %-40s %s\n", c.usage, c.summary)
	}
	fmt.Fprintln(w, "\nAnything left off is asked for. Run ryu <command> -h for details.")
	fmt.Fprintln(w, "ryu works from anywhere inside the repo.")
}

// newFlags makes a flag set whose -h output shows the command's usage.
func newFlags(c command) *flag.FlagSet {
	fs := flag.NewFlagSet("ryu "+c.name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s\n\n%s.\n\n", c.usage, strings.ToUpper(c.summary[:1])+c.summary[1:])
		fs.PrintDefaults()
	}
	return fs
}

// findProject walks up from the working directory to the folder holding
// site.yaml.
func findProject() (*project, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, configFile)); err == nil {
			return &project{root: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("no %s found here or in any parent directory; run ryu inside the site repo", configFile)
		}
		dir = parent
	}
}
