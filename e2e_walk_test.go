// Package e2e is this repo's end-to-end test: a walk of a repo's numbered tasks, through the repo's own mise file, on
// mise-check's walk (github.com/joeblew999/mise-check/walk), which runs the steps in their order and does everything
// that isn't intl's. It sits at the top of this repo, beside the root mise.toml whose tasks run it, in a file for each
// thing it does:
//
//   - e2e_walk_test.go, this file: TestWalk, the test the root's tasks run (which repo, REPO_PATH), and TestSteps.
//   - e2e_steps_test.go: the steps, the repo's numbered tasks in their order, each with its check.
//   - e2e_mise_test.go: the repo, and what's intl's in making one: a new repo (mise-intl init, then intl:template), and
//     the calls the steps make.
//
// tool_test.go is the release (mise run tool:release), mise_test.go checks that the mise files and the Go line up,
// both ways, workerd_test.go the workerd the template pins, and gsx_test.go that intl's packages need no gsx.
package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joeblew999/mise-check/walk"
)

// intlWalk is the walk: the template's numbered tasks (templates/.mise/tasks/mise-intl.toml), in their order, each
// checked. The root's tasks say which repo: the template where it is (REPO_PATH=templates), or a new repo made the
// README's way (E2E_STAMPED=true, at the release -version names or this checkout's build).
var intlWalk = walk.Walk[*walker]{Steps: steps, New: newRepo, Prepare: prepare, Of: walkOf}

// TestWalk walks the repo the root's task names (mise run template:e2e is REPO_PATH=templates). Without REPO_PATH it
// skips, so go test ./... never walks.
func TestWalk(t *testing.T) { intlWalk.Run(t) }

// TestSteps checks that the steps are the template's numbered tasks, in their order, so the walk is that file's walk.
func TestSteps(t *testing.T) {
	walk.CheckSteps(t, "templates/.mise/tasks/mise-intl.toml", steps)
}

// newRepo is a new repo, made the README's way: an empty folder with its own git, mise-intl init at the release
// version names (this checkout's build without one), then intl:template.
func newRepo(r walk.Repo, version string) walk.Repo {
	return repo{Repo: r}.stampedCopy(version).Repo
}

// prepare gives the template, where it is, the Go of the module it's a folder of: it has no go.mod of its own, and the
// walk cuts mise off from the repo around it, so its own mise files have no Go to give its tasks; the Go this test runs
// with is that module's. A new repo has a go.mod of its own, whose Go mise installs.
func prepare(r walk.Repo) walk.Repo {
	if _, err := os.Stat(filepath.Join(r.Dir, "go.mod")); err == nil {
		return r
	}
	r.With = append(r.With, "MISE_GO_VERSION="+strings.TrimPrefix(runtime.Version(), "go"))
	return r
}

// walker is the steps run on a repo in their order. The steps that change the catalogs run on a new repo only, never
// on the template where it is (inPlace).
type walker struct {
	repo
	inPlace bool
}

// newOnly is why a step that changes the catalogs doesn't run on the template where it is.
const newOnly = "it changes the catalogs, so the walk runs it on a new repo (mise run new-repo:e2e), not on the template"

// walkOf is the walk of a repo walk.Run has set up (mise install, then its setup task): first, what it runs.
func walkOf(r walk.Repo) *walker {
	w := &walker{repo: repo{Repo: r}, inPlace: os.Getenv("E2E_STAMPED") != "true"}
	w.Pass("%s runs %s", w.Name, w.versions())
	return w
}
