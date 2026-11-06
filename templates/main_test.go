package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/joeblew999/mise-intl/intl/intltest"
)

func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

// TestHome: the page in English at /, and in German at /de/, each value as that language's Intl writes it.
func TestHome(t *testing.T) {
	for path, want := range map[string][]string{
		"/":    {`lang="en"`, "intl, in your language", "July 4, 2026", "€1,234.50", "Go, mise, and intl", "Japan", "This page is in 2 languages."},
		"/de/": {`lang="de"`, "intl, in Ihrer Sprache", "4. Juli 2026", "1.234,50 €", "Go, mise und intl", "vor 3 Tagen", "Japan", "Diese Seite gibt es in 2 Sprachen."},
	} {
		w := httptest.NewRecorder()
		handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		body := w.Body.String()
		for _, s := range want {
			if !strings.Contains(body, s) {
				t.Errorf("%s: no %q in\n%s", path, s, body)
			}
		}
	}
}
