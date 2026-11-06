package e2e

import (
	"testing"

	misecheck "github.com/joeblew999/mise-check"
)

// TestRootMise is misecheck (github.com/joeblew999/mise-check) on this repo's own mise.toml, its mise side: the root
// runs Go, it isn't a binary's tasks (mise run root:check-mise).
func TestRootMise(t *testing.T) { misecheck.Check(t, misecheck.Config{MiseFile: "mise.toml"}) }

// TestTemplateMise is misecheck on the template's tasks, both ways: against mise-intl's Go and against the walk that
// runs them (mise run template:check-mise).
func TestTemplateMise(t *testing.T) {
	misecheck.Check(t, misecheck.Config{MiseFile: "templates/.mise/tasks/mise-intl.toml", GoCmd: "cmd", GoLayout: "file", GoCallers: "."})
}
