package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeblew999/mise-intl/internal/catalog"
	"github.com/joeblew999/mise-intl/intl/config"
)

// report is mise-intl show --json: where the repo is, and what to run next.
type report struct {
	Version   string   `json:"version"`   // this mise-intl: the release .mise/config.toml pins
	Locales   []string `json:"locales"`   // the shipped languages, the source first
	Generated bool     `json:"generated"` // the messages package is the catalogs' (gen has run since they changed)
	App       bool     `json:"app"`       // the repo has a Go program of its own (main.go, or the template's app/main.go)
	Left      int      `json:"left"`      // messages missing or stale, in every language
	Problems  []string `json:"problems"`  // intl's files that are out of date, or the catalogs that don't read
	Next      string   `json:"next"`      // the task to run next
	Command   string   `json:"command"`   // the same without mise
}

// show is mise-intl show, and mise-intl with no command (mise run intl): the steps, which are done for
// this repo, and the next one. It asks Intl nothing, so it's quick.
func show(args []string) {
	f := flag.NewFlagSet("show", flag.ExitOnError)
	jsonOut := f.Bool("json", false, "print one JSON document on stdout instead of text")
	f.Parse(args)
	r := report{Version: buildVersion(), Locales: []string{}, Problems: []string{}}
	c, err := config.Find(".")
	switch {
	case err != nil:
		r.Problems = append(r.Problems, "no intl.toml: this repo isn't set up for intl yet")
		r.Next, r.Command = "mise-intl init", "mise-intl init"
	default:
		r.Locales = c.Locales
		r.Problems = append(r.Problems, stale(c.Root)...)
		r.App = hasApp(c.Root)
		p, err := catalog.Read(c)
		if err != nil {
			r.Problems = append(r.Problems, err.Error())
		} else {
			r.Generated = p.SumCurrent() && exists(filepath.Join(c.CatalogDir(), "intl_gen.go"))
			for _, l := range p.Locales[1:] {
				r.Left += l.Count(catalog.Missing) + l.Count(catalog.Stale)
			}
		}
		switch {
		case anyHas(r.Problems, "mise-intl init"):
			r.Next, r.Command = "mise run intl:init", "mise-intl init"
		case len(r.Problems) > 0:
			r.Next, r.Command = "mise run intl:check", "mise-intl check"
		case !r.Generated:
			r.Next, r.Command = "mise run intl:gen", "mise-intl gen"
		case !r.App:
			r.Next, r.Command = "mise run intl:template", "mise-intl template"
		case r.Left > 0:
			r.Next, r.Command = "mise run intl:todo", "mise-intl todo"
		default: // every language complete: another, or a message changed then gen
			r.Next, r.Command = "mise run intl:add-locale", "mise-intl add-locale"
		}
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		must(enc.Encode(r))
		return
	}
	mark := map[bool]string{true: "✓", false: "·"}
	fmt.Printf("mise-intl %s; languages: %v\n", r.Version, r.Locales)
	fmt.Printf("%s 03-init   intl's files, as this version writes them\n", mark[!anyHas(r.Problems, "mise-intl init") && len(r.Locales) > 0])
	fmt.Printf("%s 06-gen    the messages package is the catalogs'\n", mark[r.Generated])
	fmt.Printf("%s app       a Go program of the repo's own (or the template's: mise run intl:template)\n", mark[r.App])
	fmt.Printf("%s 09-todo   every language translated (%d messages left)\n", mark[r.Left == 0 && len(r.Locales) > 0], r.Left)
	for _, p := range r.Problems {
		fmt.Println("  FAIL " + p)
	}
	fmt.Printf("next: %s (without mise: %s)\n", r.Next, r.Command)
}

// anyHas reports whether any of problems says s.
func anyHas(problems []string, s string) bool {
	for _, p := range problems {
		if strings.Contains(p, s) {
			return true
		}
	}
	return false
}
