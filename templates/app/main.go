// Command app is the template's little app: one page of what the runtime's own Intl writes in the reader's language,
// English or German, in plain Go (net/http, html/template, intl.Site and intl.Text; no gsx). go run ./app serves it at
// http://localhost:8080 (ADDR to change it); /de/ is the page in German. The same app runs on a Cloudflare Worker
// (GOOS=js GOARCH=wasm): serve is server.go's on a server, worker.go's on a Worker.
package main

import (
	"html/template"
	"net/http"
	"time"

	"github.com/joeblew999/mise-intl/intl"
	"github.com/joeblew999/mise-intl/templates/locales"
)

func main() { serve(handler()) }

// handler is the app: intl.Site picks the request's language (its URL's /de/ prefix, the reader's choice, or their
// browser's), and the page is written in it.
func handler() http.Handler {
	site := intl.Site{IsPage: func(path string) bool { return path == "/" }}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", home)
	return site.Handler(mux)
}

// row is one line of the page's table: what the value is, and the value in the request's language.
type row struct{ Label, Value string }

// home is the page: every word a message of locales/, every value Intl's.
func home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	at := time.Date(2026, 7, 4, 15, 30, 0, 0, time.UTC)
	page := struct {
		Lang, Dir, Title, Intro, Languages, Read string
		Rows                                     []row
		Links                                    []intl.Link
	}{
		Lang:      intl.Current(ctx).ID(),
		Dir:       intl.Current(ctx).Dir(),
		Title:     intl.Text(ctx, locales.HomeTitle),
		Intro:     intl.Text(ctx, locales.HomeIntro),
		Languages: intl.Text(ctx, locales.HomeLanguages(len(intl.Shipped()))),
		Read:      intl.Text(ctx, locales.HomeRead),
		Rows: []row{
			{intl.Text(ctx, locales.HomeRowDate), intl.Text(ctx, intl.Date(at, intl.Options("dateStyle", "long")))},
			{intl.Text(ctx, locales.HomeRowPrice), intl.Text(ctx, intl.Number("1234.5", intl.Options("style", "currency", "currency", "EUR")))},
			{intl.Text(ctx, locales.HomeRowList), intl.Text(ctx, intl.List([]string{"Go", "mise", "intl"}, ""))},
			{intl.Text(ctx, locales.HomeRowAgo), intl.Text(ctx, intl.Of("RelativeTimeFormat", "", intl.Options("numeric", "auto"), "-3", intl.JSON("day")))},
			{intl.Text(ctx, locales.HomeRowRegion), intl.Text(ctx, intl.Of("DisplayNames", "of", intl.Options("type", "region"), intl.JSON("JP")))},
		},
		Links: intl.Links(ctx),
	}
	if err := tmpl.Execute(w, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var tmpl = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="{{.Lang}}" dir="{{.Dir}}">
<head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}}</h1>
<p>{{.Intro}}</p>
<table>{{range .Rows}}
<tr><th>{{.Label}}</th><td>{{.Value}}</td></tr>{{end}}
</table>
<p>{{.Languages}}</p>
<nav><h2>{{.Read}}</h2>{{range .Links}}
<a href="{{.Href}}" hreflang="{{.Lang}}" lang="{{.Lang}}"{{if .Current}} aria-current="page"{{end}}>{{.Name}}</a>{{end}}
</nav>
</body>
</html>
`))
