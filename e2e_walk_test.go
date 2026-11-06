// Package e2e is this repo's end-to-end test: a walk of a repo's numbered tasks, through the repo's own mise file, on
// mise-check's walk (github.com/joeblew999/mise-check/walk), which runs the steps in their order and does everything
// that isn't intl's. It sits at the top of this repo, beside the root mise.toml whose tasks run it, in a file for each
// thing it does, as mise-cloudflare's does:
//
//   - e2e_walk_test.go, this file: TestWalk, the test the root's tasks run (which repo, REPO_PATH), and TestSteps.
//   - e2e_steps_test.go: the steps, the repo's numbered tasks in their order, each with its check.
//   - e2e_mise_test.go: the repo, and what's intl's in making one: a new repo (mise-intl init, then intl:template),
//     and the template where it is given its Go.
//
// tool_test.go is the release (mise run tool:release), mise_test.go checks that the mise files and the Go line up,
// both ways, workerd_test.go the workerd the template pins, and gsx_test.go that intl's packages need no gsx.
package e2e

import (
	"os"
	"testing"

	"github.com/joeblew999/mise-check/walk"
)

// intlWalk is the walk: the template's numbered tasks (templates/.mise/tasks/mise-intl.toml), in their order, through the
// repo's own mise file, each checked by what a person would see it do; on the template where it is (mise run
// template:e2e), and on a new repo made exactly as README.md's "Use it in your repo" says (mise run new-repo:e2e). mise-check's
// walk does the rest.
var intlWalk = walk.Walk[*walker]{Steps: steps, New: newRepo, Prepare: withGo, Of: func(r walk.Repo) *walker {
	return &walker{Repo: r, inPlace: os.Getenv("E2E_STAMPED") != "true"}
}}

func TestWalk(t *testing.T)  { intlWalk.Run(t) }
func TestSteps(t *testing.T) { walk.CheckSteps(t, "templates/.mise/tasks/mise-intl.toml", steps) }

// walker is intl's walk of a repo: what each step's Check is given. The steps that change the catalogs run on a new
// repo only, never on the template where it is (inPlace).
type walker struct {
	walk.Repo
	inPlace bool
}

// newOnly is why a step that changes the catalogs doesn't run on the template where it is.
const newOnly = "it changes the catalogs, so the walk runs it on a new repo (mise run new-repo:e2e), not on the template"
