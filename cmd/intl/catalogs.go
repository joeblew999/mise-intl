package main

import (
	"fmt"
	"path/filepath"
)

// catalogs is go tool intl catalogs: the catalogs' Go package name (locales), for scripts.
func catalogs(args []string) {
	c := app()
	fmt.Println(filepath.Base(c.CatalogDir()))
}
