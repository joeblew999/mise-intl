# intl-template

A Go web app in English and German with [intl](https://github.com/joeblew999/mise-intl): one page of what the
runtime's own Intl writes in the reader's language, dates, prices, lists, relative times and region names. Plain Go:
`net/http`, `html/template`, `intl.Site` and `intl.Text`, no gsx.

- `go run ./app` (`mise run serve`): the app, at http://localhost:8080. `/de/` is the page in German.
- `go test ./...` (`mise run test`): its tests.
- `go tool intl` (`mise run intl`): what's done for the app's languages, and what to run next.
- [docs/intl/README.md](docs/intl/README.md): intl in this repo, every command.

intl comes from one line of `go.mod`, with no binary; its tools come from mise: `mise.toml` pins Go (from `go.mod`)
and workerd, which intl's Intl runs on.
