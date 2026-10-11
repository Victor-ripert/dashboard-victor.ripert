import hashlib
from google_auth_oauthlib.flow import Flow
from src.config import settings

GMAIL_SCOPES = [
    "https://www.googleapis.com/auth/gmail.readonly"
]

def hash_oauth_state(state: str) -> str:
    return hashlib.sha256(state.encode("utf-8")).hexdigest()

def create_google_flow(state: str | None = None) -> Flow:
    flow = Flow.from_client_config( settings.google_client_config, scopes=GMAIL_SCOPES, state=state)
    flow.redirect_uri = settings.gmail_redirect_uri
    return flow