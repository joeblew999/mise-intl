// The repo, and what's intl's in making one, made as a person makes one: a new repo (mise-intl init, then intl:template),
// and the template where it is. Never the root's mise.toml.

package e2e

import (
	"bytes"
	"cmp"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joeblew999/mise-check/walk"
)

// newRepo is a new repo, made as a person makes one, as README.md's "Use it in your repo" says and as mise-cloudflare's
// walk makes its: an empty folder with its own git, mise-intl init at the release -version names (mise exec
// github:joeblew999/mise-intl@<version> -- mise-intl init, which pins itself in .mise/config.toml), then its own
// intl:template (the template's app, its go.mod named for the folder and requiring intl's packages at that release),
// then mise run intl. Without a version it's this checkout's build (local, mise run tool:install), and the go.mod is
// there first, using this checkout's packages (go mod init, then a replace), as a person trying a change has it.
func newRepo(r walk.Repo, version string) walk.Repo {
	n := r.New("intl-new")
	at := cmp.Or(strings.TrimPrefix(version, "v"), "local")
	if at == "local" {
		n.Must("go", "mod", "init", "intl-new")
		n.Must("go", "mod", "edit", "-require=github.com/joeblew999/mise-intl@v0.0.0", "-replace=github.com/joeblew999/mise-intl="+r.Top)
	}
	n.Must("mise", "exec", "github:joeblew999/mise-intl@"+at, "--", "mise-intl", "init")
	want := `"github:joeblew999/mise-intl" = "` + at + `"`
	if b, _ := os.ReadFile(filepath.Join(n.Dir, ".mise", "config.toml")); !bytes.Contains(b, []byte(want)) {
		r.T.Fatalf("mise-intl %s's init didn't pin itself in .mise/config.toml (%s): a release from before it did can't make a new repo", at, want)
	}
	n.MustTask("intl:template")
	if b, _ := os.ReadFile(filepath.Join(n.Dir, "go.mod")); !bytes.Contains(b, []byte("module intl-new")) {
		r.T.Fatalf("intl:template didn't name the app's module for its folder, intl-new:\n%s", b)
	}
	if out := n.MustTask("intl"); !strings.Contains(out, "next: ") {
		r.T.Fatalf("mise run intl says nothing to run next:\n%s", out)
	}
	n.Pass("%s, intl-new: an empty folder, its own git, mise-intl init at %s, then intl:template (the template's app, its module named for the folder), then mise run intl", n.Name, at)
	return n
}

// withGo gives the template, where it is, the Go of the module it's a folder of: it has no go.mod of its own, and the
// walk cuts mise off from the repo around it, so its own mise files have no Go to give its tasks (mise-intl …);
// the Go this test runs with is that module's. A new repo has a go.mod of its own, whose Go mise installs.
func withGo(r walk.Repo) walk.Repo {
	if _, err := os.Stat(filepath.Join(r.Dir, "go.mod")); err == nil {
		return r
	}
	r.With = append(r.With, "MISE_GO_VERSION="+strings.TrimPrefix(runtime.Version(), "go"))
	return r
}
