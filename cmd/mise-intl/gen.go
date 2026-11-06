package main

import (
	"os"

	"github.com/joeblew999/mise-intl/internal/setup"
)

// gen is mise-intl gen (06, mise run intl:06-gen): the catalogs compiled into the app's messages package
// (intl_gen.go, intl-messages.mjs), the gsxui translator, and intl.sum.
func gen(args []string) {
	c := app()
	p := read(c)
	must(p.Generate(os.Stderr))
	must(setup.GsxuiRenderer(c.Root, c.Gsxui, os.Stderr))
}
