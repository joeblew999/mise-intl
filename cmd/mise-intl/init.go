package main

import (
	"bytes"
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
	"github.com/joeblew999/mise-intl/internal/setup"
	"github.com/joeblew999/mise-intl/intl/config"
	"github.com/joeblew999/mise-intl/templates"
)

// theirs are the template's files a repo may have of its own already: init writes them only when it has none.
// .mise/config.toml is mise-intl's pin, as mise-cloudflare's is: written once, then the repo's, whose mise-intl parts
// init keeps (tool).
var theirs = map[string]bool{".gitignore": true, ".claude/settings.json": true, ".mise/config.toml": true}

// self is mise-intl as a repo's mise pins it, from its GitHub releases (.mise/config.toml, which init writes).
const self = "github:joeblew999/mise-intl"

// tool is mise-intl's parts of a repo's .mise/config.toml, as this build writes them (mise-check's miseconfig): its
// pin, this release (local for a checkout's build) or a newer one, and its own name in minimum_release_age_excludes
// with use_versions_host = false, so latest is its newest release the minute it's out. The rest is the repo's.
func tool() miseconfig.Tool {
	tmpl, err := fs.ReadFile(templates.Tool, ".mise/config.toml")
	must(err)
	return miseconfig.Tool{Name: self, Release: strings.TrimPrefix(version, "v"), Template: tmpl, Init: "mise run intl:init"}
}

// initRepo is mise-intl init: the repo set up for intl, as mise-cloudflare's init sets a repo up. First the
// template's tool files (templates.Tool: the tasks, and .mise/config.toml pinning this build, the release
// that ran it (mise exec github:joeblew999/mise-intl@latest -- mise-intl init, the README's way) or local; the GitHub
// workflow, the Claude Code hook and skill, the docs), each at its path; in a Go module, intl's packages required in
// its go.mod at this release (pinLibrary); then the repo's own (setup.Init): intl.toml with its languages (--locales
// en,de), the catalogs (--catalogs locales), and in a gsx app gsx.toml's filters and renderers; then gen. Every step
// leaves alone what is the repo's, so it runs again after upgrading: the files that are intl's are written as this
// version writes them.
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
	writeTool(root)
	for _, s := range oldNames(root) {
		fmt.Println("·", s)
	}
	trustConfig(root, ".mise/config.toml")
	pinLibrary(root)
	must(setup.Init(root, o, os.Stdout))
	c, err := config.Find(root)
	must(err)
	must(read(c).Generate(os.Stderr))
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

// writeTool writes the template's tool files into the repo at root, each at its path there, but a file that is the
// repo's own.
func writeTool(root string) {
	must(fs.WalkDir(templates.Tool, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		to := filepath.Join(root, filepath.FromSlash(p))
		if b, err := os.ReadFile(to); err == nil && theirs[p] {
			if p == ".mise/config.toml" {
				if out, wrote := tool().Keep(b); len(wrote) > 0 {
					must(os.WriteFile(to, out, 0o644))
					fmt.Printf("wrote %s: %s\n", p, strings.Join(wrote, "; "))
					return nil
				}
			}
			fmt.Printf("✓ %s is the repo's own: left as it is\n", p)
			return nil
		}
		b, err := fs.ReadFile(templates.Tool, p)
		if err != nil {
			return err
		}
		if p == ".mise/config.toml" {
			b, err = tool().Pinned()
			must(err)
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

// stale names the files init wrote that this version would write differently, mise-intl's parts of the repo's own
// .mise/config.toml it lacks, a .claude/settings.json of the repo's own that doesn't run the hook, and the repo's own
// files that name a task by its old number (oldNames).
func stale(root string) []string {
	out := oldNames(root)
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
			out = append(out, p+" is missing: run mise-intl init")
		case !bytes.Equal(crlf(b), crlf(want)):
			out = append(out, p+" is out of date: run mise-intl init")
		}
		return nil
	})
	if b, err := os.ReadFile(filepath.Join(root, ".mise", "config.toml")); err == nil {
		out = append(out, tool().Problems(b)...)
	}
	if b, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json")); err == nil && !bytes.Contains(b, []byte(templates.HookFile)) {
		out = append(out, ".claude/settings.json doesn't run .claude/hooks/"+templates.HookFile+": add its SessionStart entry (docs/intl/README.md)")
	}
	return out
}

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

// trustConfig has mise trust a config file mise-intl wrote into the repo, so its tasks run straight away: mise asks
// first about a config file it hasn't seen. Without mise on PATH, it says what to run.
func trustConfig(root, file string) {
	path := filepath.Join(root, filepath.FromSlash(file))
	if !exists(path) {
		return
	}
	if out, err := exec.Command("mise", "trust", "-q", path).CombinedOutput(); err != nil {
		fmt.Printf("mise didn't trust %s (%v %s): run mise trust %s\n", file, err, strings.TrimSpace(string(out)), file)
		return
	}
	fmt.Printf("✓ %s trusted by mise\n", file)
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
