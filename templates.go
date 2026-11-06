// Package miseintl is mise-intl's template, embedded for go tool intl. The template, templates/, is a whole repo that
// runs where it is, in two parts: the tool's files, which go tool intl init writes into any repo (the numbered tasks,
// Go's version for mise, the GitHub workflow, the Claude Code hook and skill, the docs, the .gitignore), and the
// template's little app, which go tool intl template writes into a repo with no app yet (a page of intl's formats in
// plain Go, its catalogs in English and German, its tasks, its README). The template is a folder of this module, with
// no go.mod of its own, so this file, which embeds it, is here, one level up.
package miseintl

import (
	"embed"
	"io/fs"
)

//go:embed templates/.mise/tasks/intl.toml templates/.mise/config.toml
//go:embed templates/.github/workflows/intl.yml
//go:embed templates/.claude/hooks/intl-session-start.sh templates/.claude/settings.json templates/.claude/skills
//go:embed templates/docs/intl
//go:embed templates/.gitignore
var tool embed.FS

// Tool is the template's tool files: what go tool intl init writes into a repo, at their paths there.
var Tool = sub(tool)

//go:embed templates/main.go templates/main_test.go templates/mise.toml templates/README.md
//go:embed templates/locales/en.toml templates/locales/de.toml
var app embed.FS

// App is the template's little app: what go tool intl template writes into a repo with no app yet, at its paths
// there.
var App = sub(app)

// AppName is the template's name in its README.md: a repo gets its own instead, its folder's.
const AppName = "intl-template"

// AppMessages is the template's app's messages package, which go tool intl template writes as the repo's own.
const AppMessages = "github.com/joeblew999/mise-intl/templates/locales"

// HookFile is the Claude Code hook's name in a repo's .claude/hooks.
const HookFile = "intl-session-start.sh"

func sub(f embed.FS) fs.FS {
	s, err := fs.Sub(f, "templates")
	if err != nil {
		panic(err)
	}
	return s
}
