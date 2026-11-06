// Command intl is go tool intl: an app's languages, from its intl.toml (found up from the working directory). A repo
// gets it from mise-intl's Go module, with one line of its go.mod (go get -tool github.com/joeblew999/mise-intl/cmd/intl):
// Go builds it from that module's source, at the version go.mod pins. There is no binary to install, and no mise is
// needed. With mise, each command is a task too, numbered in the order a repo meets them (.mise/tasks/intl.toml, which
// go tool intl init writes): mise run intl:<nn>-<command> runs go tool intl <command>.
//
// Each command is a file of its own, holding the function named for it (gen.go holds gen), so misecheck holds every
// task of the template to the Go it runs, both ways. go tool intl show-help lists them.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joeblew999/mise-intl/intl/config"
)

const help = `go tool intl: the app's languages, through the runtime's own Intl. Each command is a task too, with mise,
numbered in the order a repo meets them: mise run intl:<nn>-<command> runs go tool intl <command>.

The steps, in order (go tool intl, or mise run intl, says which is next):
  go tool intl init            03  intl's files: intl.toml, the catalogs, the tasks, the skill, the docs, the workflow
                                   (--locales en,de --catalogs locales --gsxui ui)
  go tool intl check           04  every problem, for CI (--missing warn; or only the locales named)
  go tool intl show            05  what's done, and what to run next (go tool intl, too; --json)
  go tool intl gen             06  the catalogs compiled into the app's messages package
  go tool intl status          07  per language: translated, inherited, stale, missing
  go tool intl add-locale      08  a language added: de, pt-BR
  go tool intl todo            09  what's left to translate (--json, for agents)
  go tool intl translate       10  Claude Code translates what's left, a worker per language
  go tool intl export          11  XLIFF for translators (--all, --out dir)
  go tool intl import          12  translators' XLIFF read back
  go tool intl confirm         13  stale translations that are still right, kept (<locale>[:<key>]…)
  go tool intl prune           14  keys the source no longer has, removed

And:
  go tool intl verify          what the repo's GitHub workflow runs: its setup and test tasks, then check
  go tool intl template        the template's little app, in a repo with no app yet: it never writes over a file
  go tool intl add             intl's screens, in a gsx app (--diff, --overwrite): language-list, language-offer, preferences
  go tool intl assets          the JavaScript a Worker and a page import (--worker dir, --browser dir)
  go tool intl locales         the shipped locales, comma-separated
  go tool intl catalogs        the catalogs' Go package name
  go tool intl markup          the app's own markup files, for gsx's checks
  go tool intl show-version    this build's version
  go tool intl show-help       this
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
		fail(fmt.Errorf("%w: run go tool intl init where the app's go.mod is", err))
	}
	return c
}
