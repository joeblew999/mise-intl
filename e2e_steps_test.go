// The steps: the repo's numbered tasks, in their order, each with its check (the walk tests walk them).

package e2e

import (
	"cmp"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joeblew999/mise-check/walk"
)

// steps are the repo's numbered tasks (its .mise/tasks/mise-intl.toml, which is the template's,
// templates/.mise/tasks/mise-intl.toml), in their order, each with its check: how the walk runs it and checks it did
// what it says, as a person would see it. TestSteps fails unless they're exactly the template's numbered tasks, in
// their order.
var steps = []walk.Step[*walker]{
	{Task: "intl:03-init", Check: func(w *walker) {
		before := w.Snapshot(local...)
		w.MustTask("intl:03-init")
		if changed := walk.Diff(before, w.Snapshot(local...)); len(changed) > 0 {
			w.T.Fatalf("intl:03-init changed %s: the repo isn't what init writes", strings.Join(changed, ", "))
		}
		w.Pass("intl:03-init changed nothing: the repo is what init writes")
	}},
	{Task: "intl:04-check", Check: func(w *walker) {
		if out := w.MustTask("intl:04-check"); !regexp.MustCompile(`✓ \d+ messages in \d+ locales`).MatchString(out) {
			w.T.Fatalf("intl:04-check: %s", out)
		}
		w.Pass("intl:04-check: no problem")
		// As on GitHub, from a fresh checkout: nothing installed, so verify's own setup has to install workerd.
		if err := os.RemoveAll(filepath.Join(w.Dir, "node_modules")); err != nil {
			w.T.Fatal(err)
		}
		w.MustTask("intl:verify")
		w.Pass("intl:verify: what the repo's GitHub workflow runs passes, from nothing installed, as a fresh checkout is (its setup and test tasks, then check)")
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
		// The version is the release .mise/config.toml pins (-version), or this checkout's build (local), Go's own
		// version of it.
		if want := strings.TrimPrefix(*walk.Version, "v"); want != "" && s.Version != want || s.Version == "" {
			w.T.Fatalf("intl:05-show's version is %q, not %q: the one .mise/config.toml pins", s.Version, cmp.Or(want, "this checkout's build"))
		}
		w.Pass("intl:05-show's version is the one .mise/config.toml pins: %s", s.Version)
	}},
	{Task: "intl:06-gen", Check: func(w *walker) {
		w.MustTask("intl:06-gen")
		w.Pass("intl:06-gen wrote the messages package")
		for path, want := range map[string][]string{
			"/":    {`lang="en"`, "July 4, 2026", "€1,234.50", "Go, mise, and intl", "Japan", "This page is in 2 languages."},
			"/de/": {`lang="de"`, "4. Juli 2026", "1.234,50\u00a0€", "Go, mise und intl", "vor 3 Tagen", "Diese Seite gibt es in 2 Sprachen."},
		} {
			body := w.miseServe(path)
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
