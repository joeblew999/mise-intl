package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/joeblew999/mise-check/toolfiles"
	"github.com/joeblew999/mise-intl/internal/catalog"
	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl/config"
	"github.com/joeblew999/mise-intl/intl/workerd"
	"github.com/joeblew999/mise-intl/templates"
)

// template is mise-intl template: the template's little app (templates.App: a page of intl's formats in plain Go
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
	if setup.GoMod(c.Root) == "" {
		// A new repo's go.mod, so it needs no Go before mise brings it: the module named for the folder, the Go this
		// build is, and intl's packages at this release (a checkout's build leaves them to go mod tidy, or the repo's
		// replace).
		gomod := "module " + npmName(name) + "\n\ngo " + strings.TrimPrefix(runtime.Version(), "go") + "\n"
		if version != "" {
			gomod += "\nrequire " + intlModule + " v" + buildVersion() + "\n"
		}
		_, err := writeChanged(c.Root, filepath.Join(c.Root, "go.mod"), []byte(gomod))
		must(err)
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
	for _, f := range []string{"app/main.go", "app/server.go", "app/worker.go", "app/main_test.go", "mise.toml", "pitchfork.toml", "package.json", "pnpm-lock.yaml", "locales/en.toml", "locales/de.toml"} {
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
		case (f == "mise.toml" || f == "pitchfork.toml") && exists(to):
			fmt.Printf("✓ %s is the repo's own: left as it is. The template's app runs and tests with these:\n\n%s\n", f, b)
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
	fmt.Println("✓ the template's app, in English and German: next, mise run intl, then mise run serve")
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
// the mise.toml template wrote, then the repo's own setup task installs package.json's workerd (mise installs Go, node
// and aube for it first), then go mod tidy puts the app's imports in go.mod and go.sum. Without a setup task, it says
// how instead.
func setUpApp(root string) {
	toolfiles.Trust(root, "mise.toml", nil)
	has := exec.Command("mise", "tasks", "info", "setup")
	has.Dir = root
	if has.Run() != nil {
		fmt.Println("the repo's mise.toml has no setup task: install workerd its own way (package.json pins it), and go mod tidy, then mise run intl")
		return
	}
	fmt.Println("$ mise run setup")
	cmd := exec.Command("mise", "run", "setup")
	cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = root, os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fail(fmt.Errorf("mise run setup: %w", err))
	}
	// The app's imports in its go.mod and go.sum, intl's packages among them: the app is new, so they're the template's.
	fmt.Println("$ go mod tidy")
	tidy := goCommand(root, "mod", "tidy")
	tidy.Stdout, tidy.Stderr = os.Stdout, os.Stderr
	if err := tidy.Run(); err != nil {
		fail(fmt.Errorf("go mod tidy: %w", err))
	}
}
