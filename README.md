# Ticket System (Golang Backend Intern Assignment)

A small REST backend where a user can register, log in, create tickets, view
only their own tickets, and move a ticket through its status lifecycle.

## Tech choices

- **Language:** Go 1.22, standard library only (`net/http`'s Go 1.22 method
  + path-pattern routing is used instead of a third-party router).
- **Storage:** in-memory, guarded by a mutex. Allowed by the assignment scope
  and keeps the service dependency-free, easy to build, and trivial to
  deploy anywhere. Data resets on restart (see Assumptions below).
- **Auth:** JWT (HS256), implemented directly with `crypto/hmac` — no
  external JWT library.
- **Passwords:** never stored in plain text. Stored as a per-user random
  salt + an iterated (100,000 round) salted HMAC-SHA256 hash, implemented
  with only the standard library (`crypto/hmac`, `crypto/sha256`,
  `crypto/rand`). This is a PBKDF2-style construction; functionally
  equivalent in purpose to bcrypt, chosen here to keep the module
  dependency-free.

Using zero third-party dependencies was a deliberate choice for this
assignment: it keeps `go build` fully reproducible offline and avoids any
supply-chain/module-proxy surprises during grading or deployment.

## Project layout

```
.
├── main.go                       # server wiring, routes, startup
├── internal/
│   ├── auth/                     # JWT + password hashing
│   ├── handlers/                 # HTTP handlers (business logic)
│   ├── middleware/                # JWT auth middleware
│   ├── models/                    # User, Ticket, status transition rules
│   └── store/                     # in-memory data store
├── Dockerfile
├── .env.example
└── README.md
```

## API

All responses are JSON. All protected endpoints require
`Authorization: Bearer <token>`.

| Method | Endpoint               | Auth | Purpose                     |
|--------|-------------------------|------|------------------------------|
| GET    | `/health`               | no   | Health check                 |
| POST   | `/auth/register`        | no   | Register user                |
| POST   | `/auth/login`           | no   | Login, returns JWT           |
| POST   | `/tickets`              | yes  | Create ticket                |
| GET    | `/tickets`              | yes  | List logged-in user's tickets|
| GET    | `/tickets/{id}`         | yes  | Get own ticket by ID         |
| PATCH  | `/tickets/{id}/status`  | yes  | Update own ticket status     |

**Register / Login body:**
```json
{ "email": "alice@example.com", "password": "at-least-8-chars" }
```
Response: `{ "token": "<jwt>", "user": { "id": "...", "email": "..." } }`

**Create ticket body:**
```json
{ "title": "Printer broken", "description": "Won't turn on" }
```

**Update status body:**
```json
{ "status": "in_progress" }
```

Status flow: `open -> in_progress -> closed`. A closed ticket can never move
again (attempting to do so returns `409 Conflict`). Setting a ticket to its
current status also returns `409`. A ticket belonging to another user, or
one that doesn't exist, returns `404` in both cases (so ownership is never
leaked via a 403).

## Local run (without Docker)

Requires Go 1.22+.

```bash
go build -o ticket-system .
JWT_SECRET=some-long-random-secret PORT=8080 ./ticket-system
curl http://localhost:8080/health
```

## Docker run (per assignment's local run contract)

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=some-long-random-secret ticket-system
curl http://localhost:8080/health
```

Expected health response:
```json
{ "status": "ok" }
```

## Deployment
- **Deployed URL:** https://ticket-system-2u1l.onrender.com
   - **Public health check URL:** https://ticket-system-2u1l.onrender.com/health
   - **GitHub repository:** https://github.com/nipun8890/ticket-system

This service is stateless and only needs one environment variable
(`JWT_SECRET`), so it deploys as-is to any free-tier platform that builds
from a Dockerfile (e.g. Render, Railway, Fly.io). Steps for Render, as an
example:
1. Push this repo to GitHub.
2. Create a new "Web Service" on Render, point it at the repo, and let it
   detect the `Dockerfile`.
3. Set the `JWT_SECRET` environment variable in the dashboard.
4. Render sets `PORT` automatically; the app already reads it from the
   environment, so no code changes are needed.

## Assumptions

- In-memory storage is used per the assignment's explicit allowance; all
  data is lost on process restart. No persistence layer was required by
  the scope, so none was added, to keep the implementation simple.
- Email is treated case-insensitively and normalized to lowercase.
- Passwords must be at least 8 characters.
- Setting a ticket to the status it already has is rejected with `409`
  rather than treated as a no-op, since it isn't a valid forward
  transition in `open -> in_progress -> closed`.
- Attempting to access another user's ticket returns `404 Not Found`
  (rather than `403 Forbidden`) so that the existence of another user's
  ticket ID is never revealed to a non-owner.
- JWTs expire after 24 hours; there is no refresh-token flow, as it is out
  of scope.
- No admin role, ticket assignment, or comments — all explicitly out of
  scope per the assignment brief.
