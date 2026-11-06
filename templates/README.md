# intl-template

A Go web app in English and German with [intl](https://github.com/joeblew999/mise-intl): one page of what the
runtime's own Intl writes in the reader's language, dates, prices, lists, relative times and region names. Plain Go:
`net/http`, `html/template`, `intl.Site` and `intl.Text`, no gsx.

- `go run ./app` (`mise run serve`): the app, at http://localhost:8080. `/de/` is the page in German.
- `go test ./...` (`mise run test`): its tests.
- `mise-intl` (`mise run intl`): what's done for the app's languages, and what to run next.
- [docs/intl/README.md](docs/intl/README.md): intl in this repo, every command.

Every tool comes from mise: `.mise/config.toml` pins mise-intl (and Go, `go.mod`'s), `mise.toml` pins node and aube,
and `mise run setup` installs workerd, which intl's Intl runs on, from `package.json` (`aube install`), and the modules
`go.mod` pins, intl's packages among them.
