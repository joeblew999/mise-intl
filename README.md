# mise-intl

[![ci](https://github.com/joeblew999/mise-intl/actions/workflows/ci.yml/badge.svg)](https://github.com/joeblew999/mise-intl/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/joeblew999/mise-intl)](https://github.com/joeblew999/mise-intl/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/joeblew999/mise-intl)](go.mod)
[![license](https://img.shields.io/github/license/joeblew999/mise-intl)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/mise-intl.svg)](https://pkg.go.dev/github.com/joeblew999/mise-intl)

**The problem.** Shipping a Go app in more than one language properly takes a lot of fiddly pieces, and each repo ends
up doing them a little differently, or not at all:

- messages in a format translators and tools know (ICU MessageFormat), with real plural rules for every language;
- every date, number, list, unit and language name written as the reader's language writes it, in their time zone and
  clock, the same on the server, in tests and in the browser;
- translating what's new or changed, by Claude Code or by people (XLIFF), and knowing what's missing or stale;
- CI that fails on a missing translation on `main`, but only warns on a contributor's pull request;
- all of that kept the same on every machine, in CI and in a Claude Code cloud session.

**What mise-intl does.** It does all of that through [mise](https://mise.jdx.dev), for any Go app, with
[gsx](https://github.com/gsxhq/gsx) or without: messages in catalogs, compiled into the app's own Go package, and every
value formatted by the runtime's own `Intl` (V8's: a Cloudflare Worker's, or workerd's off Cloudflare). One command adds
it to a repo, and from then on every step is a mise task, numbered in the order you need them. Each one says what it
did and what to run next, and `mise run intl` shows where the repo is at any time. Every version is pinned: mise-intl
in `.mise/config.toml`, workerd in `package.json`. CI checks it on macOS, Linux and Windows, and it sets itself up in a
Claude Code cloud session.

## Use it in your repo

How to use intl, from setting up an app to every command, gsx or not, its screens and translating, is one page:
[templates/docs/intl/README.md](templates/docs/intl/README.md), which `init` writes into your repo as
`docs/intl/README.md`. To start, at the repository's root, with [mise](https://mise.jdx.dev/getting-started.html)
installed:

```sh
mise exec github:joeblew999/mise-intl@latest -- mise-intl init
mise run intl:template   # only in a repo with no app yet: the template's little app
mise run intl            # what's done, and what to run next
```

## What the app imports

The command is the binary; the app's Go imports intl's packages, from this module at the same release (`init` and
`intl:template` require it in `go.mod`):

| Package (`github.com/joeblew999/mise-intl/…`) | For | Imports gsx? |
|---|---|---|
| `intl` | every app: messages, languages, the middleware (`intl.Site`), `intl.Text` and the formatting calls | no |
| `intl/workerd` | apps not on Cloudflare: Intl from workerd, the repo's intl daemon (pitchfork runs it) | no |
| `intl/intltest` | apps' tests | no |
| `intl/config` | tools that read an app's `intl.toml` | no |
| `gsxintl` | gsx apps: `Head`, `HTML`, `Attrs` and the renderer `gsx.toml` names | **yes, the only one** |
| `gsxintl/filters` | gsx apps' `gsx.toml`: `date`, `number`, `since`, … | no |
| `gsxintl/registry`, `gsxintl/screens` | `mise-intl add`: intl's screens (language list, language offer, preferences), and any module's gsx components | no |

An app without gsx never gets gsx in its `go.mod` (`TestWithoutGsx`).

## Working on mise-intl

This repo is two things, kept apart:

| | What | Where |
|---|---|---|
| **For other repos** | The `mise-intl` binary, the packages an app imports, and the template: a whole repo you can drop anywhere and run, whose tool half `init` writes into a repo (its tasks and their mise config, its check workflow, its Claude Code hook and skill, its docs, its `.gitignore`) | [templates/](templates/), [cmd/mise-intl/](cmd/mise-intl/), [intl/](intl/), [gsxintl/](gsxintl/), [internal/](internal/) |
| **For this repo** | The tasks for working on it, its tests, the release, its CI. This README | [mise.toml](mise.toml), the `*_test.go` at the top, [.github/workflows/](.github/workflows/) |

Nothing for this repo is ever in `templates/`, and nothing in `templates/` is this repo's own. The Go says it too:
`cmd/mise-intl` is the one entry point that ships, with the packages; everything this repo runs for itself in Go is a
test at the top (the `*_test.go`, package `e2e`), which the root's tasks run.

Everything is driven from this repo's [mise.toml](mise.toml), run at the top, and each task runs what it needs first:
run the furthest you need, here, never waiting on GitHub. In this checkout the template's `.mise/config.toml` says
`latest`, and `mise run tool:install` overrides it on this machine only, with this checkout's build.

### Rules

- Intl is the one source of language: messages, plurals, numbers, dates, names and direction all come from the
  runtime's `Intl` (`internal/js`), never from Go. Off Cloudflare it is workerd's, the one `package.json` pins, which
  pitchfork runs as the repo's intl daemon (`pitchfork.toml`); this repo's tests use the template's
  (`INTL_PITCHFORK_DIR`, in `mise.toml`).
- intl's packages never import gsx, but `gsxintl`: an app without gsx gets none.
- Every tool is pinned, and comes from mise: mise-intl in `.mise/config.toml`, pitchfork in `mise.toml`, workerd in `package.json` (aube), Go
  in `go.mod`. No Node script of ours, and no `npm`.
- Test the real thing: workerd's Intl, the real catalogs, the template's app served and read, never a fake.

### Tasks

Every task is named noun:verb, what it acts on and then what it does, as a repo's own are (`intl:06-gen`). `mise
tasks` lists each with what it does.

| Task | What | When |
|---|---|---|
| `template:setup` | The template's own setup: its workerd, which this repo's tests run Intl on | Before `tool:test`, once |
| `template:gen` | The template's messages package, which this module's build needs | Before `tool:test` |
| `template:e2e` | The walk on the template, where it is | After every change |
| `template:worker` | `TestWorker`: the template's app on a Cloudflare Worker (Go for js/wasm, workers-go's glue, intl's Worker modules) in local workerd, `/` and `/de/` as the Go server answers | After a change to intl's Worker side |
| `template:check-mise` | `TestTemplateMise` on the template's tasks file, the tool's Go, the walk and the template's skill, which names every task and flag | After a change to a task, a command or the skill |
| `new-repo:e2e` | The walk on a new repo (`--version` pins a release) | After every change |
| `tool:build` | This checkout's build, `bin/mise-intl` | Any task needs it |
| `tool:install` | That build as the mise version `local`, for any repo on this machine (`mise use github:joeblew999/mise-intl@local` there) and the template's (`templates/.mise/config.local.toml`, never committed) | To try a change in another repo, never by waiting on a release |
| `tool:test` | `go vet`, the Go tests, the release config | After every change |
| `tool:release` | A release from the commit checked out, as `release.yml` runs it; `--dry-run` tags and publishes nothing | Never by hand: a merge to main runs it |
| `root:check-mise` | `TestRootMise` on this repo's own `mise.toml`, its mise side: one line each, the Go it runs there, `usage` passed by name, no POSIX shell syntax, so every task runs on Windows too | After a change to this repo's tasks |

This repo's pins, and the rules on its mise files, are [mise-check](https://github.com/joeblew999/mise-check)'s, through
its own tasks, which `mise-check init` wrote into `.mise/` (`.mise/config.toml` pins it, at the release `go.mod`
requires its packages at): `mise run mise-check` says what's done and what to run next, and every task is
[docs/mise-check/README.md](docs/mise-check/README.md). Pins move only by `mise run mise-check:update`, then
`tool:test` and the walks prove it.

### Tests

**The walk.** A repo's steps are its own numbered tasks, `intl:03-init` to `intl:14-prune`
([templates/.mise/tasks/mise-intl.toml](templates/.mise/tasks/mise-intl.toml)), after steps 00 to 02 (the repo's git,
`mise install`, its own `setup`). One name all the way through: `intl:06-gen` runs `mise-intl gen`, which is
`main.go`'s `case "gen"` calling `gen` in `gen.go`. The walk tests (`TestWalkTemplate`, `TestWalkNewRepo`) walk them in their order, through the repo's own mise
file, as a person takes them, each step with its check: `init` writes exactly the template; `check` and the
workflow's `verify` pass, and a pull request only warns where a push fails; the app answers in English and in German,
each value as that language's Intl writes it; a language added, exported to a translator and imported back; a changed
source message confirmed; a key nobody uses pruned. It walks two kinds of repo:

- **The template**, [templates/](templates/), in place. The steps that change the catalogs run only on a new repo.
- **A new repo**, made as a person makes one: an empty folder (`intl-new`) with its own git, `mise-intl init`, then
  its own `intl:template`, then `mise run intl`. The walk makes it in a folder of its own, gone when the walk ends. Its
  `init` runs at this checkout's build (`local`, with its `go.mod` using this checkout's packages), or at a release
  with `--version`, as the release tries it, and pins itself in `.mise/config.toml`.

Each walk runs in the repo on that repo's own mise only (`MISE_CEILING_PATHS`).

**The mise ↔ Go checker**, [mise_test.go](mise_test.go)'s `TestRootMise` and `TestTemplateMise` check a mise file against the Go it runs, both
ways. Each task's command is a command's file in `cmd/mise-intl`, and each command a task's; each task the walk names
is one; every task declares its flags and arguments with mise's `usage` and passes them by name. `root:check-mise`
points it at this repo's own `mise.toml`, its mise side only: the root runs Go, it isn't a binary's tasks.

### Layout

| Path | Whose | What |
|---|---|---|
| [templates/](templates/) | other repos' | the template: a sample app plus the tool's half, which `init` writes into a repo at the same paths (the tasks and their mise config, the workflow, the Claude Code hook and settings, the agent skill, the docs, the `.gitignore`); [templates.go](templates/templates.go) lists both halves |
| [cmd/mise-intl/](cmd/mise-intl/) | other repos' | the command: a file per command, `x.go` holding `x`, which `main.go`'s `case "x"` calls |
| [intl/](intl/), [gsxintl/](gsxintl/) | other repos' | the packages an app imports (above) |
| [internal/](internal/) | other repos' | the catalogs (`catalog`), a repo's setup (`setup`), intl's JS on Intl (`js`), the badge (`badge`) |
| [e2e_walk_test.go](e2e_walk_test.go), [e2e_steps_test.go](e2e_steps_test.go), [e2e_mise_test.go](e2e_mise_test.go) | this repo's | the walk, on [mise-check's `walk`](https://github.com/joeblew999/mise-check#use-walk), which runs the steps in their order and does everything that isn't intl's: `TestWalk`; the steps, each with its check; what's intl's in making a repo |
| [tool_test.go](tool_test.go) | this repo's | `TestToolRelease`, the release: only `mise run tool:release` runs it |
| [mise_test.go](mise_test.go) | this repo's | `TestRootMise` and `TestTemplateMise`: [mise-check's `misecheck`](https://github.com/joeblew999/mise-check) on a mise file of this repo and the Go it runs, both ways |
| [workerd_test.go](workerd_test.go), [gsx_test.go](gsx_test.go) | this repo's | the workerd the template pins; intl's packages without gsx |
| [worker_test.go](worker_test.go) | this repo's | `TestWorker`: the template's app on a Cloudflare Worker, in local workerd (only `mise run template:worker` runs it) |
| [.github/workflows/](.github/workflows/) | this repo's | its own workflows; other repos get theirs from `templates/` |

### CI and releases

Shell steps only, no JavaScript actions. Each workflow checks out and installs mise, then only runs the root mise, as
on any machine:

- [ci.yml](.github/workflows/ci.yml), on every push and pull request, on Linux, macOS and Windows: `template:setup`,
  `tool:test`, `template:check-mise`, `root:check-mise`, then `template:e2e`, `new-repo:e2e` and `template:worker`.
- [release.yml](.github/workflows/release.yml), on every change to the tool merged to main: `tool:release`. It takes
  the commit's `v*` tag or tags the next patch version, [goreleaser](.goreleaser.yaml) puts the `mise-intl` binary for
  macOS, Linux and Windows on the release, a new repo made pinning that release is walked (`new-repo:e2e --version`),
  and the version goes on proxy.golang.org, so `go get` and pkg.go.dev have the packages at once. So the release is
  always main.
- [new-repo.yml](.github/workflows/new-repo.yml), once a release run has passed (or by hand, for any release): a new
  repo made the README's way at that release, walked, on Linux, macOS and Windows, its tasks in each one's own shell
  (`cmd` on Windows). It runs after the release, so releases stay fast.

intl came from [go-htmx4](https://github.com/joeblew999/go-htmx4), which uses it.

This README is the one place for working on mise-intl, for people and agents alike: CLAUDE.md only loads it (`@README.md`). How to use intl is the docs page's, [templates/docs/intl/README.md](templates/docs/intl/README.md), never this README's: here, only how to start, and the link there.

## License

MIT
