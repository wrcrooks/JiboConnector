# JiboConnector

Worker service for the [OpenJibo](https://github.com/wrcrooks/JiboExperiments) self-hosted cloud. Runs as a separate Docker container alongside `jibo-api`.

## Purpose

When Jibo recognizes an enrolled person and (with their consent) takes a photo, `jibo-api` tags the resulting media record with the recognized person's id and stores it via the `Media_20160725` API. JiboConnector is responsible for the next step: picking up newly captured, person-tagged photos and delivering them to that person's configured notification contacts (text and/or email), as recorded via the `jibo-api` portal's photo-contacts endpoints.

## Stack

**Go**, standard library only (no web framework) — chosen for performance (compiled, static binary, goroutine-based concurrency well suited to a poll-then-fan-out worker), a minimal attack surface (the final Docker image is `distroless/static`, non-root, no shell — this service holds notification-provider credentials and handles household PII), and low ceremony for iterating on a small, focused service.

## Status

End-to-end for both email and SMS.

- `cmd/worker/main.go` — poll loop on a timer, plus a `/health` HTTP endpoint. Each tick lists recently captured person-tagged media from `jibo-api`, looks up that person's notification contacts, and delivers to each over every enabled channel. Tracks an in-memory high-water mark (see "Known gaps" below — this is a deliberately accepted tradeoff, not an oversight).
- `internal/config` — env-based config (`JIBOCONNECTOR_` prefix).
- `internal/jiboapi` — client for `jibo-api`'s `/api/connector/media` and `/api/connector/loop-members/{personId}/photo-contacts` endpoints, authenticated with a shared-secret bearer token.
- `internal/notify` — a `Notifier` interface, fanned out by `CompositeNotifier` so a contact with both an email and a phone number gets both:
  - `SESNotifier` — email via Amazon SES. No-ops for a contact with no email.
  - `SNSNotifier` — SMS via Amazon SNS. No-ops for a contact with no phone number; returns an error (visible in the poll loop's logs) if the number isn't in E.164 format (`+`-prefixed).
  - `NoopNotifier` (logs only) is used automatically if neither channel is configured, so the worker still starts and polls even before notification config is finished.
  - Both AWS notifiers use the SDK's standard credential chain — no bespoke secret handling in this repo.

### Known gaps

- **The "already processed" cursor is in-memory only.** It starts at "now" on every restart, so a restart could in principle miss a photo captured in the same instant. Accepted for now — not planned as a near-term fix.
- **No display name for the recognized person.** `jibo-api`'s connector endpoints only expose a person's id, not their name, so notification messages are generic ("Jibo took a new photo!") rather than personalized.

## Configuration

| Env var | Purpose | Default |
|---|---|---|
| `JIBOCONNECTOR_JIBO_API_BASE_URL` | Base URL of `jibo-api` | `http://api:8080` |
| `JIBOCONNECTOR_JIBO_API_KEY` | Bearer token for `jibo-api`'s `/api/connector/*` endpoints — must match `OpenJibo__Connector__ApiKey` on the `jibo-api` side | *(required — every call 401s without it)* |
| `JIBOCONNECTOR_POLL_INTERVAL_SECONDS` | How often to check for new photos | `30` |
| `JIBOCONNECTOR_HEALTH_ADDR` | Bind address for `/health` | `:8090` |
| `JIBOCONNECTOR_AWS_REGION` | Region for both SES and SNS | `us-east-1` |
| `JIBOCONNECTOR_SES_FROM_ADDRESS` | SES-verified sender address | *(unset — email disabled)* |
| `JIBOCONNECTOR_SMS_ENABLED` | Turn on SMS via SNS | `false` — explicit opt-in, since (unlike SES) there's no required per-service value to gate on, and being able to send email shouldn't silently imply being willing to spend money on SMS |

AWS credentials themselves aren't a JiboConnector setting — the SDK's standard chain (`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`/`AWS_SESSION_TOKEN`, a shared credentials file, or an IAM role) handles that. Both SES and SNS use the same chain.

## Build & run

```
go build -o bin/worker ./cmd/worker
./bin/worker
```

or via Docker:

```
docker build -t jibo-connector .
docker run --rm -p 8090:8090 \
  -e JIBOCONNECTOR_JIBO_API_KEY=... \
  -e JIBOCONNECTOR_SES_FROM_ADDRESS=... \
  -e JIBOCONNECTOR_SMS_ENABLED=true \
  -e AWS_ACCESS_KEY_ID=... -e AWS_SECRET_ACCESS_KEY=... \
  jibo-connector
```

## Relationship to jibo-api

This repository is intentionally separate from [`JiboExperiments`](https://github.com/wrcrooks/JiboExperiments) (the `jibo-api` cloud server). `jibo-api` never sends SMS/email directly — it only tags and stores photos and stores contact preferences. JiboConnector is the piece that actually reaches out to a phone/email provider.
