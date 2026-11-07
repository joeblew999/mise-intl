package e2e

import (
	"testing"

	misecheck "github.com/joeblew999/mise-check"
)

// TestRootMise is misecheck (github.com/joeblew999/mise-check) on this repo's own mise.toml, its mise side (the root
// runs Go, it isn't a binary's tasks), and the shape every mise tool repo has: root:ci and root:verify, a ci.yml that runs
// only mise run root:ci, a README that opens with the problem and what mise-intl does (mise run root:check-mise).
func TestRootMise(t *testing.T) {
	misecheck.Check(t, misecheck.Config{MiseFile: "mise.toml"})
	misecheck.CheckToolRepo(t, ".")
}

// TestTemplateMise is misecheck on the template's tasks, both ways: against mise-intl's Go, against the walk that
// runs them, and against the template's skill, which names every task and flag (mise run template:check-mise).
func TestTemplateMise(t *testing.T) {
	misecheck.Check(t, misecheck.Config{
		MiseFile:  "templates/.mise/tasks/mise-intl.toml",
		GoCmd:     "cmd",
		GoLayout:  "file",
		GoCallers: ".",
		Skills:    []string{"templates/.claude/skills/intl/SKILL.md"},
	})
}
