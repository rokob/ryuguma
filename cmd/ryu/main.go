// Command ryu builds, previews, and helps write the ryuguma site.
// Run it with no arguments for a list of commands.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type command struct {
	name    string
	group   string
	summary string
	usage   string
	run     func(p *project, fs *flag.FlagSet, args []string) error
}

// Commands are listed in help in this order.
var commands = []command{
	{name: "serve", group: "Site", summary: "preview locally with drafts, rebuilding on change", usage: "ryu serve [-addr :8080] [-drafts=true]", run: runServe},
	{name: "build", group: "Site", summary: "write the site to dist/", usage: "ryu build [-out dist] [-drafts]", run: runBuild},
	{name: "check", group: "Site", summary: "validate content, references, and templates", usage: "ryu check", run: runCheck},
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
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

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printHelp(stdout)
		return nil
	}
	for _, c := range commands {
		if c.name == args[0] {
			p, err := findProject()
			if err != nil {
				return err
			}
			return c.run(p, newFlags(c), args[1:])
		}
	}
	return usageError(fmt.Sprintf("ryu: unknown command %q (run ryu with no arguments for help)", args[0]))
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
	fmt.Fprintln(w, "\nRun ryu <command> -h for details. ryu works from anywhere inside the repo.")
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
