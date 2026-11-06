# mise-intl

[![ci](https://github.com/joeblew999/mise-intl/actions/workflows/ci.yml/badge.svg)](https://github.com/joeblew999/mise-intl/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/mise-intl.svg)](https://pkg.go.dev/github.com/joeblew999/mise-intl)

Every language a Go app ships, with [gsx](https://github.com/gsxhq/gsx) or without: messages in ICU MessageFormat
catalogs, and every date, number, plural, list and name written by the runtime's own `Intl` (V8's, a Cloudflare
Worker's or workerd's), in the reader's language, time zone and clock. Translating, by Claude Code or by people, is a
command, and so is checking all of it in CI.

> **A repo that uses intl needs only this Go module. There is no binary.** One line of its `go.mod` pins everything:
> `go get -tool github.com/joeblew999/mise-intl/cmd/intl@latest`. The command is `go tool intl`, which Go builds from
> this module's source at the version `go.mod` pins, checked by `go.sum`. Off Cloudflare, intl fetches the workerd
> release it pins and checks its SHA-256 itself. mise is optional: with it, each command is a numbered task too.

## Start a repo

```sh
go mod init myapp
go get -tool github.com/joeblew999/mise-intl/cmd/intl@latest
go tool intl init        # intl's files: tasks, intl.toml, locales/en.toml, skill, docs, workflow, Claude Code hook
go tool intl template    # the little app, only in a repo with no app yet: a page of intl's formats, English and German
go tool intl             # what's done, and what to run next (or: mise run intl)
go run .                 # http://localhost:8080, and /de/ in German
```

`init` writes `docs/intl/README.md` into the repo: everything about intl there, every command, gsx or not, the
screens, translating. It is [templates/docs/intl/README.md](templates/docs/intl/README.md) here, as a repo has it.

## The packages

| Package (`github.com/joeblew999/mise-intl/…`) | For | Imports gsx? |
|---|---|---|
| `intl` | every app: messages, languages, the middleware (`intl.Site`), `intl.Text` and the formatting calls | no |
| `intl/workerd` | apps not on Cloudflare: Intl from workerd | no |
| `intl/intltest` | apps' tests | no |
| `intl/config` | tools that read an app's `intl.toml` | no |
| `gsxintl` | gsx apps: `Head`, `HTML`, `Attrs` and the renderer `gsx.toml` names | **yes, the only one** |
| `gsxintl/filters` | gsx apps' `gsx.toml`: `date`, `number`, `since`, … | no |
| `gsxintl/registry`, `gsxintl/screens` | `go tool intl add`: intl's screens (language list, language offer, preferences), and any module's gsx components | no |
| `cmd/intl` | `go tool intl` | no |

An app without gsx never gets gsx in its `go.mod`, not even through `go tool intl` (`TestWithoutGsx`).

## The template

[templates/](templates/) is a whole repo that runs where it is, for testing this one and for others to start from, in
two parts: the tool's files, which `go tool intl init` writes into any repo (the numbered tasks, Go's version for mise,
the GitHub workflow, Claude Code's hook and skill, the docs), and the little app, which `go tool intl template` writes
into a repo with no app yet (one page in plain Go: `net/http`, `html/template`, `intl.Site`, `intl.Text`).
[templates.go](templates.go) embeds both.

Its walk, [mise-check](https://github.com/joeblew999/mise-check)'s `walk` ([walk_test.go](walk_test.go)), runs the
template's numbered tasks in their order, through its own mise file, each checked: on the template where it is, and on
a new repo made from nothing as above. `init` writes exactly the template; `check` and the workflow's `verify` pass;
the app answers in English and in German, each value as that language's Intl writes it; a language added, exported to
a translator and imported back; a changed source message confirmed; a key nobody uses pruned.

## This repo

`mise.toml` here is only for working on mise-intl: a repo that uses intl never gets it.

```sh
mise run tool:test             # go vet, and every Go test
mise run root:check-mise       # misecheck on mise.toml
mise run template:check-mise   # misecheck on the template's tasks, against go tool intl's Go and the walk
mise run template:e2e          # the walk on the template
mise run new-repo:e2e          # the walk on a new repo (--version v0.1.0: at that release)
```

CI runs all of them on Linux, macOS and Windows. intl came from
[go-htmx4](https://github.com/joeblew999/go-htmx4), which uses it: #1 says what moved and what stayed there.

## License

MIT
