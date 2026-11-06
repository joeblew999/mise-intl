package miseintl

import (
	"os/exec"
	"strings"
	"testing"
)

// TestWithoutGsx: only gsxintl imports gsx. intl, its command and everything else build without it, so an app
// without gsx never gets gsx in its go.mod, not even through go tool intl.
func TestWithoutGsx(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "./intl/...", "./cmd/intl", "./gsxintl/filters", "./gsxintl/registry", "./gsxintl/screens", "./internal/...").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, p := range strings.Fields(string(out)) {
		if strings.HasPrefix(p, "github.com/gsxhq/gsx") {
			t.Errorf("%s is built with gsx: only gsxintl may import it", p)
		}
	}
}
