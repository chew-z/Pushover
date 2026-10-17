---
title: Pushover open items
date: 2026-10-17
status: active
tags: [open-items]
---

# Open items

> Everything still unresolved: verified defects, debt, and decisions waiting on a human.
> Current state lives in [STATUS.md](STATUS.md).
> Resolved items move to docs/retired/ or are deleted; this file lists only what is open.

## Decisions waiting on a human

None.

## Defects

Verified against the code on 2026-10-17.

- **Emergency limits in the MCP schema exceed the API.** `expire` allows up to 86400 seconds (`mcp_server.go:47`)
  while Pushover caps emergency `expire` at 10800 seconds and `retry` at 50 attempts.
  The API refuses values above the cap; lower the schema maximum to 10800.
- **Emergency priority without retry and expire fails at send time.** The `send_notification` description does not say
  that `priority: 2` requires both (`mcp_server.go:24`); the library rejects the message instead.
- **Token generation requires Pushover keys it does not use.** `generateTokenFromArgs` calls `NewMCPConfig(false)`
  (`mcp_cli.go:104`), whose validation requires `APP_KEY` and `RECIPIENT_KEY` (`config.go:169`, `config.go:173`),
  although only `PUSHOVER_AUTH_SECRET_KEY` is needed. The tests set the keys to work around it (`main_test.go:1277`).
- **`/generate-token` handler is not mounted.** `handleGenerateToken` exists (`http_server.go:189`) but the mux
  registers only `/`, `/health` and `/capabilities` (`http_server.go:63`); `README.md` documents the endpoint anyway.
  Remove the handler or mount it behind authentication.
- **`/capabilities` drifts from the tool schema.** It omits `retry` and states the expire range loosely
  (`http_server.go:160`), while the schema defines both explicitly (`mcp_server.go:44`).

## Debt

- **Trading integration requests are unimplemented.** From `docs/HANDOFF-2026-10-17-trading.md`, in priority order:
  `url` and `url_title`; one image `attachment` (5 MiB, path or base64); `html` and `monospace` (mutually exclusive);
  `ttl`; `timestamp`; return `X-Limit-App-Remaining` and support emergency cancellation; a `send_glance` tool for
  Pushover Glances. The receipt is already returned (`mcp_server.go:167`). The underlying `gregdel/pushover` v1.4.0
  covers the message fields; the MCP schema and CLI flags need to grow.
- **README is stale.** The architecture section cites outdated line counts, and the HTTP section documents
  `POST /generate-token`, which is not routed. Build commands still describe the pre-merge Taskfile.
- **Unused auth helpers.** `getUserID` and `getRole` are unused and kept under `nolint:unused`
  (`auth.go:149`, `auth.go:167`); either use them or drop them.
- **No `golang-concurrency` skill installed.** Offered by the scaffold; see the decision above.
