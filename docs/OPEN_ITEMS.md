---
title: Pushover open items
date: 2026-10-18
status: active
tags: [open-items]
---

# Open items

> Everything still unresolved: verified defects, debt, and decisions waiting on a human.
> Current state lives in [STATUS.md](STATUS.md).
> Resolved items move to docs/retired/ or are deleted; this file lists only what is open.

## Decisions waiting on a human

- **Misspelled subcommand sends a notification.** `hasSubcommand` (`main.go:16-28`) knows three names; anything else becomes the message body (`cli.go:166-173`).
  Drop the positional fallback, or reject near-misses of a subcommand name.
- **Fail-closed configuration.** Making a malformed `.env` value a startup error is a behavior change on the production host.
- **First git tag.** Version injection cannot be made honest without one.
- **Static token lifetime.** Access and static JWTs cannot be revoked before expiry; `protectMCP` (`auth.go:123`) never consults the store.
  The static token default is 744h (`mcp_cli.go:75`). Keep, shorten, or add a denylist.
- **Acceptance sweep.** The connector plan's second criterion (priorities -1 to 1 with URL, a chart attachment, a glance on the watch) has not been run;
  only login plus `send_notification` is recorded (`STATUS.md:36`).

## Defects

Confirmed on 2026-10-18 by re-verification of the week-in-review report; fix order in `docs/reports/2026-10-18-fix-plan.md`.

- **A delivered notification is reported as a failure.** `handleSendNotification` (`mcp_server.go:265-268`) and `Run` (`cli.go:222-225`)
  treat `pushover.ErrInvalidHeaders` as a send error,
  but the library returns it only after Pushover accepted the message (`request.go:38-45` in gregdel/pushover v1.4.0).
  `handleSendGlance` (`mcp_server.go:309`) excludes it correctly. A caller that retries produces duplicates.
- **Configuration fails open.** `parseEnvBool`, `parseEnvDuration` (`config.go:278-300`) and `parseLogLevel` (`logging.go:18-20`) substitute defaults on parse errors,
  and both `PUSHOVER_PRIORITY` reads ignore the `Atoi` error (`config.go:86-89`, `164-167`), so a malformed priority becomes `-1`.
  `Config` from `LoadConfig` has no `Validate`; only `MCPConfig` does (`config.go:191`), so the CLI runs unvalidated.
- **The consent screen names another product.** `oauth_authorize.go:384` says "manage your watchlist as you". Copied from Alerts in `0a11331`.
- **Rate limiting on `/authorize`, `/login` and `/token` has no test.** Deleting the `AllowRequest` calls (`oauth_authorize.go:56`, `:167`) leaves the suite green.
  The per-pending `maxLoginAttempts` counter resets with every fresh `GET /authorize`, so it does not bound guessing.
- **A wrong client secret is never tested.** Disabling the comparison at `oauth_token.go:130` leaves the suite green;
  `TestTokenEndpointClientAuthentication` (`oauth_flow_test.go:388`) only ever presents the correct secret.
- **Login has no concurrency bound.** `credentialsValid` (`oauth_authorize.go:277`) runs 600,000 PBKDF2 iterations per call with no semaphore,
  unlike the metadata fetch path (`fetchSlots`, `oauth_clients.go:80-94`).
- **`formatCancelResult` says "(sent by ...)"** (`mcp_server.go:321`), and `cancel_test.go:24` asserts that text.
- **Contradictory ttl and timestamp messages.** `pushover.go:74-79` reports "must be at least 1 second" while rejecting only negatives.
- **Three comment artifacts and a stale doc comment.** Truncated `//nolint:gocyclo` directives at `oauth_token.go:83`, `oauth_clients.go:148`, `:212`
  for a linter that is off; `oauth_metadata.go:92-98` opens with the old lowercase name and stray text.
- **Deployed bind address bypasses nginx.** The staged unit sets `PUSHOVER_HTTP_ADDRESS=:8092` and `docs/OPERATIONS.md:86` repeats it,
  while `OPERATIONS.md:67` and `deploy/nginx-pushover.conf:11` assume loopback. The login form is reachable over cleartext HTTP.
- **Three version strings.** No git tags, so `Taskfile.yml:13` yields a hash; `ci.yml:27` has no ldflags so CI reports `"1.0.0"` (`main.go:13`);
  `task release` falls back to `v0.0.0`. The build fingerprint (`Taskfile.yml:30-35`) ignores the version.
- **`GOLANG.md` cites wrong lines.** `pushover.go:61`, `auth.go:102` and `mcp_server.go:99` point at unrelated code; line 60 denies the `internal/oauth` subpackage.
- **Help omits `-r`/`-retry` and `PUSHOVER_RETRY`** (`cli.go:47`, `mcp_cli.go:32`) though both are registered and documented.
- **Tests that cannot detect regressions.** `mcp_tools_test.go:182` computes its expectation with the function under test;
  `TestStoreIsRaceFree` has no assertions and `-race` is never run (`Taskfile.yml:51-61`, `run_test.sh:13`);
  `TestMCPSchemaValidation` (`main_test.go:351`) never swaps in the mock.
- **Gate omits markdown lint** (`Taskfile.yml:112-118`, no CI step); the dialogue rule skips `docs/` and misses blockquoted replicas (`dialogue-paragraph.cjs:30`).
- **`docs/ARCHITECTURE.md:64`** claims a configured idle timeout; `http_server.go:79` derives it as twice the HTTP timeout.

## Debt

- **OAuth fixes must be ported manually across three copies.** The OAuth server exists in Alerts, polcritique, and Pushover's `internal/oauth` copy.
  A shared module remains deferred, so a security or correctness fix must be applied to each copy.
- **OAuth restart continuity is incomplete.** HTTP construction creates a new in-memory `oauth.Store` (`http_server.go:47-65`),
  which loses client registrations and refresh tokens on restart (`internal/oauth/oauth_store.go:133-164`).
  A still-valid access JWT continues to pass the stateless bearer check (`auth.go:82-98`),
  but after it expires the absent refresh token forces a fresh authorization and login (`internal/oauth/oauth_token.go:177-203`).
- **CLI zero values cannot override defaults.** `PUSHOVER_EXPIRE` and `PUSHOVER_RETRY` set expiry and retry defaults,
  which otherwise are 180 and 60 seconds (`config.go:79-80`, `config.go:96-106`).
  `ParseArgs` tracks whether only priority was supplied (`cli.go:126-153`).
  Message construction ignores `-ttl 0` and `-timestamp 0`, and uses non-zero checks for `-e 0` and `-r 0` (`pushover.go:97-142`).
- **Live check pending: do Glance responses carry `X-Limit` headers?** The client accepts a successful response without quota headers
  and reports quota as unknown (`glance.go:110-120`).
- **No `golang-concurrency` skill is installed.** The scaffold offers it, but this checkout contains only the `golang-modern` project skill.
