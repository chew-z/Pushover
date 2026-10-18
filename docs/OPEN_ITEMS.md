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

None.

## Defects

None.

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
- **The SpaceGrey post-commit hook is not executable.** `.git/hooks/post-commit` is mode `0644`,
  so Git reports that it was ignored because it is not executable.
  Fix the executable bit on GoldenAir through `fix-exec-bits`, not in this checkout.
- **Live check pending: do Glance responses carry `X-Limit` headers?** The client accepts a successful response without quota headers
  and reports quota as unknown (`glance.go:110-120`).
- **No `golang-concurrency` skill is installed.** The scaffold offers it, but this checkout contains only the `golang-modern` project skill.
