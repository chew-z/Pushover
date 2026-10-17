# GOLANG.md

Go conventions for `Pushover`.
The house rules below hold across projects;
the sections after them are specific to this project.

## Go version

The target release is Go 1.27; the `go` directive in `go.mod` is the source of truth.
Language features, `encoding/json/v2`, `go fix` modernization and version discipline
live in the `golang-modern` skill; use it rather than restating them here.

The directive is `1.27.0`.

## Taskfile

Every Go project has a `Taskfile.yml`; it is the one place commands are defined.

- If the project has none, add one. If it no longer covers the project —
  a new binary, a code-generation step, a deploy or release target — update it
  in the same change.
- Expected tasks: `default` (lists tasks), `build`, `test`, `lint`, `fmt`,
  and `check` — the gate that runs at least fmt/vet, lint and test.
- The gate rule is `quality-gate.md` in the tool's rules folder (`.claude/rules/` or `.qoder/rules/`).

## Build output

All binaries go into `bin/`, which is gitignored.
Build through the Taskfile, or `go build -o bin/<name> <pkg>` — never into the repo root.

## House rules

- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`; match with `errors.Is` / `errors.As`.
- `ctx context.Context` is the first parameter of every function that does I/O.
- Keep `main` thin: exit with `os.Exit` from `main` or a `run()` wrapper;
  never `log.Fatal` or `log.Panic`.
- No `io/ioutil`; use `os.ReadFile`, `os.WriteFile` and friends.
- Files are `snake_case.go`, named by concern (`config.go`, `errors.go`, `logging.go`);
  packages are single lowercase words.
- Layout follows size: a small tool stays a flat `package main` at the root;
  a service with real subsystems uses `cmd/<name>/main.go` plus `internal/<pkg>/`.
- Tests sit beside the code as `*_test.go`: stdlib `testing`, table-driven with `t.Run`,
  `t.Helper()` in helpers, `t.TempDir()` / `t.Setenv()` for state.
  Use testify only where the project already does.
- Lint with `golangci-lint` and format with `gofmt`; both clean before done.

## Stack

- `github.com/mark3labs/mcp-go` — MCP server, streamable HTTP transport, and the in-process client used by tests.
- `github.com/gregdel/pushover` — Pushover API client and message model.
- `github.com/golang-jwt/jwt/v5` — JWT issuance and validation for the HTTP transport.
- `github.com/joho/godotenv/autoload` — `.env` loading, blank-imported in `main.go`.
- Standard library for everything else: `flag`, `log/slog`, `net/http`.
- No database driver: the project has no database.

Tests use the standard library and a local mock implementing `PushoverClient`; testify is not a dependency.

## Project conventions

- **Flat package.** All Go files stay in one `package main` at the repository root, split by concern.
  Introduce `cmd/` and `internal/` only if real subsystems appear.
- **CLI over environment over defaults.** `CreateMessage` applies this order (`pushover.go:61`).
  `CLIArgs.PrioritySet` preserves whether a priority was supplied, so an explicit priority, including `-p 0`,
  overrides `PUSHOVER_PRIORITY`.
- **MCP input constraints belong in the tool schema.** `server.WithInputSchemaValidation()` is enabled
  (`mcp_server.go:18`); put type, range and length limits in `mcp.With*` helpers, not in the handler.
- **STDIO logging goes to stderr.** JSON `slog` writes to stderr (`logging.go:27`) because stdout carries
  JSON-RPC frames in STDIO transport; never print logs to stdout there.
- **JWT details.** HS256 only, issuer `pushover-mcp`, audience `pushover-mcp-user`, 60-second leeway
  (`auth.go:102`). Authentication is enforced only for HTTP requests (`mcp_server.go:99`); STDIO is trusted.
- **Version injection.** `main.version` must stay a string variable so `-ldflags "-X main.version=..."`
  works (`main.go:11`); a constant silently ignores the linker flag.
- **Deliberate lint exclusion.** `.golangci.yml` excludes `errcheck` for stdout/stderr writes,
  `os.Setenv` and JSON encoding (`fmt.Fprint*`, `(*encoding/json.Encoder).Encode`).
  Keep that block when refreshing the scaffold config.
