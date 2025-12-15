# Repository Guidelines

## Project Structure & Module Organization
The module root hosts the exported API (`stow.go`, `store.go`, `namespace*.go`), while helpers and background workers live in `internal/`. Documentation assets are under `docs/`, runnable walkthroughs in `examples/` (e.g., `examples/basic`), manual datasets in `data/`, and regression suites in `tests/`. Keep assets such as large fixtures outside tracked namespaces to avoid polluting `_blobs`.

## Build, Test, and Development Commands
Install Go ≥1.21 (go.mod currently targets 1.24) and work with modules enabled.
- `go test ./... -v` — run every package test, including the storage engine.
- `go test ./tests -race -v` — execute the integration suite with the race detector.
- `go test ./tests -bench=. -benchmem` — reproduce the published benchmarks.
- `go run examples/basic/main.go` (or any other example folder) to validate docs.
Use `go vet ./...` before opening a PR when modifying concurrency, caching, or blob routing.

## Coding Style & Naming Conventions
Always format Go sources via `gofmt`/`goimports` (tabs for indentation) and keep filenames descriptive and lowercase (`namespace_config.go`). Exported identifiers should read like GoDoc titles (`NamespaceConfig`, `MustOpen`), while unexported helpers stay camelCase. Favor structured logging through `logger.go`, and never commit generated JSONL data or blob payloads.

## Testing Guidelines
Unit tests mirror the package under test and follow the `TestFeatureName` pattern. Integration tests under `tests/` rely on temporary directories—reuse helpers from `filedata.go` to avoid leaks. Add regression coverage whenever you touch storage paths, blob rules, namespace options, or concurrency primitives. For features involving goroutines or file watchers, run `go test ./tests -race` locally and keep coverage comparable to the README matrix (KV ops, blobs, history, compact, GC, concurrency).

## Commit & Pull Request Guidelines
Commits use short, imperative subjects (`Add compaction throttle`). Include a body when changing persistent formats or lock ordering. Pull requests should explain the motivation, summarize affected namespaces/files, and link issues. For bug fixes, document reproduction steps; for performance work, include before/after benchmark snippets; attach logs or screenshots only when CLI output changes.

## Security & Configuration Tips
Never place secrets inside `data/` or checked-in namespaces. Keep new configuration toggles documented both in `namespace_config.go` and `README.md`, and default them to safe values. When adjusting blob routing or compaction thresholds, describe the guardrails against unbounded file growth and back the change with tests in `tests/gc_*`.
