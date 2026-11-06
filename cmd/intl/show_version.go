package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// showVersion prints this build's version (mise run intl:show-version): the version of mise-intl's module go.mod
// pins.
func showVersion() { fmt.Println(buildVersion()) }

// buildVersion is the version of mise-intl's module this command was built from: in a repo, the one its go.mod pins.
// go tool intl builds the command as its module's own main package (go version -m: mod github.com/joeblew999/mise-intl
// v0.1.1), so it's the build's main module. "(devel)" in mise-intl's own checkout, or replaced by a folder.
func buildVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok || bi.Main.Version == "" || bi.Main.Version == "(devel)" || bi.Main.Replace != nil && (bi.Main.Replace.Version == "" || bi.Main.Replace.Version == "(devel)") {
		return "(devel)" // mise-intl's own checkout, or a go.mod that replaces the module with one
	}
	return strings.TrimPrefix(bi.Main.Version, "v")
}
