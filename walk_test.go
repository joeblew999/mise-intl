package e2e

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/mise-check/walk"
)

// intlWalk is the walk: the template's numbered tasks (templates/.mise/tasks/intl.toml), in their order, through the
// repo's own mise file, each checked by what a person would see it do; on the template where it is (mise run
// template:e2e), and on a new repo made from nothing as docs/intl/README.md says (mise run new-repo:e2e). mise-check's
// walk does the rest.
var intlWalk = walk.Walk[*walker]{Steps: steps, New: newRepo, Prepare: withGo, Of: func(r walk.Repo) *walker {
	return &walker{Repo: r, inPlace: os.Getenv("E2E_STAMPED") != "true"}
}}

func TestWalk(t *testing.T)  { intlWalk.Run(t) }
func TestSteps(t *testing.T) { walk.CheckSteps(t, "templates/.mise/tasks/intl.toml", steps) }

// walker is intl's walk of a repo: what each step's Check is given. The steps that change the catalogs run on a new
// repo only, never on the template where it is (inPlace).
type walker struct {
	walk.Repo
	inPlace bool
}

// newOnly is why a step that changes the catalogs doesn't run on the template where it is.
const newOnly = "it changes the catalogs, so the walk runs it on a new repo (mise run new-repo:e2e), not on the template"

