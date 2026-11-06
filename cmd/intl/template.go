package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/joeblew999/mise-intl/internal/catalog"
	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl/config"
	"github.com/joeblew999/mise-intl/intl/workerd"
	"github.com/joeblew999/mise-intl/templates"
)

// template is go tool intl template: the template's little app (templates.App: a page of intl's formats in plain Go
// in app/, its catalogs in English and German, its tasks, its README) written into a repo with no app yet, so it
// runs straight away. It never writes over a file of the repo's own: a catalog only while it has no message (init's
// starter), and German goes into intl.toml only when the repo doesn't ship it. The app's import of its messages
// becomes the repo's own package. Then gen.
func template(args []string) {
	c := app()
	name := filepath.Base(c.Root)
	if exists(filepath.Join(c.Root, "README.md")) {
		fmt.Println("✓ README.md is the repo's own: left as it is")
	} else {
		b, _ := fs.ReadFile(templates.App, "README.md")
		_, err := writeChanged(c.Root, filepath.Join(c.Root, "README.md"), bytes.ReplaceAll(b, []byte(templates.AppName), []byte(name)))
		must(err)
	}
	if hasApp(c.Root) {
		fmt.Println("✓ the repo has a Go program of its own (main.go, or app/main.go): none of the template's app written")
		return
	}
	if !slices.ContainsFunc(c.Locales, func(l string) bool { return strings.EqualFold(l, "de") }) {
		// German's name is only the new catalog's first comment, which the template's de.toml replaces at once: no
		// Intl, so no workerd, before the package.json that pins it is written and installed.
		must(catalog.AddLocale(c, "de", "German"))
		fmt.Println("added de to intl.toml: the template's app is in English and German")
		var err error
		c, err = config.Find(c.Root)
		must(err)
	}
	messages := messagesPackage(c)
	ownPackage := exists(filepath.Join(c.Root, "package.json"))
	for _, f := range []string{"app/main.go", "app/main_test.go", "mise.toml", "package.json", "pnpm-lock.yaml", "locales/en.toml", "locales/de.toml"} {
		b, _ := fs.ReadFile(templates.App, f)
		to := filepath.Join(c.Root, filepath.FromSlash(f))
		if dir, base := filepath.Split(f); dir == "locales/" {
			to = filepath.Join(c.CatalogDir(), base)
		}
		switch {
		case f == "app/main.go":
			b = bytes.ReplaceAll(b, []byte(`"`+templates.AppMessages+`"`), []byte(`"`+messages+`"`))
		case f == "package.json" && ownPackage:
			fmt.Printf("✓ package.json is the repo's own: left as it is. Add workerd to it (aube add -D workerd@%s), which intl's Intl runs on\n", workerd.Version)
			continue
		case f == "pnpm-lock.yaml" && ownPackage:
			continue
		case f == "package.json":
			b = bytes.ReplaceAll(b, []byte(`"`+templates.AppName+`"`), []byte(`"`+npmName(name)+`"`))
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
	must(read(c).Generate(os.Stderr))
	setUpApp(c.Root)
	fmt.Println("✓ the template's app, in English and German: next, go tool intl (or mise run intl), then go run ./app (mise run serve)")
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

// hasApp reports whether the repo at root has a Go program of its own: a main.go at its top, or the template's,
// app/main.go.
func hasApp(root string) bool {
	return exists(filepath.Join(root, "main.go")) || exists(filepath.Join(root, "app", "main.go"))
}

// npmName is a folder's name as a package.json name: lower case, with only what npm allows.
func npmName(folder string) string {
	s := strings.Trim(regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(strings.ToLower(folder), "-"), "-._")
	if s == "" {
		return "app"
	}
	return s
}

// setUpApp installs what the template's app runs on, as mise-cloudflare's template installs its Worker's: mise trusts
// the mise.toml template wrote, mise install gets its tools (Go from go.mod, node and aube), then its setup task runs
// (aube install, which installs package.json's workerd, and go mod download).
func setUpApp(root string) {
	trustConfig(root, "mise.toml")
	if _, err := exec.LookPath("mise"); err != nil {
		fmt.Println("· no mise here: install it (https://mise.jdx.dev), then mise install and mise run setup")
		return
	}
	for _, args := range [][]string{{"install"}, {"run", "setup"}} {
		fmt.Println("$ mise " + strings.Join(args, " "))
		cmd := exec.Command("mise", args...)
		cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = root, os.Stdin, os.Stdout, os.Stderr
		must(cmd.Run())
	}
}
