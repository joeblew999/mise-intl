package e2e

import (
	"os"
	"regexp"
	"testing"

	"github.com/joeblew999/mise-intl/intl/workerd"
)

// TestWorkerdPins: the template's mise.toml (the app's own, which go tool intl template writes) and this repo's own
// mise.toml pin the workerd intl is tested with, intl/workerd's Version.
func TestWorkerdPins(t *testing.T) {
	for _, f := range []string{"templates/mise.toml", "mise.toml"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		m := regexp.MustCompile(`(?m)^workerd = "([^"]+)"`).FindSubmatch(b)
		if m == nil || string(m[1]) != workerd.Version {
			t.Errorf("%s pins workerd %q, not intl/workerd's Version %s", f, m, workerd.Version)
		}
	}
}
