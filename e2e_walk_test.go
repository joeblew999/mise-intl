// Package e2e is this repo's end-to-end test: a walk of a repo's numbered tasks, through the repo's own mise file, on
// mise-check's walk (github.com/joeblew999/mise-check/walk), which runs the steps in their order and does everything
// that isn't intl's. It sits at the top of this repo, beside the root mise.toml whose tasks run it, in a file for each
// thing it does:
//
//   - e2e_walk_test.go, this file: the walk tests the root's tasks run by name (TestWalkTemplate, TestWalkNewRepo),
//     and TestSteps.
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
// checked. Which repo is the test's name: the template where it is, or a new repo made the README's way (at the
// release -version names or this checkout's build).
var intlWalk = walk.Walk[*walker]{Steps: steps, New: newRepo, Prepare: prepare, Of: walkOf}

// TestWalkTemplate walks the template, templates/, where it is (mise run template:e2e). Each walk runs only when its
// task names it, so go test ./... never walks.
func TestWalkTemplate(t *testing.T) { intlWalk.Template(t, "templates") }

// TestWalkNewRepo walks a new repo made the README's way, the steps that change the catalogs too (mise run
// new-repo:e2e).
func TestWalkNewRepo(t *testing.T) {
	walk.OnlyByName(t)
	walkNew = true
	intlWalk.NewRepo(t)
}

// walkNew is whether the walk is on a new repo, which the steps that change the catalogs run on too.
var walkNew bool

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
	r.With = append(r.With, "INTL_PITCHFORK_DIR=") // the repo's own intl daemon, not this repo's tests' (mise.toml)
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

// walkOf is the walk of a repo the walk has set up (mise install, then its setup task): first, what it runs.
func walkOf(r walk.Repo) *walker {
	w := &walker{repo: repo{Repo: r}, inPlace: !walkNew}
	w.T.Cleanup(func() { w.Run(nil, "pitchfork", "stop", "intl") }) // its intl daemon, before its folder goes
	w.Pass("%s runs %s", w.Name, w.versions())
	return w
}