// steps are the template's numbered tasks, each checked. TestSteps fails unless they're exactly those, in their order.
var steps = []walk.Step[*walker]{
	{Task: "intl:03-init", Check: func(w *walker) {
		before := w.Snapshot()
		w.MustTask("intl:03-init")
		if changed := walk.Diff(before, w.Snapshot()); len(changed) > 0 {
			w.T.Fatalf("intl:03-init changed %s: the repo isn't what init writes", strings.Join(changed, ", "))
		}
		w.Pass("intl:03-init changed nothing: the repo is what init writes")
	}},
	{Task: "intl:04-check", Check: func(w *walker) {
		if out := w.MustTask("intl:04-check"); !regexp.MustCompile(`✓ \d+ messages in \d+ locales`).MatchString(out) {
			w.T.Fatalf("intl:04-check: %s", out)
		}
		w.Pass("intl:04-check: no problem")
		w.MustTask("intl:verify")
		w.Pass("intl:verify: what the repo's GitHub workflow runs passes (the app's test task, then check)")
	}},
	{Task: "intl:05-show", Check: func(w *walker) {
		var s struct {
			Version        string
			Generated, App bool
			Left           int
			Next, Command  string
		}
		w.TaskJSON(&s, "intl:05-show", "--json")
		if !s.Generated || !s.App || s.Left != 0 || s.Next != "mise run intl:08-add-locale" {
			w.T.Fatalf("intl:05-show --json: %+v", s)
		}
		w.Pass("intl:05-show says next: %s (without mise: %s)", s.Next, s.Command)
		// The version is the release the repo was made at (-version), or a checkout's build.
		want := "(devel)"
		if *walk.Version != "" {
			want = strings.TrimPrefix(*walk.Version, "v")
		}
		if s.Version != want {
			w.T.Fatalf("intl:05-show's version is %q, not %q: the one go.mod pins", s.Version, want)
		}
		w.Pass("intl:05-show's version is the one go.mod pins: %s", s.Version)
	}},
	{Task: "intl:06-gen", Check: func(w *walker) {
		w.MustTask("intl:06-gen")
		w.Pass("intl:06-gen wrote the messages package")
		for path, want := range map[string][]string{
			"/":    {`lang="en"`, "July 4, 2026", "€1,234.50", "Go, mise, and intl", "Japan", "This page is in 2 languages."},
			"/de/": {`lang="de"`, "4. Juli 2026", "1.234,50\u00a0€", "Go, mise und intl", "vor 3 Tagen", "Diese Seite gibt es in 2 Sprachen."},
		} {
			body := w.serve(path)
			for _, s := range want {
				if !strings.Contains(body, s) {
					w.T.Fatalf("the app at %s has no %q:\n%s", path, s, body)
				}
			}
			w.Pass("the app, run as a person runs it (mise run serve), at %s: %s", path, strings.Join(want[1:3], ", "))
		}
	}},
	{Task: "intl:07-status", Check: func(w *walker) {
		if out := w.MustTask("intl:07-status"); !strings.Contains(out, "de") || !strings.Contains(out, "translated") {
			w.T.Fatalf("intl:07-status: %s", out)
		}
		w.Pass("intl:07-status: every language translated")
	}},
	{Task: "intl:08-add-locale", Check: func(w *walker) {
		if w.inPlace {
			w.Pass("intl:08-add-locale: " + newOnly)
			return
		}
		w.MustTask("intl:08-add-locale", "nl")
		if b, _ := os.ReadFile(filepath.Join(w.Dir, "intl.toml")); !strings.Contains(string(b), `"nl"`) {
			w.T.Fatalf("intl.toml has no nl:\n%s", b)
		}
		if _, err := w.Task(nil, "", "intl:07-status"); err == nil {
			w.T.Fatal("intl:07-status passes with nl untranslated")
		}
		w.Pass("intl:08-add-locale nl: Dutch in intl.toml with its catalog, and intl:07-status fails until it's translated")
		// A contributor's pull request may leave its translations to the maintainer; a push may not.
		if out, err := w.Task([]string{"INTL_MISSING=", "GITHUB_EVENT_NAME=pull_request"}, "", "intl:04-check"); err != nil || !strings.Contains(out, "only warn") {
			w.T.Fatalf("intl:04-check on a pull request, nl untranslated: %v\n%s", err, out)
		}
		if _, err := w.Task([]string{"INTL_MISSING=", "GITHUB_EVENT_NAME=push"}, "", "intl:04-check"); err == nil {
			w.T.Fatal("intl:04-check on a push passes with nl untranslated")
		}
		w.Pass("intl:04-check with nl untranslated: a pull request only warns, a push fails")
	}},
	{Task: "intl:09-todo", Check: func(w *walker) {
		if w.inPlace {
			if out := w.MustTask("intl:09-todo"); !strings.Contains(out, "nothing to translate") {
				w.T.Fatalf("intl:09-todo on the template: %s", out)
			}
			w.Pass("intl:09-todo: nothing to translate")
			return
		}
		var todo struct {
			Locales []struct {
				ID   string
				Todo []struct{ Key string }
			}
		}
		w.TaskJSON(&todo, "intl:09-todo", "--json", "nl")
		if len(todo.Locales) != 1 || todo.Locales[0].ID != "nl" || len(todo.Locales[0].Todo) == 0 {
			w.T.Fatalf("intl:09-todo --json nl: %+v", todo)
		}
		w.Pass("intl:09-todo --json nl: %d messages to translate", len(todo.Locales[0].Todo))
	}},
	{Task: "intl:10-translate", Skip: "Claude Code translates: it needs a person's own Claude Code, so the walk translates as a translator does instead (intl:11-export, intl:12-import)"},
	{Task: "intl:11-export", Check: func(w *walker) {
		if w.inPlace {
			w.Pass("intl:11-export: " + newOnly)
			return
		}
		w.MustTask("intl:11-export", "nl")
		b, err := os.ReadFile(filepath.Join(w.Dir, "xliff", "out", "nl.xlf"))
		if err != nil || !strings.Contains(string(b), "<source>") {
			w.T.Fatalf("xliff/out/nl.xlf: %v\n%s", err, b)
		}
		w.Pass("intl:11-export nl: xliff/out/nl.xlf, %d messages for a translator", strings.Count(string(b), "<source>"))
	}},
	{Task: "intl:12-import", Check: func(w *walker) {
		if w.inPlace {
			w.Pass("intl:12-import: " + newOnly)
			return
		}
		// The translator's work: every message's target, here its source (Dutch has English's plural forms), each
		// segment marked translated, as a translator's tool marks it (XLIFF 2.0's state).
		b, err := os.ReadFile(filepath.Join(w.Dir, "xliff", "out", "nl.xlf"))
		if err != nil {
			w.T.Fatal(err)
		}
		done := regexp.MustCompile(`(?s)<source>(.*?)</source>`).ReplaceAll(b, []byte("<source>$1</source><target>$1</target>"))
		done = regexp.MustCompile(`<segment( state="[a-z]+")?>`).ReplaceAll(done, []byte(`<segment state="translated">`))
		if err := os.MkdirAll(filepath.Join(w.Dir, "xliff", "in"), 0o755); err != nil {
			w.T.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(w.Dir, "xliff", "in", "nl.xlf"), done, 0o644); err != nil {
			w.T.Fatal(err)
		}
		w.MustTask("intl:12-import")
		w.MustTask("intl:07-status")
		w.Pass("intl:12-import: the translator's xliff/in/nl.xlf read back, and every language is translated again")
	}},
	{Task: "intl:13-confirm", Check: func(w *walker) {
		if w.inPlace {
			w.Pass("intl:13-confirm: " + newOnly)
			return
		}
		// A developer changes a source message: its translations go stale until they're redone, or confirmed.
		en := filepath.Join(w.Dir, "locales", "en.toml")
		b, err := os.ReadFile(en)
		if err != nil || !strings.Contains(string(b), `read = "Read it in"`) {
			w.T.Fatalf("locales/en.toml: %v", err)
		}
		if err := os.WriteFile(en, []byte(strings.Replace(string(b), `read = "Read it in"`, `read = "Read it in:"`, 1)), 0o644); err != nil {
			w.T.Fatal(err)
		}
		w.MustTask("intl:06-gen")
		if _, err := w.Task(nil, "", "intl:07-status"); err == nil {
			w.T.Fatal("intl:07-status passes with home.read changed in English and not in the translations")
		}
		w.MustTask("intl:13-confirm", "de:home.read", "nl:home.read")
		w.MustTask("intl:07-status")
		w.Pass("intl:13-confirm: home.read changed in English, its translations stale, then confirmed as still right")
	}},
	{Task: "intl:14-prune", Check: func(w *walker) {
		if w.inPlace {
			w.Pass("intl:14-prune: " + newOnly)
			return
		}
		de := filepath.Join(w.Dir, "locales", "de.toml")
		b, err := os.ReadFile(de)
		if err != nil {
			w.T.Fatal(err)
		}
		if err := os.WriteFile(de, append(b, "\n[gone]\nold = \"Nicht mehr\"\n"...), 0o644); err != nil {
			w.T.Fatal(err)
		}
		w.MustTask("intl:14-prune")
		if b, _ := os.ReadFile(de); strings.Contains(string(b), "Nicht mehr") {
			w.T.Fatalf("intl:14-prune left gone.old in de.toml:\n%s", b)
		}
		w.MustTask("intl:04-check")
		w.Pass("intl:14-prune: gone.old, which English no longer has, removed from de.toml; intl:04-check passes")
	}},
}

