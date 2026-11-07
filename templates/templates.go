// Package templates is mise-intl's template, embedded for mise-intl: a whole repo that runs where it is, in two
// parts: the tool's files, which mise-intl init writes into any repo (the numbered tasks and
// the mise config they run with (Go, from go.mod), the GitHub workflow, the Claude Code hook and skill, the docs, the .gitignore), and the
// template's little app, which mise-intl template writes into a repo with no app yet (a page of intl's formats in
// plain Go in app/, its catalogs in English and German, its tasks, the package.json that pins its workerd, its README). This file is the template's only Go at
// its top, as mise-cloudflare's templates.go is; the app's is a package of its own, in app/.
package templates

import (
	"embed"
	"io/fs"
)

//go:embed .mise/tasks/mise-intl.toml .mise/config.toml
//go:embed .github/workflows/intl.yml
//go:embed .claude/hooks/mise-intl-session-start.sh .claude/settings.json .claude/skills
//go:embed docs/intl
//go:embed .gitignore
var tool embed.FS

// Tool is the template's tool files: what mise-intl init writes into a repo, at their paths there.
var Tool fs.FS = tool

//go:embed app/main.go app/server.go app/worker.go app/main_test.go mise.toml pitchfork.toml package.json pnpm-lock.yaml README.md
//go:embed locales/en.toml locales/de.toml
var app embed.FS

// App is the template's little app: what mise-intl template writes into a repo with no app yet, at its paths
// there.
var App fs.FS = app

// AppName is the template's name in its README.md and package.json: a repo gets its own instead, its folder's.
const AppName = "intl-template"

// AppMessages is the template's app's messages package, which mise-intl template writes as the repo's own.
const AppMessages = "github.com/joeblew999/mise-intl/templates/locales"

// HookFile is the Claude Code hook's name in a repo's .claude/hooks.
const HookFile = "mise-intl-session-start.sh"
