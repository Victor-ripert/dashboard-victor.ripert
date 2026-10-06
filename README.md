
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
- `SMTP_HOST` / `SMTP_PORT` (optional): the mail server the gateway sends emails to. Leave them unset to use the Mailpit container.

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

## Gateway routes

| Route | Login required | Description |
|---|---|---|
| `GET /about.json` | No | Lists the services and their widgets |
| `POST /api/auth/register` | No | Creates an account (`{"email", "password"}`) and returns a token |
| `POST /api/auth/login` | No | Returns a token (`{"email", "password"}`) |
| `GET /api/auth/me` | Yes | Returns the logged-in user |
| `GET /api/weather/widgets/current_weather?city=Paris` | Yes | Current weather for a city |

For routes that require login, send the token in the header: `Authorization: Bearer <token>`.

## Current status

- **Connected:** register, login and logout, and the weather widget (real data through the gateway).
- **Still demo data:** the trading widget. Its routes (`/api/trading/...`) don't exist in the gateway yet, so placing an order fails.
