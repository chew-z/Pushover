---
title: Pushover status
date: 2026-10-17
status: active
tags: [status]
---

# Status

> Where the project stands today.
> Design lives in [ARCHITECTURE.md](ARCHITECTURE.md); unresolved work in [OPEN_ITEMS.md](OPEN_ITEMS.md).

## Toolchain

- `go` directive: `1.27.0` (`go.mod`), upgraded from `1.26.2` on 2026-10-17 with `go fix` applied.
- Installed Go: `go1.27.2 darwin/arm64`.
- `golangci-lint`: `2.14.0` (built with go1.27.1).
- `markdownlint-cli2`: `0.23.3` (markdownlint `0.41.1`).
- `task`: `3.54.0`.

## Health

- `task check` (fmt:check + vet + lint:ci + test:short) passes as of 2026-10-17.
- Test coverage: 83.8% of statements (`task test`, 2026-10-17).
- `task build` produces `bin/pushover` with the version injected from `git describe`.
- No deployment is recorded here; the HTTP transport is run behind an nginx reverse proxy, as the comment at `config.go:54` describes.

## Recent changes

- 2026-10-18 — Added Pushover Glances support through the `send_glance` MCP tool and `glance` CLI subcommand,
  including explicit empty-field clearing, quota reporting, and real-library response-path tests.
- 2026-10-17 — Notification inputs and MCP parity expanded: added a testable client seam; cancellation and Glances client methods;
  explicit priority zero; URL/title, HTML/monospace, TTL and timestamp support; shared send-result formatting; the 10,800-second
  emergency expire cap; and `/capabilities` metadata derived from registered MCP tools.
- 2026-10-17 — Scaffold re-initialization finished: Taskfile merged, lint and markdown configs refreshed,
  docs filled from the code, `.gitignore` corrected so gate config and rules are versioned.
  Fixed version injection by making `main.version` a variable (`main.go:11`).
- 2026-10-17 — Go 1.27 upgrade: directive bumped to `1.27.0`, `go fix` applied until stable, dependencies tidied.
- 2026-10-17 — Release flow folded into `task release`; `run_release.sh` removed.
  The stray nginx snippet at the repository root was removed.
- 2026-10-17 — Handoffs recorded: Trading integration requests and Pushover Glances (`docs/HANDOFF-2026-10-17-trading.md`).
- 2026-07-25 — Unit tests for auth, CORS and CLI handlers; `errorlint` enabled; structured JSON logging with `slog`;
  dependency checksums and golangci-lint config tracked; CI workflow and release automation; Makefile replaced by Taskfile.
- 2026-07-11 — HTTP transport can disable mcp-go's localhost protection for reverse-proxy deployments.
- 2026-05-05 — `retry` parameter support for emergency messages.

## Notes

- `go.sum` stays tracked. The scaffold template ignores it; this project keeps module checksums in Git
  so a fresh clone verifies dependencies without a network round-trip.
