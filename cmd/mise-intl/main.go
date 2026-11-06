// Command mise-intl is an app's languages, from its intl.toml (found up from the working directory). A repo gets it as
// mise-cloudflare's repos get mise-cloudflare: a binary on mise-intl's GitHub releases (goreleaser, .goreleaser.yaml),
// which mise installs at the release the repo's .mise/config.toml pins (github:joeblew999/mise-intl). The app imports
// intl's packages from the same release's Go module. Each command is a task too, numbered in the order a repo meets them
// (.mise/tasks/mise-intl.toml, which mise-intl init writes): mise run intl:<nn>-<command> runs mise-intl <command>.
//
// Each command is a file of its own, holding the function named for it (gen.go holds gen), so misecheck holds every
// task of the template to the Go it runs, both ways. mise-intl show-help lists them.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joeblew999/mise-intl/intl/config"
)

const help = `mise-intl: the app's languages, through the runtime's own Intl. Each command is a task too, with mise,
numbered in the order a repo meets them: mise run intl:<nn>-<command> runs mise-intl <command>.

The steps, in order (mise-intl, or mise run intl, says which is next):
  mise-intl init            03  intl's files: intl.toml, the catalogs, the tasks, the skill, the docs, the workflow
                                   (--locales en,de --catalogs locales --gsxui ui)
  mise-intl check           04  every problem, for CI (--missing warn; or only the locales named)
  mise-intl show            05  what's done, and what to run next (mise-intl, too; --json)
  mise-intl gen             06  the catalogs compiled into the app's messages package
  mise-intl status          07  per language: translated, inherited, stale, missing
  mise-intl add-locale      08  a language added: de, pt-BR
  mise-intl todo            09  what's left to translate (--json, for agents)
  mise-intl translate       10  Claude Code translates what's left, a worker per language
  mise-intl export          11  XLIFF for translators (--all, --out dir)
  mise-intl import          12  translators' XLIFF read back
  mise-intl confirm         13  stale translations that are still right, kept (<locale>[:<key>]…)
  mise-intl prune           14  keys the source no longer has, removed

And:
  mise-intl verify          what the repo's GitHub workflow runs: its setup and test tasks, then check
  mise-intl template        the template's little app, in a repo with no app yet: it never writes over a file
  mise-intl add             intl's screens, in a gsx app (--diff, --overwrite): language-list, language-offer, preferences
  mise-intl assets          the JavaScript a Worker and a page import (--worker dir, --browser dir)
  mise-intl locales         the shipped locales, comma-separated
  mise-intl catalogs        the catalogs' Go package name
  mise-intl markup          the app's own markup files, for gsx's checks
  mise-intl show-version    this build's version
  mise-intl show-help       this
`

func main() {
	args := os.Args[1:]
	command := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}
	switch command {
	case "", "show":
		show(args)
	case "init":
		initRepo(args)
	case "check":
		check(args)
	case "gen":
		gen(args)
	case "status":
		status(args)
	case "add-locale":
		addLocale(args)
	case "todo":
		todo(args)
	case "translate":
		translate(args)
	case "export":
		export(args)
	case "import":
		importXLIFF(args)
	case "confirm":
		confirm(args)
	case "prune":
		prune(args)
	case "verify":
		verify(args)
	case "template":
		template(args)
	case "add":
		add(args)
	case "assets":
		assets(args)
	case "locales":
		locales(args)
	case "catalogs":
		catalogs(args)
	case "markup":
		markup(args)
	case "show-version":
		showVersion()
	case "show-help":
		showHelp()
	default:
		fmt.Fprint(os.Stderr, help)
		os.Exit(2)
	}
}

// app is the app's intl.toml, found up from the working directory.
func app() config.Config {
	c, err := config.Find(".")
	if err != nil {
		fail(fmt.Errorf("%w: run mise-intl init where the app's go.mod is", err))
	}
	return c
}
