//go:build !(js && wasm)

package main

import (
	"cmp"
	"log"
	"net/http"
	"os"

	"github.com/joeblew999/mise-intl/intl/workerd"
)

// serve on an ordinary Go server: Intl outside a Cloudflare Worker is workerd's, the repo's intl daemon (pitchfork.toml;
// run the app through mise: mise run serve).
func serve(h http.Handler) {
	if err := workerd.Start(); err != nil {
		log.Fatal(err)
	}
	addr := cmp.Or(os.Getenv("ADDR"), "localhost:8080")
	log.Printf("http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}
