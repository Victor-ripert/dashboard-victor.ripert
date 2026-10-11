
# Dashboard

Dashboard is a EPITECH project, that display some informations get by micro-services, with an account system that store preferences and every customise ui/ux modifications from the user.

## Project parts

| Part | Folder | Stack | Default port |
|---|---|---|---|
| Front-end | `frontend/` | React + Vite + Tailwind | 5173 |
| Gateway | `gateway/` | Go + Gin | 8080 |
| Weather service | `service/weather/` | Node.js + Express | 4000 (internal) |
| Database | — | PostgreSQL 16 | 5432 |
| Mail server (development) | — | Mailpit | 8025 (web UI), 1025 (SMTP) |

The front-end talks only to the gateway. The gateway handles authentication (users are stored in PostgreSQL) and forwards widget requests to the services.

In development, the emails the gateway sends go to Mailpit, a fake mail server: it catches every email and shows it in a web page instead of delivering it.

## Getting started

### Prerequisites

- Docker with Docker Compose
- Node.js 20+ and npm (for the front-end)
- Go 1.22+ (only to run the gateway outside Docker)

### 1. Configure

```bash
cp .env.example .env
```

Then edit `.env`:

- `JWT_SECRET` (required): the key used to sign login tokens. Generate one with `openssl rand -hex 32`.
- `GATEWAY_PORT` / `DB_PORT` / `MAILPIT_UI_PORT` / `MAILPIT_SMTP_PORT` (optional): change them if 8080, 5432, 8025 or 1025 are already used on your machine.
- `SMTP_HOST` / `SMTP_PORT` / `SMTP_FROM` / `SMTP_USERNAME` / `SMTP_PASSWORD` (optional): the mail server the gateway sends emails to, the sender address and the credentials. Leave them unset to use the Mailpit container.

### 2. Start the back-end (database, weather service, mail server, gateway)

From the project root:

```bash
docker compose up --build
```

This starts PostgreSQL, the weather service and Mailpit, then the gateway once the database and Mailpit are ready.

- The gateway is available at `http://localhost:8080` (or `GATEWAY_PORT`).
- The emails sent by the gateway can be read at `http://localhost:8025` (or `MAILPIT_UI_PORT`).

Useful commands:

```bash
docker compose up --build -d   # start in the background
docker compose logs -f gateway # follow the logs of one container
docker compose down            # stop everything (data is kept)
docker compose down -v         # stop everything and delete the database
```

### 3. Start the front-end

In a second terminal:

```bash
cd frontend
npm install   # first time only
npm run dev
```

Open the address Vite prints, usually `http://localhost:5173`. You land on the login page; create an account there.

Vite forwards requests starting with `/api` to the gateway at `http://localhost:8080`. If the gateway runs on another port, set `GATEWAY_URL` (copy `frontend/.env.example` to `frontend/.env`, or run `GATEWAY_URL=http://localhost:8099 npm run dev`).

### Running one part without Docker

Useful when working on a single part. Run each command from the project root.

**Weather service:**

```bash
cd service/weather
npm install
npm start   # listens on port 4000
```

**Gateway** (needs the database and the mail server, which can still run in Docker):

```bash
docker compose up -d db mailpit
cd gateway
JWT_SECRET=dev-secret go run .
```

The gateway reads these environment variables:

