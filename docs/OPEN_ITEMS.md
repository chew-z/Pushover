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

- **Acceptance sweep.** The connector plan's second criterion (priorities -1 to 1 with URL, a chart attachment, a glance on the watch) has not been run;
  only login plus `send_notification` is recorded (`STATUS.md:37`).
- **Redeploy with the loopback bind.** `tmp/pushover.service` and the OPERATIONS sample now bind `127.0.0.1:8092`;
  the unit on karma.rrj.pl still binds `:8092` until it is redeployed and `:8092` is confirmed closed from outside.
- **First tag.** `v1.0.0` is agreed and the build is ready for it (`main.go:15` defaults to `dev`); the tag has not been created.

## Defects

Fixed on 2026-10-18 in `d505362`, `5fc5e59`, `e731dec`, `6e69f6b` and the docs commit that follows them: see `STATUS.md`.
Still open from that review:

- **`TestMCPSchemaValidation` never swaps in the mock** (`main_test.go:351`), so it exercises the schema only through a client it cannot observe.
- **`exported` is inert on the root package.** revive skips `package main`, so the doc-comment rule enforced since `6e69f6b` covers only `internal/oauth`.
- **The per-pending `maxLoginAttempts` counter resets with every fresh `GET /authorize`,** so it bounds guessing per form, not per attacker;
  the per-IP bucket and the login semaphore are the real limits.

## Debt

- **OAuth fixes must be ported manually across three copies.** The OAuth server exists in Alerts, polcritique, and Pushover's `internal/oauth` copy.
  A shared module remains deferred, so a security or correctness fix must be applied to each copy.
- **OAuth restart continuity is incomplete.** HTTP construction creates a new in-memory `oauth.Store` (`http_server.go:47-65`),
  which loses client registrations and refresh tokens on restart (`internal/oauth/oauth_store.go:133-164`).
  A still-valid access JWT continues to pass the stateless bearer check (`auth.go:82-98`),
  but after it expires the absent refresh token forces a fresh authorization and login (`internal/oauth/oauth_token.go:177-203`).
- **CLI zero values cannot override defaults.** `PUSHOVER_EXPIRE` and `PUSHOVER_RETRY` set expiry and retry defaults,
  which otherwise are 180 and 60 seconds; since `5fc5e59` an explicit value is taken as given and range-checked rather than replaced.
  `ParseArgs` tracks whether only priority was supplied.
  Message construction treats `-ttl 0`, `-timestamp 0`, `-e 0` and `-r 0` as unset (`pushover.go:70-130`).
- **Typo guard rejects some one-word messages.** A first word within one edit or one transposition of `mcp`, `cancel` or `glance`
  (for example `map`) is refused as a misspelled subcommand (`main.go:21-39`); use `-m` for such a message.
- **Port the 2026-10-18 OAuth fixes to Alerts and polcritique:** rate-limit tests, wrong-secret test, login semaphore, consent text, comment repairs (`e731dec`).
- **Live check pending: do Glance responses carry `X-Limit` headers?** The client accepts a successful response without quota headers
  and reports quota as unknown (`glance.go:110-120`).
- **No `golang-concurrency` skill is installed.** The scaffold offers it, but this checkout contains only the `golang-modern` project skill.
