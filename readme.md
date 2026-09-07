# OpenLobby

A lightweight, high-concurrency **virtual waiting room** service built with [Go](https://go.dev) and [Fiber v3](https://docs.gofiber.io). It protects backend services during traffic spikes (product drops, ticket sales, flash sales) by placing users into a FIFO queue and handing out short-lived access tokens once a slot frees up.

## Features

- **FIFO waiting queue** — users join in order and are admitted in order.
- **Passthrough mode** — when fewer than 50 users are active and the queue is empty, new users are admitted immediately without waiting.
- **Time-limited access tokens** — admitted users receive a cryptographically random 32-character token that expires after 15 minutes.
- **Session-based tracking** — users are identified through cookie-backed sessions (Fiber session middleware), no client-side state required.
- **Automatic session cleanup** — a background goroutine runs every 30 seconds to evict expired active sessions and admit the next users in line.
- **Built-in monitoring dashboard** — `/metrics` powered by `gofiber/contrib/monitor` for real-time metrics.
- **CORS enabled** — pre-configured for a companion frontend running on `http://localhost:5173` (e.g. a Vite dev server).
- **Load-testing ready** — ships with a k6 script to simulate heavy traffic.

## How it works

1. A client calls `GET /request`.
2. If the client has no session yet, a `userId` is generated (UUID) and stored in a cookie-backed session.
3. The server checks for a **passthrough**: if fewer than 50 users are active **and** the queue is empty, the user is admitted instantly and receives a token.
4. Otherwise the user is added to the end of the FIFO queue.
5. Returning clients poll `GET /request` — once admitted, the server returns their token.
6. The client validates its token with `GET /validate-token` before accessing protected resources and invalidates it with `POST /invalidate-token` when done.

The queue is guarded by a `sync.Mutex` and keeps two maps: `users` (waiting) and `activeUser` (admitted with a live token), plus an ordered slice for FIFO semantics.

## Project structure

```
OpenLobby/
├── main.go               # Fiber app, session/CORS setup, HTTP routes
├── queue/
│   └── queue.go          # WaitingRoom: FIFO queue, admission, expiry logic
├── utils/
│   ├── env.go            # Config: .env loader + env-var reader (caarlos0/env)
│   └── hash.go           # Crypto-safe random token generation
├── views/
│   └── index.html        # Minimal waiting-room placeholder page
├── load-testing/
│   ├── index.js          # k6 load test (1000 VUs for 60s)
│   └── package.json
├── go.mod
└── readme.md
```

## API

| Method | Path                 | Params           | Description                                      |
| ------ | -------------------- | ---------------- | ------------------------------------------------ |
| GET    | `/request`           | —                | Join the queue or receive a token (passthrough/admitted) |
| GET    | `/validate-token`    | `?token=`        | Check whether a token is valid for this session  |
| POST   | `/invalidate-token`  | `?token=`        | Invalidate a token (e.g. after access completes) |
| GET    | `/metrics`           | —                | Live monitoring dashboard                         |
| GET    | `/`                  | —                | Serves the placeholder waiting-room page          |

> **Note:** the token endpoints are session-scoped — the session cookie must match the user that was issued the token.

## Getting started

### Prerequisites

- Go 1.26+

### Run the server

```bash
go run main.go
```

The server listens on `:3000`.

### Build

```bash
go build -o openlobby .
```

## Configuration

OpenLobby reads its configuration from environment variables through `LoadConfig()` in [`utils/env.go`](utils/env.go), which uses [`caarlos0/env`](https://github.com/caarlos0/env) under the hood. Outside of `PRODUCTION`, a `.env` file in the project root is loaded automatically (via `godotenv`) so you can override settings locally.

| Variable                  | Type               | Default                                        | Description                                                   |
| ------------------------- | ------------------ | ---------------------------------------------- | ------------------------------------------------------------- |
| `ALLOWED_ORIGINS`         | comma-separated    | `http://localhost:5173,http://localhost:3000`  | Origins allowed by CORS                                       |
| `NUMBER_OF_ALLOWED_USERS` | int                | `50`                                           | Passthrough threshold — max active users admitted while the queue is empty |
| `REMOVE_EXPIRED_SESSION`  | duration           | `30s`                                          | Interval between runs of the expired-session cleanup (e.g. `30s`, `1m30s`) |
| `PORT`                    | string             | `3000`                                         | Port the HTTP server listens on                               |
| `APP_ENV`                 | string             | _(unset)_                                      | When set to `PRODUCTION`, the `.env` file is **not** loaded    |

Example `.env` file:

```env
APP_ENV=DEVELOPMENT
PORT=8080
ALLOWED_ORIGINS=http://localhost:5173,https://app.example.com
NUMBER_OF_ALLOWED_USERS=100
REMOVE_EXPIRED_SESSION=1m
```

> **Note:** `LoadConfig()` is ready to use, but the server entrypoint (`main.go`) does not consume it yet — it still runs with the hardcoded values shown above.

## Load testing

The `load-testing` folder contains a [k6](https://k6.io) script that simulates 1000 concurrent users hitting `/request` for 60 seconds.

```bash
k6 run load-testing/index.js
```

> The script currently targets a deployed instance at `http://103.196.155.119:3000` — update the URL in `load-testing/index.js` to point at your local server (`http://localhost:3000`).

## Configuring the frontend

CORS currently allows requests from `http://localhost:5173` (see `cors.New(...)` in `main.go`). Once `LoadConfig()` is wired into `main.go`, the allowed origins will instead be driven by the `ALLOWED_ORIGINS` environment variable (default: `http://localhost:5173,http://localhost:3000`) — see [Configuration](#configuration).
