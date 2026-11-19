# Contributing

Thanks for helping. catude is a single Go binary with no CGO.

```sh
go test -race ./...     # all tests; they never read your real ~/.claude
go vet ./...
gofmt -l .              # must print nothing
go run ./cmd/catude snapshot   # render every tab as text
```

## Ground rules

- **Read-only by default.** A new write path needs a backup, an explicit key press and a test.
- **No secrets in output.** Anything that prints configuration goes through `internal/redact`. Add a test with a
  fake token that proves it never appears.
- **No new network calls** without discussion. No telemetry, ever.
- **Cross-platform.** Paths come from `internal/paths`; add a test for Windows, macOS and Linux layouts when you
  touch locations. CI runs on all three.
- **Exact or labelled.** If a number is an estimate, the UI must say so.
- Dependencies: pinned versions, at least two weeks old, and a reason in the PR.

## Pull requests

Small and focused, with a test. Describe what you changed and how you checked it. AI-assisted contributions are
welcome if you have read, run and understood every line you submit.

## Catalog entries

See [docs/catalog.md](docs/catalog.md#adding-or-fixing-an-entry).

## Configuration presets

Add them to `internal/claudecfg/config-presets.json`. Every preset needs a public source and the exact sentence
that supports it; every key must be a documented setting. `go test ./internal/claudecfg` checks both.

## Interface text and translations

Write interface text in English inside `T("…")` or `F("… %s …", x)`, then run `node scripts/i18n-extract.mjs`.
`go test ./internal/i18n` fails until all nine catalogs in `internal/i18n/locales/` have the new message with the
same placeholders. Use the terms of each language's glossary in `docs/translations/`.

## Screenshots and GIFs

They are generated from an invented machine, never taken by hand: see [scripts/demo](scripts/demo/README.md).
Regenerate them when a change is visible in the README images.

## Writing style

Plain sentences. No em dashes, no marketing adjectives, no sentences that only repeat the previous one.
