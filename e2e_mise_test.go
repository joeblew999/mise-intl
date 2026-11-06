// The repo, and the calls into its own mise file, made in the repo as a person there makes them. Never the root's
// mise.toml.

package e2e

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/joeblew999/mise-check/walk"
)

// repo is a Go app's repo that uses mise-intl, walked through its own mise file: walk.Repo (this repo, Top; the repo's
// name, and its folder).
type repo struct {
	walk.Repo
}

// The stamped copy's app is its own, so its walk never touches the template's: its folder's name, as template names
// the app's module.
const stampedApp = "intl-new"

// stampedCopy is a new repo, made as a person makes one: an empty folder with its own git, then mise-intl init, run at
// version, a release from GitHub, or this checkout's build (local) when it is "" (init pins mise-intl at itself, in
// .mise/config.toml), then the repo's own intl:template, which writes the template's app, its module named for the
// folder. With this checkout's build, the go.mod is there first, using this checkout's packages (go mod init, then a
// replace), as a person trying a change has it. It is in a folder of its own that goes when the test ends.
func (r repo) stampedCopy(version string) repo {
	r.T.Helper()
	n := repo{Repo: r.New(stampedApp)}
	at := cmp.Or(strings.TrimPrefix(version, "v"), "local")
	if at == "local" {
		n.Must("go", "mod", "init", stampedApp)
		n.Must("go", "mod", "edit", "-require=github.com/joeblew999/mise-intl@v0.0.0", "-replace=github.com/joeblew999/mise-intl="+r.Top)
	}
	n.miseInit(at)
	if b, _ := os.ReadFile(filepath.Join(n.Dir, ".mise", "tasks", "mise-intl.toml")); !bytes.Contains(b, []byte(`["intl:template"]`)) {
		r.T.Fatalf("mise-intl %s has no intl:template: a release from before it can't make a new repo", at)
	}
	n.Name = "a new repo"
	n.MustTask("intl:template")
	if b, _ := os.ReadFile(filepath.Join(n.Dir, "go.mod")); !bytes.Contains(b, []byte("module "+stampedApp)) {
		r.T.Fatalf("intl:template didn't name the app's module for its folder, %s: the walk would walk %s's\n%s", stampedApp, r.Name, b)
	}
	n.Pass("%s, %s: an empty folder, its own git, mise-intl init at %s, then intl:template (the template's app, its module named for the folder)", n.Name, stampedApp, at)
	return n
}

// versions is what the repo runs, each as its own tool reports it: its mise tools (mise ls --local --json), the build
// of mise-intl (intl:show-version) and the workerd mise run setup installed (its package.json).
func (r repo) versions() string {
	r.T.Helper()
	var tools map[string][]struct {
		Version string
		Active  bool
	}
	if err := json.Unmarshal(r.Stdout("mise", "ls", "--local", "--json"), &tools); err != nil || len(tools) == 0 {
		r.T.Fatalf("the repo's mise tools, mise ls --local --json: %v (%d read)", err, len(tools))
	}
	var v []string
	for _, name := range slices.Sorted(maps.Keys(tools)) {
		for _, t := range tools[name] {
			if t.Active {
				v = append(v, name+" "+t.Version)
			}
		}
	}
	build := strings.TrimSpace(string(r.Stdout("mise", "run", "intl:show-version")))
	var w struct{ Version string }
	b, err := os.ReadFile(filepath.Join(r.Dir, "node_modules", "workerd", "package.json"))
	if err == nil {
		err = json.Unmarshal(b, &w)
	}
	if err != nil || w.Version == "" || build == "" {
		r.T.Fatalf("the installed workerd (%q, %v) or mise-intl's build (%q)", w.Version, err, build)
	}
	return fmt.Sprintf("%s (mise); mise-intl's build %s; workerd %s, installed", strings.Join(v, ", "), build, w.Version)
}

// miseInit runs mise-intl init at the release at (local: this checkout's build) in the repo, on its own mise only, as a
// person runs it, and checks it pinned itself.
func (r repo) miseInit(at string) {
	r.T.Helper()
	r.Must("mise", "exec", "github:joeblew999/mise-intl@"+at, "--", "mise-intl", "init")
	want := `"github:joeblew999/mise-intl" = "` + at + `"`
	if b, _ := os.ReadFile(filepath.Join(r.Dir, ".mise", "config.toml")); !bytes.Contains(b, []byte(want)) {
		r.T.Fatalf("mise-intl %s's init didn't pin itself in .mise/config.toml (%s): a release from before it did can't make a new repo", at, want)
	}
}

// local is what isn't a repo's own, wherever it is, which the walk never counts as a step's change: installed
// packages.
var local = []string{"node_modules"}

// miseServe runs the app as a person does, with the repo's own serve task (mise run serve), on a free port, and returns
// its page at path.
func (r repo) miseServe(path string) string {
	r.T.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		r.T.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	cmd := exec.Command("mise", "run", "serve")
	cmd.Dir, cmd.Env = r.Dir, r.Env("ADDR="+addr)
	walk.OwnGroup(cmd)
	var logs strings.Builder
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		r.T.Fatal(err)
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
	r.T.Fatalf("the app (mise run serve) never answered on %s:\n%s", addr, logs.String())
	return ""
}
