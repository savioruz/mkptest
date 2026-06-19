# MKP Backend Engineer Test — Online Cinema Ticketing Platform

**Kandidat / Candidate:** `<ISI NAMA LENGKAP — FILL IN YOUR FULL NAME>`

A national-scale online cinema ticketing backend in **Go**, implementing the full
system topology in [`docs/topology.md`](docs/topology.md): catalog management,
concurrency-safe seat selection (Redis holds + Postgres partial unique index),
bookings, a mock payment gateway with webhook settlement, Asynq background jobs,
a Kafka event bus with consumers, and the refund / cancellation flow.

> Test deliverables: **A.** System design → [`docs/topology.md`](docs/topology.md)
> · **B.** PostgreSQL schema → [`docs/db/cinema_schema.sql`](docs/db/cinema_schema.sql)
> · **C.** API (Login + CRUD jadwal tayang with authorization) → this service.
> Postman collection → [`docs/postman`](docs/postman).

---

## Architecture at a glance

| Concern | Technology | Role |
|---|---|---|
| API | Go (chi), modular monolith | Auth, catalog, booking, payment, refund |
| Source of truth | PostgreSQL (sqlx, read/write split) | Transactional data + integrity constraints |
| Cache & locks | Redis | Atomic seat holds (Lua), seat-map reads |
| Task queue | Asynq (on Redis) | Deferred/retryable jobs: release expired holds, e-tickets |
| Event bus | Kafka | Durable facts: `ticket.sold`, `schedule.cancelled`, `refund.requested`, `seat.restocked` |
| Payments | Internal **mock** gateway + webhook | Run end-to-end with no external account |
| Observability | OpenTelemetry | Tracing scopes per layer |

**Anti–double-booking (two layers):**
1. **Redis hold** — `SET seat:{schedule}:{seat} NX EX` via an all-or-nothing Lua script (fast, UX).
2. **Postgres** — `CREATE UNIQUE INDEX … ON booking_seats (schedule_id, seat_id) WHERE status='booked'` (correct, final). The confirm step flips `held → booked` and the index rejects any conflict.

Schedules can't overlap per studio: a GiST exclusion constraint
(`EXCLUDE USING gist (studio_id WITH =, tstzrange(start_time, end_time) WITH &&)`).

---

## Requirements

- Go 1.24+ (developed on 1.26), `make`, Docker + Docker Compose
- Ports used by compose: Postgres `5432`, Redis `6379`, Kafka `29092` (host), Kafka UI `8090`, app `8080`

## Quick start

```bash
cp .env.example .env          # defaults work as-is for docker-compose

# 1) Infrastructure
docker-compose up -d postgres redis kafka kafka-ui

# 2) Schema — either run migrations…
make migrate.up
#    …or import the standalone Part-B script (also seeds demo data):
#    psql "postgres://postgres:password@localhost:5432/oil" -f docs/db/cinema_schema.sql

# 3) API + workers + consumers (single process)
make run        # or: make dev   (hot reload)
```

> Running the Go binary **on the host** (outside Docker)? Point `.env` hosts at
> `localhost` and Kafka at `localhost:29092` (the published EXTERNAL listener).
> Running everything in Docker uses the in-network hosts (`postgres`, `redis`, `kafka:9092`).

Swagger UI: <http://localhost:8080/swagger/index.html> · Kafka UI: <http://localhost:8090>

### Seeded logins (password `admin123` for both)

| Email | Role | Can |
|---|---|---|
| `admin@cinema.test` | `admin` | Manage catalog & schedules, cancel schedules |
| `customer@cinema.test` | `user` | Browse, book, self-refund |

---

## API overview

All routes are under `/api` and (except the public ones) require `Authorization: Bearer <token>`.

| Method & path | Auth | Description |
|---|---|---|
| `POST /api/auth/login` | public | **Login** (test C.1) → access + refresh token |
| `POST /api/auth/register` | public | Register a user |
| `GET/POST /api/movies`, `…/{id}` PATCH/DELETE | read: any · write: **admin** | Movies CRUD |
| `GET/POST /api/cinemas`, `…/{id}` | read: any · write: **admin** | Cinemas CRUD |
| `GET/POST /api/studios`, `…/{id}` | read: any · write: **admin** | Studios CRUD (auto-generates seats) |
| `GET/POST /api/schedules`, `…/{id}` PATCH/DELETE | read: any · write: **admin** | **Jadwal tayang CRUD** (test C.2) |
| `POST /api/schedules/{id}/cancel` | **admin** | Cancel schedule → mass refund |
| `GET /api/schedules/{id}/seat-map` | any | Live seat availability |
| `POST /api/bookings` | user | Hold seats + create pending booking |
| `GET /api/bookings`, `…/{id}` | user | My bookings |
| `POST /api/bookings/{id}/refund` | user (owner) | Customer self-refund |
| `POST /api/payments/webhook` | secret header | (Mock) gateway settlement callback |

---

## End-to-end runbook (mock gateway — no external account)

With the stack running and logged in as the seeded users:

1. **Login** `POST /api/auth/login` → copy `data.access_token`.
2. **Build catalog** (admin): create a movie → cinema → studio (seats auto-generate) → schedule.
3. **Seat map** `GET /api/schedules/{id}/seat-map` → seats show `available`.
4. **Hold + book** `POST /api/bookings` `{schedule_id, seat_ids:[…]}` → booking `pending` + `payment_reference`. Seat map now shows those seats `held`. Re-booking a held seat → `409`.
5. **Settle** `POST /api/payments/webhook` with header `X-Webhook-Secret: <PAYMENT_WEBHOOK_SECRET>` and body `{ "reference": "<payment_reference>", "status": "success" }` → booking `confirmed`; `ticket.sold` appears in Kafka UI; seat map shows `booked`. Repeat the webhook → idempotent `200`.
6. **Auto-expiry**: if a booking is not paid within `PAYMENT_EXPIRE_MINUTES`, the Asynq worker releases the hold and marks it `expired`.
7. **Cancel** `POST /api/schedules/{id}/cancel` (admin) → `schedule.cancelled` → the refund consumer creates a refund per confirmed booking → the refund processor settles each → bookings become `refunded`, seats `released`. Self-refund: `POST /api/bookings/{id}/refund`.

A `status:"failed"` webhook (step 5) instead releases the booking and frees the seats.

---

## Make targets

```bash
make run            # generate (swagger+wire) then run the app
make dev            # hot-reload via air
make build          # compile binary
make test           # run unit tests
make generate       # swagger docs + wire DI
make generate.mock  # regenerate gomock mocks
make migrate.up     # apply migrations
make migrate.down   # roll back one
make migrate.create name=<x>   # new migration pair
```

## Tests

```bash
make test    # gomock unit tests (auth, schedule, payment, …) + shared utilities
```

The concurrency-critical guarantees (partial unique index, exclusion constraint,
Redis Lua hold) are exercised against live Postgres + Redis in the runbook above.

## Project layout

```
cmd/app            – entrypoint (HTTP + Asynq workers + Kafka consumers)
config             – env config
di                 – google/wire dependency injection
events             – Kafka consumers (refund, notification)
infras             – postgres, redis, kafka, asynq, payment gateway, jwt, otel, s3
internal/domains   – movie, cinema, studio, seat, schedule, booking, payment, refund, auth, user
internal/handlers  – HTTP handlers per domain
internal/workers   – Asynq task handlers
migrations/postgres– golang-migrate SQL
shared             – generic repository, dto/filters, seatlock, response, failure, validator
docs               – topology.md (A), db/cinema_schema.sql (B), postman/ (collection)
```
