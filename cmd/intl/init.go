package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl/config"
	"github.com/joeblew999/mise-intl/templates"
)

// theirs are the template's files a repo may have of its own already: init writes them only when it has none.
var theirs = map[string]bool{".gitignore": true, ".claude/settings.json": true, ".mise/config.toml": true}

// initRepo is go tool intl init (03): the repo set up for intl, in its Go module. First the template's tool files
// (templates.Tool: the numbered tasks, Go's version for mise, the GitHub workflow, the Claude Code hook and skill, the
// docs), each at its path; then the repo's own (setup.Init): intl.toml with its languages (--locales en,de), the
// catalogs (--catalogs locales), and in a gsx app gsx.toml's filters and renderers; then gen. Every step leaves alone
// what is the repo's, so it runs again after upgrading: the files that are intl's are written as this version writes
// them.
func initRepo(args []string) {
	f := flag.NewFlagSet("init", flag.ExitOnError)
	locales := f.String("locales", "en", "BCP 47 ids, comma-separated, the source first")
	catalogs := f.String("catalogs", "locales", "the catalogs' folder")
	gsxui := f.String("gsxui", "", "in a gsx app: where gsxui add puts its components (default: gsxui.json's ui)")
	f.Parse(args)
	root, err := os.Getwd()
	must(err)
	o := setup.Options{Locales: strings.Split(*locales, ","), Catalogs: *catalogs, Gsxui: *gsxui}
	if c, err := config.Find(root); err == nil && c.Root == root {
		o = setup.Options{Locales: c.Locales, Catalogs: c.Catalogs, Gsxui: c.Gsxui} // already set up: keep its choices
	}
	if o.Gsxui == "" {
		o.Gsxui = setup.DetectGsxui(root) // gsxui set up since: intl.toml gets it
	}
	writeTool(root)
	must(setup.Init(root, o, os.Stdout))
	trustConfig(root, ".mise/config.toml")
	c, err := config.Find(root)
	must(err)
	must(read(c).Generate(os.Stderr))
	fmt.Println("✓ next: go tool intl (or mise run intl): what's done, and what to run next")
}

// writeTool writes the template's tool files into the repo at root, each at its path there, but a file that is the
// repo's own.
func writeTool(root string) {
	must(fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		to := filepath.Join(root, filepath.FromSlash(p))
		if theirs[p] && exists(to) {
			return nil
		}
		b, err := fs.ReadFile(templates.Tool, p)
		if err != nil {
			return err
		}
		if _, err := writeChanged(root, to, b); err != nil {
			return err
		}
		if strings.HasSuffix(p, ".sh") {
			return os.Chmod(to, 0o755)
		}
		return nil
	}))
}

// stale names the files init wrote that this version would write differently, and a .claude/settings.json of the
// repo's own that doesn't run the hook.
func stale(root string) []string {
	var out []string
	fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || theirs[p] {
			return err
		}
		want, _ := fs.ReadFile(templates.Tool, p)
		// Line endings aside: a Windows checkout of the repo may have \r\n where init wrote \n.
		crlf := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		switch {
		case err != nil:
			out = append(out, p+" is missing: run go tool intl init")
		case !bytes.Equal(crlf(b), crlf(want)):
			out = append(out, p+" is out of date: run go tool intl init")
		}
		return nil
	})
	if b, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json")); err == nil && !bytes.Contains(b, []byte(templates.HookFile)) {
		out = append(out, ".claude/settings.json doesn't run .claude/hooks/"+templates.HookFile+": add its SessionStart entry (docs/intl/README.md)")
	}
	return out
}

// trustConfig has mise trust a config file intl wrote, so its tasks run straight away, when mise is there: a repo
// without mise needs none of it.
func trustConfig(root, file string) {
	path := filepath.Join(root, filepath.FromSlash(file))
	if !exists(path) {
		return
	}
	if _, err := exec.LookPath("mise"); err != nil {
		return
	}
	if out, err := exec.Command("mise", "trust", "-q", path).CombinedOutput(); err != nil {
		fmt.Printf("mise didn't trust %s (%v %s): run mise trust %s\n", file, err, strings.TrimSpace(string(out)), file)
	}
}
