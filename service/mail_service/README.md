# Mail Service

A FastAPI microservice that allows each user to connect their Gmail account with OAuth 2.0 and retrieve their emails. Google credentials are linked to the application user and encrypted before being stored in the database.

## Run the service

Create a `.env` file at the project root:

```env
DATABASE_URL=postgresql+psycopg://mail_user:mail_password@db:5432/mail_db

GMAIL_CLIENT_ID=your-client-id
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REDIRECT_URI=http://localhost:8000/oauth/google/callback

TOKEN_ENCRYPTION_KEY=your-fernet-key

JWT_SECRET=your-jwt-secret
JWT_ISSUER=auth-service
JWT_AUDIENCE=mail-service

CORS_ORIGINS=http://localhost:3000
```

Start the API and PostgreSQL:

```bash
docker compose up --build
```

The API is available at `http://localhost:8000` and the documentation at `http://localhost:8000/docs`.

Stop the containers:

```bash
docker compose down
```

## Routes

All routes except the Google callback require a JWT in the request header:

```http
Authorization: Bearer <token>
```

| Method | Route | Description |
|---|---|---|
| `POST` | `/gmail/connect` | Returns the URL used to connect a Gmail account |
| `GET` | `/oauth/google/callback` | Handles the Google OAuth callback |
| `GET` | `/gmail/status` | Checks whether Gmail is connected |
| `DELETE` | `/gmail/connection` | Deletes the Gmail connection |
| `GET` | `/emails` | Returns the authenticated user's emails |

## Database

PostgreSQL contains two tables:

- `gmail_connections`: stores each user's Gmail address and encrypted Google credentials;
- `oauth_states`: stores temporary OAuth states used during the Google connection flow.

PostgreSQL data is stored in the `postgres_data` Docker volume. Running `docker compose down -v` deletes this volume and all its data.