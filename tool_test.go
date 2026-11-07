// The root's tool: task that isn't a one-line command: mise run tool:release. It runs only from its own task, so go
// test ./... never releases.

package e2e

import (
	"testing"

	"github.com/joeblew999/mise-check/release"
)

// TestToolRelease: mise run tool:release releases mise-intl from the commit checked out, as release.yml runs it on
// GitHub (mise-check's release): the commit's v* tag or the next patch version, goreleaser's binaries on its GitHub
// release (.goreleaser.yaml), a new repo made pinning it and walked (mise run new-repo:e2e --version), and the version
// on proxy.golang.org. With --dry-run it tags and publishes nothing: goreleaser builds a snapshot into dist/.
func TestToolRelease(t *testing.T) { release.Run(t, release.Config{NewRepo: "new-repo:e2e"}) }