| Variable | Default | Role |
|---|---|---|
| `PORT` | `8080` | Port the gateway listens on |
| `JWT_SECRET` | — (required) | Key used to sign login tokens |
| `DATABASE_URL` | `postgres://dashboard:dashboard@localhost:5432/dashboard?sslmode=disable` | PostgreSQL connection. Change the port if you changed `DB_PORT`. |
| `WEATHER_URL` | `http://localhost:4000` | Address of the weather service |
| `SMTP_HOST` / `SMTP_PORT` | `localhost` / `1025` | SMTP server used to send emails (Mailpit's published port by default). Change the port if you changed `MAILPIT_SMTP_PORT`. |
| `SMTP_FROM` | `Dashboard <noreply@dashboard.local>` | Sender of the emails |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | — (optional) | Credentials for a real provider. They are only sent over an encrypted connection (STARTTLS) or to `localhost`. |

**Gateway tests** (no Docker needed):

```bash
cd gateway
go test ./auth/ ./mailer/ ./internal/github/ .
```

## Gateway routes

| Route | Login required | Description |
|---|---|---|
| `GET /about.json` | No | Lists the services and their widgets |
| `POST /api/auth/register` | No | Creates an unverified account (`{"email", "password"}`) and emails it a 6-digit code. Returns `201 {"email", "verification_required": true}`, no token. `409` if the email already has an account. |
| `POST /api/auth/verify` | No | Checks the code (`{"email", "code"}`), verifies the account and returns a token. `400` for a wrong or expired code. |
| `POST /api/auth/resend-code` | No | Emails a new code (`{"email"}`). `204`, or `429` with `retry_after` (seconds) if asked again within 60 seconds. |
| `POST /api/auth/login` | No | Returns a token (`{"email", "password"}`). `403 "Email not verified"` if the account isn't verified yet. |
| `GET /api/auth/me` | Yes | Returns the logged-in user |
| `GET /api/weather/widgets/current_weather?city=Paris` | Yes | Current weather for a city |
| `GET /api/github/activity?repo=owner/repository` | Yes | Recent commits and today/current-week statistics for a public GitHub repository |

For routes that require login, send the token in the header: `Authorization: Bearer <token>`.

### GitHub activity service

The GitHub widget reads activity from a public repository. The repository must use the following format:

```text
owner/repository
```

The endpoint requires an authenticated Dashboard user:

```text
GET /api/github/activity?repo=owner/repository
```

Example request:

```bash
curl \
  -H "Authorization: Bearer YOUR_DASHBOARD_TOKEN" \
  "http://localhost:8080/api/github/activity?repo=owner/repository"
```

Example response:

```json
{
  "repository": "owner/repository",
  "commits": [
    {
      "sha": "abc123",
      "message": "Fix authentication redirect",
      "author": "octocat",
      "avatar_url": "https://avatars.githubusercontent.com/u/1",
      "date": "2026-10-08T09:30:00Z",
      "url": "https://github.com/owner/repository/commit/abc123"
    }
  ],
  "counts": {
    "today": 4,
    "this_week": 18
  }
}
```

`today` counts commits from midnight UTC until the current time. `this_week` counts commits from Monday at `00:00 UTC` until the current time. The response includes up to 10 recent commits, while statistics can use up to 10 pages of GitHub results.

Successful activity responses are cached per repository for 60 seconds. Concurrent requests for the same repository share one GitHub request. Failed requests are not cached.

The endpoint returns these errors:

| Status | Meaning |
|---|---|
| `400` | The repository is missing or does not use the `owner/repository` format |
| `401` | The Dashboard authentication token is missing or invalid |
| `404` | The GitHub repository does not exist or is not public |
| `429` | GitHub rate limit exceeded; `retry_after` may contain the number of seconds to wait |
| `502` | GitHub could not be reached or returned an unexpected response |

Only public repositories are supported currently. Private repositories require a future GitHub OAuth connection and are not accessed with the Dashboard account credentials.

To test the service locally:

1. Start the backend with `docker compose up --build`.
2. Create and verify a Dashboard account, then log in to obtain a Dashboard JWT.
3. Replace `YOUR_DASHBOARD_TOKEN` in the example request with that JWT.
4. Replace `owner/repository` with a public repository such as `octocat/Hello-World`.
5. Check the response for recent commits and the `counts` object.

The GitHub client and cache tests can be run without Docker:

```bash
cd gateway
go test ./internal/github/ -race
```

### Email verification rules

- A new account must be verified with the code emailed at registration before it can log in.
- A code is valid for 10 minutes and allows 5 guesses. Asking for a new code replaces the previous one.
- An account that can't be verified anymore is deleted, which frees its email for a new registration:
  - its code expired, or it received 5 wrong codes;
  - the verification email could not be sent;
  - it was created more than 30 minutes ago, however many new codes were requested.
- Accounts created before email verification existed are considered verified.

## Current status

- **Connected:** registration with email verification, login and logout, the weather widget, and the GitHub activity widget for public repositories.
- **Still demo data:** the trading widget. Its routes (`/api/trading/...`) don't exist in the gateway yet, so placing an order fails.
- **Not yet supported:** GitHub OAuth and private repository access.
