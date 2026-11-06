package e2e

import (
	"testing"

	misecheck "github.com/joeblew999/mise-check"
)

// TestMise is misecheck (github.com/joeblew999/mise-check) on a mise file of this repo and the Go it runs, both ways:
// root:check-mise checks this repo's mise.toml, template:check-mise the template's tasks against go tool intl's Go
// and against the walk that runs them.
func TestMise(t *testing.T) { misecheck.Check(t, misecheck.FromEnv()) }