// serve runs the app as a person does (mise run serve, its own task) on a free port, and returns its page at path.
func (w *walker) serve(path string) string {
	w.T.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		w.T.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	cmd := exec.Command("mise", "run", "serve")
	cmd.Dir, cmd.Env = w.Dir, w.Env("ADDR="+addr)
	walk.OwnGroup(cmd)
	var logs strings.Builder
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		w.T.Fatal(err)
	}
	defer walk.KillGroup(cmd)
	for deadline := time.Now().Add(2 * time.Minute); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
		resp, err := http.Get("http://" + addr + path)
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return string(b)
	}
	w.T.Fatalf("the app (mise run serve) never answered on %s:\n%s", addr, logs.String())
	return ""
}

// intlTool is intl's tool line: the one line of a repo's go.mod that pins its command and its packages.
const intlTool = "github.com/joeblew999/mise-intl/cmd/intl"

// newRepo is a new repo, made as docs/intl/README.md says a person makes one: an empty folder with its own git, go mod
// init, intl's tool line, then go tool intl init and intl:template. Without a version it's this checkout: go.mod
// requires mise-intl and replaces it with this checkout, as a person trying a change does, and the tool line goes in
// with go mod edit and go mod tidy; with one, it's that release (go get -tool …@version).
func newRepo(r walk.Repo, version string) walk.Repo {
	n := r.New("intl-new")
	n.Must("go", "mod", "init", "example.com/intlnew")
	at := "this checkout"
	if version == "" {
		n.Must("go", "mod", "edit", "-require=github.com/joeblew999/mise-intl@v0.0.0", "-replace=github.com/joeblew999/mise-intl="+r.Top, "-tool="+intlTool)
		n.Must("go", "mod", "tidy") // go.sum: what the checkout's module needs, as a release's go get -tool writes it
	} else {
		n.Must("go", "get", "-tool", intlTool+"@"+version)
		at = version
	}
	n.Must("go", "tool", "intl", "init")
	n.MustTask("intl:template")
	n.Pass("%s: an empty folder, its own git, go mod init, intl's tool line at %s, go tool intl init, then intl:template", n.Name, at)
	return n
}

// withGo gives the template, where it is, the Go of the module it's a folder of: it has no go.mod of its own, and the
// walk cuts mise off from the repo around it, so its own mise files have no Go to give its tasks (go tool intl …);
// the Go this test runs with is that module's. A new repo has a go.mod of its own, whose Go mise installs.
func withGo(r walk.Repo) walk.Repo {
	if _, err := os.Stat(filepath.Join(r.Dir, "go.mod")); err == nil {
		return r
	}
	r.With = append(r.With, "MISE_GO_VERSION="+strings.TrimPrefix(runtime.Version(), "go"))
	return r
}
