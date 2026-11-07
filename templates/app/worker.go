//go:build js && wasm

package main

import (
	"net/http"

	"github.com/syumai/workers-go"
)

// serve on a Cloudflare Worker (workers-go's glue): Intl is the Worker's own, which its entry module gives Go
// (globalThis.goIntl, from intl's modules: mise-intl assets --worker).
func serve(h http.Handler) { workers.Serve(h) }
