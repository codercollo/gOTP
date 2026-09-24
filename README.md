<p align="center">
  <img src="assets/logo.png" alt="GoTP" width="120" height="120">
</p>

<h1 align="center">GoTP</h1>

<p align="center"><strong>OTP that pays for itself.</strong></p>

<p align="center">
  SMS one-time-password delivery and verification, with a built-in fraud guard
  that blocks the abusive sends other providers bill you for.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-%3E=1.22-00ADD8?style=flat-square" alt="Go version">
  <img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/deps-httprouter-4f46e5?style=flat-square" alt="Dependencies">
  <img src="https://img.shields.io/badge/database-none-16a34a?style=flat-square" alt="No database">
</p>

---

## Install

```shell
go install github.com/codercollo/gOTP/cmd/api@latest
```

Or run from source:

```shell
git clone https://github.com/codercollo/gOTP
cd gOTP
make run/api
```

Or with Docker:

```shell
docker build -t gotp -f build/Dockerfile .
docker run -p 4000:4000 --env-file .env gotp
```

Then open <http://localhost:4000> for the dashboard.

## Why GoTP

OTP send endpoints are the most abused API in fintech. Attackers flood them to
run **SMS-pumping** (traffic they profit from, on your bill) and **OTP-bombing**
(harassment). GoTP hardens both sides:

* **No black-box ML.** Every block is an auditable number — adaptive statistics,
  no model, no training data.
* **No database.** In-memory by default; deployable as an isolated instance.
* **One binary.** The dashboard is embedded; the image is just the binary.

## Quick start

```shell
# send a code
curl -X POST localhost:4000/v1/otp/send \
  -H 'Content-Type: application/json' \
  -d '{"phone_number":"+254712345678"}'

# verify it
curl -X POST localhost:4000/v1/otp/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone_number":"+254712345678","code":"123456"}'
```

In development the code is printed to the logs (SMS is a no-op without live
Africa's Talking credentials).

## API

| Method | Path                | Body                                | Success | Blocked            |
|--------|---------------------|-------------------------------------|---------|--------------------|
| POST   | `/v1/otp/send`      | `{ phone_number }`                  | `202`   | `429` / `403`      |
| POST   | `/v1/otp/verify`    | `{ phone_number, code }`            | `200`   | `422` / `429`      |
| GET    | `/v1/healthcheck`   | —                                   | `200`   | —                  |
| GET    | `/debug/vars`       | — (expvar metrics)                  | `200`   | —                  |

When `API_KEYS` is set, send the key in the `X-API-Key` header.

## The fraud guard

Four checks run before any SMS is sent. Each is pure, concurrency-safe, and
explainable:

1. **Velocity** — per-number / per-IP token bucket (OTP-bombing).
2. **Verify-ratio** — per-prefix send-vs-verify ratio (SMS-pumping / AIT).
3. **Fan-out** — one source spraying many distinct numbers (distributed abuse).
4. **SIM-swap** — optional check via the Africa's Talking Insights API.

Verify-side hardening: hashed codes, expiry, attempt lockout, and atomic
check-and-consume (a code can never be accepted twice, even under concurrency).

## Configuration

Config is read from flags or environment variables (env wins in containers).
See [`configs/.env.example`](configs/.env.example).

| Env               | Default       | Purpose                              |
|-------------------|---------------|--------------------------------------|
| `PORT`            | `4000`        | HTTP port                            |
| `OTP_LENGTH`      | `6`           | Code length                          |
| `OTP_TTL`         | `5m`          | Code lifetime                        |
| `AT_USERNAME`     | —             | Africa's Talking username            |
| `AT_API_KEY`      | —             | Africa's Talking API key             |
| `SIM_SWAP_CHECK`  | `false`       | Enable the SIM-swap check            |
| `API_KEYS`        | —             | Comma-separated tenant keys          |

## Development

```shell
make run/api     # run the server
make test        # go test ./...
make audit       # vet + race tests
make docker      # build the image
```

## License

[MIT](LICENSE)
