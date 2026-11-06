package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	miseintl "github.com/joeblew999/mise-intl"
	"github.com/joeblew999/mise-intl/internal/catalog"
	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl"
	"github.com/joeblew999/mise-intl/intl/config"
)

// template is go tool intl template: the template's little app (miseintl.App: a page of intl's formats in plain Go,
// its catalogs in English and German, its tasks, its README) written into a repo with no app yet (no main.go), so it
// runs straight away. It never writes over a file of the repo's own: a catalog only while it has no message (init's
// starter), and German goes into intl.toml only when the repo doesn't ship it. The app's import of its messages
// becomes the repo's own package. Then gen.
func template(args []string) {
	c := app()
	name := filepath.Base(c.Root)
	if exists(filepath.Join(c.Root, "README.md")) {
		fmt.Println("✓ README.md is the repo's own: left as it is")
	} else {
		b, _ := fs.ReadFile(miseintl.App, "README.md")
		_, err := writeChanged(c.Root, filepath.Join(c.Root, "README.md"), bytes.ReplaceAll(b, []byte(miseintl.AppName), []byte(name)))
		must(err)
	}
	if exists(filepath.Join(c.Root, "main.go")) {
		fmt.Println("✓ the repo has a Go program of its own (main.go): none of the template's app written")
		return
	}
	if !slices.ContainsFunc(c.Locales, func(l string) bool { return strings.EqualFold(l, "de") }) {
		stop := startIntl()
		name := intl.NewLocale("de").Name()
		stop()
		must(catalog.AddLocale(c, "de", name))
		fmt.Println("added de, " + name + ", to intl.toml: the template's app is in English and German")
		var err error
		c, err = config.Find(c.Root)
		must(err)
	}
	messages := messagesPackage(c)
	for _, f := range []string{"main.go", "main_test.go", "mise.toml", "locales/en.toml", "locales/de.toml"} {
		b, _ := fs.ReadFile(miseintl.App, f)
		to := filepath.Join(c.Root, filepath.FromSlash(f))
		if dir, base := filepath.Split(f); dir == "locales/" {
			to = filepath.Join(c.CatalogDir(), base)
		}
		switch {
		case f == "main.go":
			b = bytes.ReplaceAll(b, []byte(`"`+miseintl.AppMessages+`"`), []byte(`"`+messages+`"`))
		case f == "mise.toml" && exists(to):
			fmt.Printf("✓ mise.toml is the repo's own: left as it is. The template's app runs and tests with these:\n\n%s\n", b)
			continue
		case strings.HasSuffix(f, ".toml") && exists(to) && !starter(to):
			fmt.Printf("✓ %s has messages of the repo's own: left as it is\n", to)
			continue
		}
		_, err := writeChanged(c.Root, to, b)
		must(err)
	}
	trustConfig(c.Root, "mise.toml")
	must(read(c).Generate(os.Stderr))
	fmt.Println("✓ the template's app, in English and German: next, go tool intl (or mise run intl), then go run . (mise run serve)")
}

// messagesPackage is the import path of the repo's messages package, its catalogs' folder in its module.
func messagesPackage(c config.Config) string {
	gomod := setup.GoMod(c.Root)
	b, err := os.ReadFile(gomod)
	must(err)
	module := ""
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == "module" {
			module = strings.Trim(f[1], `"`)
		}
	}
	rel, err := filepath.Rel(filepath.Dir(gomod), c.CatalogDir())
	must(err)
	if rel == "." {
		return module
	}
	return module + "/" + filepath.ToSlash(rel)
}

// starter reports whether a catalog has no message yet: only comments and blank lines, as init writes one.
func starter(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	for _, line := range strings.Split(string(b), "\n") {
		if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "#") {
			return false
		}
	}
	return true
}
