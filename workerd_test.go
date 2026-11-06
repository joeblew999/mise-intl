package e2e

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"github.com/joeblew999/mise-intl/intl/workerd"
)

// TestWorkerdPins: the template's package.json (the app's own, which go tool intl template writes) pins the workerd
// intl is tested with, intl/workerd's Version, and its lockfile has it.
func TestWorkerdPins(t *testing.T) {
	b, err := os.ReadFile("templates/package.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		t.Fatal(err)
	}
	if v := pkg.DevDependencies["workerd"]; v != workerd.Version {
		t.Errorf("templates/package.json pins workerd %q, not intl/workerd's Version %s", v, workerd.Version)
	}
	lock, err := os.ReadFile("templates/pnpm-lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^      workerd:\n        specifier: ` + regexp.QuoteMeta(workerd.Version) + `\n`).Match(lock) {
		t.Errorf("templates/pnpm-lock.yaml doesn't have workerd %s: mise run template:setup after package.json moves (aube install)", workerd.Version)
	}
}
