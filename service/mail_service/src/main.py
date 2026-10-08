import secrets
from contextlib import asynccontextmanager
from datetime import datetime, timedelta, timezone
from fastapi import Depends, FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from googleapiclient.discovery import build
from sqlalchemy.orm import Session
from src.auth import current_user_id
from src.config import settings
from src.db import Base, GmailConnection, OAuthState, engine, get_db
from src.gmail import encrypt_credentials, gmail_for_user, google_flow


@asynccontextmanager
async def lifespan(app: FastAPI):
    Base.metadata.create_all(engine)
    yield

app = FastAPI(title="Mail service", lifespan=lifespan)

app.add_middleware(
CORSMiddleware,
allow_origins=settings.allowed_origins,
allow_credentials=True,
allow_methods=["*"],
allow_headers=["*"]
)


@app.post("/gmail/connect")
def connect_gmail(user_id: str = Depends(current_user_id), db: Session = Depends(get_db)):
    state = secrets.token_urlsafe(32)

    db.add(OAuthState(state=state, user_id=user_id, expires_at=(datetime.now(timezone.utc) + timedelta(minutes=10))))
    db.commit()
    flow = google_flow(state)
    authorization_url, _ = flow.authorization_url(access_type="offline",prompt="consent")
    return {"authorization_url": authorization_url}


@app.get("/oauth/callback")
def google_callback(code: str,state: str,db: Session = Depends(get_db)):
    oauth_state = db.get(OAuthState, state)

    if oauth_state is None:
        raise HTTPException(status_code=400, detail="Invalid OAuth state")

    if oauth_state.expires_at < datetime.now(timezone.utc):
        db.delete(oauth_state)
        db.commit()
        raise HTTPException(status_code=400, detail="Expired OAuth state")

    user_id = oauth_state.user_id
    db.delete(oauth_state)
    db.commit()
    flow = google_flow(state)
    flow.fetch_token(code=code)
    credentials = flow.credentials
    google = build("gmail", "v1", credentials=credentials, cache_discovery=False)
    profile = google.users().getProfile(userId="me").execute()
    connection = db.get(GmailConnection, user_id)

    if connection is None:
        connection = GmailConnection(
            user_id=user_id,
            gmail_email=profile["emailAddress"],
            credentials=encrypt_credentials(credentials),
        )
        db.add(connection)
    else:
        connection.gmail_email = profile["emailAddress"]
        connection.credentials = encrypt_credentials(credentials)
    db.commit()
    return {"connected": True,"email": connection.gmail_email}


@app.get("/gmail/status")
def gmail_status(user_id: str = Depends(current_user_id),db: Session = Depends(get_db)):
    connection = db.get(GmailConnection,user_id)

    if connection is None:
        return {"connected": False}
    return {"connected": True, "email": connection.gmail_email}


@app.delete("/gmail/connection")
def disconnect_gmail(user_id: str = Depends(current_user_id), db: Session = Depends(get_db)):
    connection = db.get(GmailConnection, user_id)

    if connection is not None:
        db.delete(connection)
        db.commit()

    return {"disconnected": True}


@app.get("/emails")
def list_emails(
    limit: int = Query(default=20, ge=1, le=50),
    search: str | None = None,
    user_id: str = Depends(current_user_id),
    db: Session = Depends(get_db)
):
    gmail = gmail_for_user(db, user_id)

    result = gmail.users().messages().list(
        userId="me",
        maxResults=limit,
        q=search
    ).execute()

    emails = []

    for item in result.get("messages", []):
        message = gmail.users().messages().get(
            userId="me",
            id=item["id"],
            format="metadata",
            metadataHeaders=[
                "From",
                "To",
                "Subject",
                "Date"
            ]
        ).execute()

        headers = {
            header["name"].lower(): header["value"]
            for header in message["payload"].get(
                "headers",[])
        }

        emails.append({
            "id": message["id"],
            "thread_id": message["threadId"],
            "from": headers.get("from"),
            "to": headers.get("to"),
            "subject": headers.get(
                "subject",
                "(Sans objet)",
            ),
            "date": headers.get("date"),
            "snippet": message.get("snippet", "")
        })

    return {
        "emails": emails,
        "next_page_token": result.get(
            "nextPageToken"
        )
    }
