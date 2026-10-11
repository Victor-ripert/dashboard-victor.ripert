# Mail Service

Microservice FastAPI permettant à chaque utilisateur de connecter son compte Gmail avec OAuth 2.0 et de récupérer ses emails. Les credentials Google sont associés à l'utilisateur de l'application et chiffrés avant leur stockage en base.

## Lancer le service

Créer un fichier `.env` à la racine :

```env
DATABASE_URL=postgresql+psycopg://mail_user:mail_password@db:5432/mail_db

GMAIL_CLIENT_ID=ton-client-id
GMAIL_CLIENT_SECRET=ton-client-secret
GMAIL_REDIRECT_URI=http://localhost:8000/oauth/google/callback

TOKEN_ENCRYPTION_KEY=ta-cle-fernet

JWT_SECRET=ton-secret-jwt
JWT_ISSUER=auth-service
JWT_AUDIENCE=mail-service

CORS_ORIGINS=http://localhost:3000
```

Lancer l'API et PostgreSQL :

```bash
docker compose up --build
```

L'API est disponible sur `http://localhost:8000` et la documentation sur `http://localhost:8000/docs`.

Pour arrêter les conteneurs :

```bash
docker compose down
```

## Routes

Toutes les routes, sauf le callback Google, utilisent un JWT dans le header :

```http
Authorization: Bearer <token>
```

| Méthode | Route | Description |
|---|---|---|
| `POST` | `/gmail/connect` | Retourne l'URL permettant de connecter Gmail |
| `GET` | `/oauth/google/callback` | Reçoit le retour OAuth de Google |
| `GET` | `/gmail/status` | Indique si Gmail est connecté |
| `DELETE` | `/gmail/connection` | Supprime la connexion Gmail |
| `GET` | `/emails` | Retourne les emails de l'utilisateur connecté |

## Base de données

PostgreSQL contient deux tables :

- `gmail_connections` : adresse Gmail et credentials Google chiffrés pour chaque utilisateur ;
- `oauth_states` : états OAuth temporaires utilisés pendant la connexion Google.

Les données PostgreSQL sont conservées dans le volume Docker `postgres_data`. La commande `docker compose down -v` supprime ce volume et toutes ses données.