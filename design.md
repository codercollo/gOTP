# GoTP — Design

OTP-as-a-Service over SMS with a built-in fraud guard. Two endpoints
(`POST /v1/otp/send`, `POST /v1/otp/verify`), one SMS integration, no LLM and
no trained model — the fraud logic is auditable adaptive statistics.
Category: Fraud & SIM-Swap Detection.

**One line:** GoTP is OTP that pays for itself — it blocks the fraudulent
sends (SMS-pumping, OTP-bombing) that other providers bill you for.

---

## Engineering conventions

- `cmd/api` layout; `config` struct from flags/env; an `application` struct
  with dependency injection; httprouter; JSON envelope with
  `readJSON`/`writeJSON`; centralised JSON error responses; a `validator`
  package (→ 422); structured JSON logging; per-IP rate limiter; API-key
  auth middleware; a `background()` helper; expvar metrics; graceful
  shutdown; a `Makefile` with an `audit` target.
- Explicit server timeouts (`ReadTimeout`, `ReadHeaderTimeout`,
  `WriteTimeout`, `IdleTimeout`); a `context` deadline on every outbound SMS
  call; always drain and close response bodies; signal-driven graceful
  shutdown that drains in-flight work before exit.

---

## Core principles

### 1. Concurrency-safe, atomic verify-and-consume

- Verifying an OTP is an **atomic check-and-consume**: a valid code is
  accepted and invalidated in one guarded step, so the same code can never
  be accepted twice under a race.
- The guard's per-number / per-prefix counters are **mutex- or
  atomic-guarded**.
- One owner per record; keep the critical section small. This is what makes
  the service trustworthy.

### 2. Concurrency tests as the reliability proof

- Fire many concurrent verifies at one code → assert it is consumed
  **exactly once**.
- Fire a burst at the guard → assert the counts are **exact** (no lost
  updates).
- These tests are the proof of correctness and a strong thing to demo.

### 3. Mock the store interface

- `data.Models` is the mockable seam: generate a mock and unit-test
  send/verify and the guard **without** hitting SMS or any external service.
- Fast, deterministic tests → confidence the demo won't break live.

### 4. Never store secrets in plaintext

- **OTP code:** stored hashed; verify by comparing hashes (like a password),
  never by string-comparing a stored plaintext code.
- **API key:** stored **SHA-256-hashed**; look up tenants by hash.

### Offload slow I/O, return fast

- Send SMS with `background()` so the handler returns immediately; no task
  queue.

---

## Deliberately left out (over-engineering for a one-day build)

- A database / migrations — the in-memory store is enough for the demo; add
  later behind `data.Store`.
- A task queue — `background()` covers the one async job.
- gRPC / gateways, session/refresh tokens, orchestration, multi-service
  compose — none earns its cost before Thursday.

---

## Request flow

**Send** (`POST /v1/otp/send`)

1. `authenticate` (API-key hash) → `rateLimit` (per IP) → `validate` (→ 422).
2. **Guard check** (pure, in-memory): velocity (token-bucket + EWMA/CUSUM per
   number & IP) and verify-ratio per prefix. Blocked → `429`, no SMS sent,
   reason logged (every block = an SMS fee saved).
3. Generate code with `crypto/rand`; store **hashed** with a TTL.
4. Send SMS via `background()` (context deadline, drain+close body); return
   immediately.

**Verify** (`POST /v1/otp/verify`)

1. `authenticate` → `rateLimit` → `validate`.
2. **Atomic check-and-consume**: compare hash, check expiry, enforce
   max-attempts lockout, invalidate on success — all in one guarded step.
3. Feed the outcome back to the guard (verify events update the ratio).

---

## The fraud guard (the moat)

- **Velocity** — token-bucket + EWMA/CUSUM per number and per IP catches
  bursts above the learned baseline (OTP-bombing / harassment).
- **Verify-ratio** — send-vs-verify ratio per number prefix; a prefix that
  receives floods but verifies almost none is SMS-pumping (AIT) and gets
  auto-throttled.
- Every decision is an explainable number, not a black box.

**Scope, stated honestly:** GoTP defends the _send_ endpoint (abuse, cost
inflation) and hardens _verify_ (expiry, max-attempts). It does **not** detect
a literal SIM-swap (needs telco SIM-change signals) — that's the telco half.
