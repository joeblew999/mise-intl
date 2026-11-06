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

## Use it in your repo

Step by step, at the repository's root (GitHub and Claude Code read `.github/` and `.claude/` only there), in a repo
that's new or has a Go app of its own:

1. **Install Go**, once on each machine: [Go's download page](https://go.dev/dl/) (`brew install go`, or `mise use -g
   go`). From then on the repo's `go.mod` names its Go, and any Go from 1.21 on downloads that one. A new repo needs a
   Go module first: `go mod init example.com/myapp`.
2. **intl's files, always:**

   ```sh
   go run github.com/joeblew999/mise-intl/cmd/intl@latest init
   ```

   It runs the newest release, or the one you name instead of `latest` (`@v0.1.2`: they're all on
   [its tags](https://github.com/joeblew999/mise-intl/tags)), built from source. It pins itself first: intl's tool line
   in `go.mod` at exactly that release, so from then on it's `go tool intl`, at that version, and the app's Go imports
   the same one. Then it writes intl's tasks (`.mise/tasks/intl.toml`), `intl.toml` with the source catalog
   (`--locales en,de` for more languages), its GitHub workflow, Claude Code's hook and skill, and its docs.

3. **The template's app, only in a repo with no app yet:**

   ```sh
   go tool intl template
   ```

   It writes the template's little app, `app/`: one page of intl's formats in plain Go, in English and German, with
   its catalogs and a README. It never writes over a file: a repo with an app of its own gets none of it.

4. **Then:**

   ```sh
   go tool intl
   ```

   It shows what's done, and what to run next. With mise, each command is a task too: `mise run intl`, and
   `mise run intl:template` for step 3.

Commit what steps 1 to 3 wrote. `go run ./app` serves the template's app at http://localhost:8080, `/de/` in German.

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
two parts: the tool's files, which `go tool intl init` writes into any repo (the numbered tasks, the GitHub workflow, Claude Code's hook and skill, the docs), and the little app, which `go tool intl template` writes
into a repo with no app yet (one page in plain Go, in `app/`: `net/http`, `html/template`, `intl.Site`, `intl.Text`).
[templates/templates.go](templates/templates.go) embeds both, the template's only Go at its top, as mise-cloudflare's
is.

Its walk, [mise-check](https://github.com/joeblew999/mise-check)'s `walk` ([walk_test.go](walk_test.go)), runs the
template's numbered tasks in their order, through its own mise file, each checked: on the template where it is, and on
a new repo made exactly as [Use it in your repo](#use-it-in-your-repo) says. `init` writes exactly the template; `check` and the workflow's `verify` pass;
the app answers in English and in German, each value as that language's Intl writes it; a language added, exported to
a translator and imported back; a changed source message confirmed; a key nobody uses pruned.

## This repo

`mise.toml` here is only for working on mise-intl: a repo that uses intl never gets it.

```sh
mise run tool:test             # go vet, and every Go test
mise run root:check-mise       # misecheck on mise.toml
mise run template:check-mise   # misecheck on the template's tasks, against go tool intl's Go and the walk
mise run template:e2e          # the walk on the template
mise run new-repo:e2e          # the walk on a new repo, made the README's way (--version v0.1.2: at that release)
mise run tool:release          # a release: the commit's tag or the next patch, the Go proxy, a new repo walked at it
```

CI runs all of them on Linux, macOS and Windows. A merge to `main` that changes intl releases the next patch
(`release.yml`, `mise run tool:release`), and every release is walked again, a new repo made the README's way at
that release from the Go proxy, on all three (`new-repo.yml`). intl came from
[go-htmx4](https://github.com/joeblew999/go-htmx4), which uses it: #1 says what moved and what stayed there.

## License

MIT
