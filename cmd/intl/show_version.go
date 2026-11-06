package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// showVersion prints this build's version (mise run intl:show-version): the version of mise-intl's module go.mod
// pins.
func showVersion() { fmt.Println(buildVersion()) }

// buildVersion is the version of mise-intl's module this command was built from: in a repo, the one its go.mod pins
// (go tool intl builds a dependency's command, so the module is one of the build's), "(devel)" in mise-intl's own
// checkout.
func buildVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	const module = "github.com/joeblew999/mise-intl"
	if bi.Main.Path == module {
		return "(devel)"
	}
	for _, d := range bi.Deps {
		if d.Path == module {
			if d.Replace != nil {
				return strings.TrimPrefix(d.Version, "v") + " (replaced by " + d.Replace.Path + ")"
			}
			return strings.TrimPrefix(d.Version, "v")
		}
	}
	return "(devel)"
}
