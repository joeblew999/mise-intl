package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl/config"
	"github.com/joeblew999/mise-intl/templates"
)

// theirs are the template's files a repo may have of its own already: init writes them only when it has none.
// .mise/config.toml is what intl needs from mise (Go, from go.mod), as mise-cloudflare's init writes its pin there.
var theirs = map[string]bool{".gitignore": true, ".claude/settings.json": true, ".mise/config.toml": true}

// initRepo is go tool intl init (03): the repo set up for intl, in its Go module. Run as a release (go run
// github.com/joeblew999/mise-intl/cmd/intl@latest init, the README's way), it pins itself first: intl's tool line in
// go.mod at exactly that release (pinSelf), so go tool intl is that release from then on. Then the template's tool files
// (templates.Tool: the numbered tasks and their mise config, the GitHub workflow, the Claude Code hook and skill, the
// docs), each at its path; then the repo's own (setup.Init): intl.toml with its languages (--locales en,de), the
// catalogs (--catalogs locales), and in a gsx app gsx.toml's filters and renderers; then gen. Every step leaves alone
// what is the repo's, so it runs again after upgrading: the files that are intl's are written as this version writes
// them.
func initRepo(args []string) {
	f := flag.NewFlagSet("init", flag.ExitOnError)
	locales := f.String("locales", "en", "BCP 47 ids, comma-separated, the source first")
	catalogs := f.String("catalogs", "locales", "the catalogs' folder")
	gsxui := f.String("gsxui", "", "in a gsx app: where gsxui add puts its components (default: gsxui.json's ui)")
	f.Parse(args)
	root, err := os.Getwd()
	must(err)
	o := setup.Options{Locales: strings.Split(*locales, ","), Catalogs: *catalogs, Gsxui: *gsxui}
	if c, err := config.Find(root); err == nil && c.Root == root {
		o = setup.Options{Locales: c.Locales, Catalogs: c.Catalogs, Gsxui: c.Gsxui} // already set up: keep its choices
	}
	if o.Gsxui == "" {
		o.Gsxui = setup.DetectGsxui(root) // gsxui set up since: intl.toml gets it
	}
	pinSelf(root)
	writeTool(root)
	trustConfig(root, ".mise/config.toml")
	must(setup.Init(root, o, os.Stdout))
	c, err := config.Find(root)
	must(err)
	must(read(c).Generate(os.Stderr))
	fmt.Println("✓ next: go tool intl (or mise run intl): what's done, and what to run next")
}

// writeTool writes the template's tool files into the repo at root, each at its path there, but a file that is the
// repo's own.
func writeTool(root string) {
	must(fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		to := filepath.Join(root, filepath.FromSlash(p))
		if theirs[p] && exists(to) {
			return nil
		}
		b, err := fs.ReadFile(templates.Tool, p)
		if err != nil {
			return err
		}
		if _, err := writeChanged(root, to, b); err != nil {
			return err
		}
		if strings.HasSuffix(p, ".sh") {
			return os.Chmod(to, 0o755)
		}
		return nil
	}))
}

// stale names the files init wrote that this version would write differently, and a .claude/settings.json of the
// repo's own that doesn't run the hook.
func stale(root string) []string {
	var out []string
	fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || theirs[p] {
			return err
		}
		want, _ := fs.ReadFile(templates.Tool, p)
		// Line endings aside: a Windows checkout of the repo may have \r\n where init wrote \n.
		crlf := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		switch {
		case err != nil:
			out = append(out, p+" is missing: run go tool intl init")
		case !bytes.Equal(crlf(b), crlf(want)):
			out = append(out, p+" is out of date: run go tool intl init")
		}
		return nil
	})
	if b, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json")); err == nil && !bytes.Contains(b, []byte(templates.HookFile)) {
		out = append(out, ".claude/settings.json doesn't run .claude/hooks/"+templates.HookFile+": add its SessionStart entry (docs/intl/README.md)")
	}
	return out
}

// trustConfig has mise trust a config file intl wrote, so its tasks run straight away, when mise is there: a repo
// without mise needs none of it.
func trustConfig(root, file string) {
	path := filepath.Join(root, filepath.FromSlash(file))
	if !exists(path) {
		return
	}
	if _, err := exec.LookPath("mise"); err != nil {
		return
	}
	if out, err := exec.Command("mise", "trust", "-q", path).CombinedOutput(); err != nil {
		fmt.Printf("mise didn't trust %s (%v %s): run mise trust %s\n", file, err, strings.TrimSpace(string(out)), file)
	}
}

// intlModule and intlTool are intl's module and its tool line: the one line of a repo's go.mod that pins its command
// and its packages.
const (
	intlModule = "github.com/joeblew999/mise-intl"
	intlTool   = intlModule + "/cmd/intl"
)

// pinSelf pins this release of intl in the repo's go.mod (go get -tool …@vX.Y.Z), as mise-cloudflare's init pins
// itself in .mise/config.toml: run by hand at a newer release than the repo pins, init is the move to it. A go.mod that
// pins this release or a newer one, or replaces the module with a folder, stays as it is, and so does every go.mod
// when this is a checkout's build.
func pinSelf(root string) {
	gomod := setup.GoMod(root)
	if gomod == "" {
		fail(errors.New("intl init: run it in a Go module, where the app's go.mod is (go mod init <module> first)"))
	}
	b, err := os.ReadFile(gomod)
	must(err)
	text, at := string(b), buildVersion()
	pinned, replaced := requiredVersion(text, intlModule), strings.Contains(text, intlModule+" =>")
	switch {
	case at == "(devel)" || replaced:
		if !hasToolLine(text, intlTool) {
			fmt.Println("· this is a checkout's build of intl: pin a release yourself, go get -tool " + intlTool + "@latest")
		}
		return
	case pinned != "" && hasToolLine(text, intlTool) && !olderVersion(strings.TrimPrefix(pinned, "v"), at):
		fmt.Printf("✓ go.mod pins intl %s (go tool intl)\n", strings.TrimPrefix(pinned, "v"))
		return
	}
	cmd := exec.Command("go", "get", "-tool", intlTool+"@v"+at)
	cmd.Dir, cmd.Stdout, cmd.Stderr = filepath.Dir(gomod), os.Stdout, os.Stderr
	must(cmd.Run())
	fmt.Printf("✓ go.mod pins intl %s: go tool intl from now on\n", at)
}

// requiredVersion is the version a go.mod's text requires of module, "" if it doesn't.
func requiredVersion(text, module string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		f := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "require "))
		if len(f) >= 2 && f[0] == module && strings.HasPrefix(f[1], "v") {
			return f[1]
		}
	}
	return ""
}

// hasToolLine reports whether a go.mod's text has the tool line for tool, in a tool block or on its own.
func hasToolLine(text, tool string) bool {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if f := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "tool ")); len(f) == 1 && f[0] == tool {
			return true
		}
	}
	return false
}

// olderVersion reports whether release a is older than release b, both x.y.z.
func olderVersion(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, errx := strconv.Atoi(pa[i])
		y, erry := strconv.Atoi(pb[i])
		if errx != nil || erry != nil {
			return false
		}
		if x != y {
			return x < y
		}
	}
	return len(pa) < len(pb)
}
