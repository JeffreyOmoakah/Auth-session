# Auth-Session

A session-based authentication API built in Go. Uses server-side sessions stored in Redis rather than JWTs — meaning sessions can be invalidated instantly, logout is real, and the server owns the truth.

## Tech Stack

| Layer | Tool |
|---|---|
| Language | Go |
| Router | Chi |
| User Store | PostgreSQL |
| Session Store | Redis |
| SQL Generation | sqlc |
| Migrations | Goose |
| Containers | Docker Compose |
| Password Hashing | bcrypt |

## How It Works

```
POST /login
  │
  ├── verify credentials against Postgres
  ├── generate 32-byte crypto/rand session ID
  ├── store session record in Redis with 24h TTL
  └── set HttpOnly cookie with session ID

Every protected request
  │
  ├── middleware extracts session_id cookie
  ├── Redis GET — validates session exists and hasn't expired
  ├── injects session into request context
  └── handler reads user identity from context

DELETE /logout
  │
  ├── Redis DEL — session is dead immediately
  └── cookie cleared on client
```

The session ID in the cookie is a random pointer. All identity data lives in Redis. The UUID in Postgres never leaves the server.

## Project Structure

```
auth-session/
│
├── cmd/
│   ├── api.go          # Chi router, middleware, route mounting
│   └── main.go         # Entry point, wires deps, starts server
│
├── internal/
│   ├── adapters/
│   │   └── postgresql/
│   │       ├── db.go
│   │       ├── migrations/
│   │       └── sqlc/          # sqlc generated code
│   │
│   ├── env/
│   │   └── env.go             # Typed config from environment
│   │
│   ├── json/
│   │   └── json.go            # Read/write helpers
│   │
│   └── users/
│       ├── types.go           # User and Session types
│       ├── service.go         # Auth + session logic
│       ├── handler.go         # HTTP handlers
│       └── middleware.go      # Session validation middleware
│
├── docker-compose.yml
├── sqlc.yaml
├── Makefile
├── .env.example
├── go.mod
└── go.sum
```

## Getting Started

### Prerequisites

- Go 1.22+
- Docker + Docker Compose
- [sqlc](https://sqlc.dev)
- [goose](https://github.com/pressly/goose)

### Setup

**1. Clone the repo**
```bash
git clone https://github.com/JeffreyOmoakah/Auth-session.git
cd Auth-session
```

**2. Copy env file**
```bash
cp .env.example .env
```

**3. Start Postgres and Redis**
```bash
make docker-up
```

**4. Run migrations**
```bash
make migrate-up
```

**5. Start the server**
```bash
make run
```

Server starts on `http://localhost:3000`

## Environment Variables

```env
PORT=3000

# pgxpool connection (URL format)
DATABASE_URL=postgres://postgres:postgres@localhost:5432/auth-sessions?sslmode=disable

# Goose migration (key=value format)
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=host=localhost user=postgres password=postgres dbname=auth-sessions sslmode=disable
GOOSE_MIGRATION_DIR=./internal/adapters/postgresql/migrations

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
```

## API Endpoints

### Auth

| Method | Endpoint | Description | Auth required |
|---|---|---|---|
| `POST` | `/v1/auth-sessions/signup` | Create a new account | No |
| `POST` | `/v1/auth-sessions/login` | Login and receive session cookie | No |
| `DELETE` | `/v1/auth-sessions/logout` | Invalidate session | Yes |

### Protected

| Method | Endpoint | Description | Auth required |
|---|---|---|---|
| `GET` | `/v1/me` | Get current user from session | Yes |

### Signup

```bash
curl -X POST http://localhost:3000/v1/auth-sessions/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "jeff@example.com", "password": "yourpassword"}'
```

### Login

```bash
curl -X POST http://localhost:3000/v1/auth-sessions/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{"email": "jeff@example.com", "password": "yourpassword"}'
```

The session cookie is set automatically. Use `-c cookies.txt` to persist it.

### Protected Route

```bash
curl -X GET http://localhost:3000/v1/me \
  -b cookies.txt
```

### Logout

```bash
curl -X DELETE http://localhost:3000/v1/auth-sessions/logout \
  -b cookies.txt
```

## Session Security

The cookie is hardened with these flags on every response:

```
Set-Cookie: session_id=<64-char hex>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=86400
```

| Flag | What it blocks |
|---|---|
| `HttpOnly` | JavaScript cannot read the cookie — blocks XSS theft |
| `Secure` | Cookie only sent over HTTPS — blocks plain HTTP interception |
| `SameSite=Strict` | Cookie not sent on cross-origin requests — blocks CSRF |

Session IDs are generated with `crypto/rand` — never `math/rand`, never UUIDs.

## Makefile

```bash
make run            # start the server
make docker-up      # spin up Postgres + Redis
make docker-down    # tear down containers
make migrate-up     # run goose migrations
make migrate-down   # rollback last migration
make migrate-status # check migration state
make sqlc           # regenerate sqlc output from queries
```

## Why Sessions over JWT

JWT tokens cannot be invalidated before they expire. If a token is stolen or a user is compromised, the server has no way to revoke it without a blacklist — at which point you've rebuilt sessions badly.

With server-side sessions, calling `DELETE /logout` deletes the record from Redis. The next request with that cookie hits a dead key and gets a 401 immediately. The server owns the truth.

```
JWT:      Client holds truth  →  Server verifies signature  →  Trusts it
Session:  Client holds key    →  Server looks up truth      →  Trusts its own store
```
