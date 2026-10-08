import base64
from typing import Any
from bs4 import BeautifulSoup
from fastapi import HTTPException
from google.auth.exceptions import RefreshError
from google.auth.transport.requests import Request as GoogleRequest
from googleapiclient.discovery import build
from sqlalchemy import select
from sqlalchemy.orm import Session
from src.encryption import decrypt_credentials, encrypt_credentials
from src.models import GmailConnection

def get_connection(db: Session, user_id: str) -> GmailConnection:
    connection = db.scalar(select(GmailConnection).where(GmailConnection.user_id == user_id))
    
    if connection is None:
        raise HTTPException(status_code=409, detail="No account logged.")
    else:
        return connection

def get_gmail_service(db: Session,user_id: str):
    connection = get_connection(db, user_id)
    credentials = decrypt_credentials(connection.encrypted_credentials)

    if credentials.expired and credentials.refresh_token:
        try:
            credentials.refresh(GoogleRequest())
        except RefreshError:
            raise HTTPException(status_code=401, detail=("Connection expired"))
        connection.encrypted_credentials = encrypt_credentials(credentials)
        db.commit()

    elif not credentials.valid:
        raise HTTPException(status_code=401,detail=("Invalid Gmail credentials."))
    return build("gmail", "v1", credentials=credentials, cache_discovery=False)

def get_header(headers: list[dict[str, str]], name: str) -> str | None:
    expected_name = name.lower()

    for header in headers:
        if header.get("name", "").lower() == expected_name:
            return header.get("value")
    return None


def decode_body(data: str) -> str:
    padding = "=" * (-len(data) % 4)
    decoded = base64.urlsafe_b64decode(data + padding)

    return decoded.decode("utf-8", errors="replace")

def visit_part(part: dict[str, Any], plain_parts: list[str], html_parts: list[str]):
    mime_type = part.get("mimeType", "")
    body_data = part.get("body", {}).get("data")

    if body_data:
        decoded = decode_body(body_data)
        if mime_type == "text/plain":
            plain_parts.append(decoded)
        elif mime_type == "text/html":
            html_parts.append(decoded)
    for child in part.get("parts", []):
        visit_part(child)
    return plain_parts, html_parts


def extract_email_body(payload: dict[str, Any]) -> str:
    plain_parts: list[str] = []
    html_parts: list[str] = []

    plain_parts, html_parts = visit_part(payload, plain_parts, html_parts)
    if plain_parts:
        return "\n".join(plain_parts).strip()
    if html_parts:
        html = "\n".join(html_parts)
        return BeautifulSoup(html, "lxml").get_text("\n", strip=True)
    return ""
