package main

import (
	"fmt"
	"strings"
)

// locales is go tool intl locales: the shipped locales, comma-separated.
func locales(args []string) {
	c := app()
	fmt.Println(strings.Join(c.Locales, ","))
}
