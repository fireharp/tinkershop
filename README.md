# tinkershop

Local activity detector for Tinkershop.

The Go daemon is the extraction layer. It scans local programming projects and
agent-session stores, writes normalized state to SQLite, and keeps raw evidence
in a content-addressed blob store. Publishing is deliberately separate: a site,
wiki, WordPress plugin, or other consumer can read SQLite/blobs and render its
own view.

## Commands

```bash
go test ./...
go run ./cmd/tinkershop scan
go run ./cmd/tinkershopd -interval 12h
go run ./cmd/tinkershop serve -addr 127.0.0.1:8739
```

Default storage:

- SQLite: `~/.local/share/tinkershop/tinkershop.sqlite`
- Blobs: `~/.local/share/tinkershop/blobs`
- Scan root: `~/Prog`

## Design

- `cmd/tinkershop`: CLI for scan, serve, and policy inspection.
- `cmd/tinkershopd`: daemon entrypoint for scheduled scans.
- `internal/collectors`: source-specific detectors.
- `internal/storage`: SQLite schema and queries.
- `internal/blobstore`: gzip-compressed content-addressed evidence store.
- `internal/policy`: allow/block/review decisions.

Raw transcript parsing belongs in collectors. Consumer-specific exports do not
belong in this module.

## Development

```bash
task fmt      # gofmt + goimports
task vet      # go vet ./...
task lint     # golangci-lint
task test     # go test ./...
task ci       # fmt + vet + lint + test
```

Install pre-commit/pre-push hooks once with `lefthook install` (uses
[`lefthook.yml`](lefthook.yml)). Hooks run `gofmt`/`go vet` on commit and
`task lint`/`task test` on push.

## Releases

Versioning and changelog are automated by
[release-please](https://github.com/googleapis/release-please) (Go release
type). Use [Conventional Commits](https://www.conventionalcommits.org/) on
`main` and a release PR will be opened/updated automatically; merging it tags
the new version and writes `CHANGELOG.md`.
