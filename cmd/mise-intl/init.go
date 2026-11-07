package main

import (
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/mise-check/miseconfig"
	"github.com/joeblew999/mise-check/toolfiles"
	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl/config"
	"github.com/joeblew999/mise-intl/templates"
)

// self is mise-intl as a repo's mise pins it, from its GitHub releases (.mise/config.toml, which init writes).
const self = "github:joeblew999/mise-intl"

// tool is the template's tool half as init writes it into a repo and check names what a repo lacks of it (mise-check's
// toolfiles): every file at its path, and of the three a repo may have of its own (.mise/config.toml, .gitignore,
// .claude/settings.json) only mise-intl's parts: its pin, this release (local for a checkout's build) or a newer one,
// its own name in minimum_release_age_excludes with use_versions_host = false, the template's ignores, its hook.
func tool() toolfiles.Tool {
	tmpl, err := fs.ReadFile(templates.Tool, ".mise/config.toml")
	must(err)
	return toolfiles.Tool{Files: templates.Tool, Hook: templates.HookFile,
		Config: miseconfig.Tool{Name: self, Release: strings.TrimPrefix(version, "v"), Template: tmpl, Init: "mise run intl:init"}}
}

// initRepo is mise-intl init: the repo set up for intl. First the template's tool files (tool: the tasks, and
// .mise/config.toml pinning this build, the release that ran it (mise exec github:joeblew999/mise-intl@latest --
// mise-intl init, the README's way) or local; the GitHub workflow, the Claude Code hook and skill, the docs), each at its
// path; in a Go module, intl's packages required in its go.mod at this release (pinLibrary); then the repo's own
// (setup.Init): intl.toml with its languages (--locales en,de), the catalogs (--catalogs locales), and in a gsx app
// gsx.toml's filters and renderers; then gen; then mise-check's own init, so the repo has mise-check's tasks itself
// (toolfiles.MiseCheckInit). Every step leaves alone what is the repo's, so it runs again after upgrading: the files that
// are intl's are written as this version writes them.
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
	must(tool().Write(root))
	for _, s := range oldNames(root) {
		fmt.Println("·", s)
	}
	toolfiles.Trust(root, ".mise/config.toml", nil)
	pinLibrary(root)
	must(setup.Init(root, o, os.Stdout))
	c, err := config.Find(root)
	must(err)
	must(read(c).Generate(os.Stderr))
	must(toolfiles.MiseCheckInit(root))
	sayNewest()
	fmt.Println("✓ next: mise run intl: what's done, and what to run next")
}

// sayNewest says, of a release, when GitHub has a newer one, and how to set the repo up at it: mise takes a release as
// latest only once it's a day old, so @latest can run an older one than the newest.
func sayNewest() {
	if version == "" {
		return
	}
	switch n := newestRelease(); {
	case n == "":
		fmt.Printf("· this is mise-intl %s; GitHub didn't say what its newest release is\n", buildVersion())
	case olderVersion(buildVersion(), n):
		fmt.Printf("· this is mise-intl %s, and the newest release is %s: mise takes a release as latest only once it's a day old. For it, which moves the repo's pin to it:\n  mise exec %s@%s -- mise-intl init\n", buildVersion(), n, self, n)
	default:
		fmt.Printf("✓ mise-intl %s, the newest release\n", buildVersion())
	}
}

// newestRelease is mise-intl's newest release, as GitHub's page for it names it (…/releases/latest redirects to its
// tag), or "" if GitHub can't say.
func newestRelease() string {
	c := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Get("https://github.com/joeblew999/mise-intl/releases/latest")
	if err != nil {
		return ""
	}
	resp.Body.Close()
	_, tag, _ := strings.Cut(resp.Header.Get("Location"), "/releases/tag/")
	return strings.TrimPrefix(tag, "v")
}

// stale is what init would write again in the repo at root (tool's Problems: its files missing or out of date, its
// parts of the repo's own files), and the repo's own files that name a task by its old number (oldNames).
func stale(root string) []string { return append(oldNames(root), tool().Problems(root)...) }

// oldName is an intl task by the number it had before the tasks were named for their commands: intl:06-gen.
var oldName = regexp.MustCompile(`\bintl:[0-9]{2}-([a-z-]+)`)

// oldNames are the repo's own mise files and workflows that name a task by its old number (intl:06-gen is intl:gen
// now), each with its new name. The files init writes are this version's already.
func oldNames(root string) []string {
	var out []string
	for _, glob := range []string{"mise.toml", "mise.*.toml", ".mise/*.toml", ".mise/tasks/*.toml", ".github/workflows/*.yml"} {
		files, _ := filepath.Glob(filepath.Join(root, filepath.FromSlash(glob)))
		for _, f := range files {
			rel, _ := filepath.Rel(root, f)
			if p := filepath.ToSlash(rel); p == ".mise/tasks/mise-intl.toml" || p == ".github/workflows/intl.yml" {
				continue
			}
			b, _ := os.ReadFile(f)
			for _, m := range oldName.FindAllSubmatch(b, -1) {
				out = append(out, fmt.Sprintf("%s names %s, which is intl:%s now: rename it there", filepath.ToSlash(rel), m[0], m[1]))
			}
		}
	}
	return out
}

// intlModule is intl's module: an app's go.mod requires it for the packages it imports (intl, intl/workerd, gsxintl).
const intlModule = "github.com/joeblew999/mise-intl"

// pinLibrary requires this release of intl's packages in the repo's go.mod (go get …@vX.Y.Z, with the Go mise gives
// the repo), so the packages the app imports are the release its mise-intl is: run by hand at a newer release, init is
// the move to it. A repo with no go.mod yet gets it from mise-intl template; a go.mod that requires this release or a
// newer one, or replaces the module with a folder, stays as it is, and so does every go.mod when this is a checkout's
// build.
func pinLibrary(root string) {
	gomod := setup.GoMod(root)
	at := buildVersion()
	if gomod == "" || version == "" {
		return
	}
	b, err := os.ReadFile(gomod)
	must(err)
	text := string(b)
	if pinned := requiredVersion(text, intlModule); strings.Contains(text, intlModule+" =>") || pinned != "" && !olderVersion(strings.TrimPrefix(pinned, "v"), at) {
		return
	}
	cmd := goCommand(filepath.Dir(gomod), "get", intlModule+"@v"+at)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	must(cmd.Run())
	fmt.Printf("✓ go.mod requires intl's packages at %s, mise-intl's release\n", at)
}

// goCommand is go with args in dir: through mise when it's there (mise exec, so it's the Go the repo's mise gives it,
// go.mod's), else the go on the PATH.
func goCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	if _, err := exec.LookPath("mise"); err == nil {
		cmd = exec.Command("mise", append([]string{"exec", "--", "go"}, args...)...)
	}
	cmd.Dir = dir
	return cmd
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
