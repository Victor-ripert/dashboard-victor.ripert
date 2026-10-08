import json
from cryptography.fernet import Fernet, InvalidToken
from fastapi import HTTPException
from google.oauth2.credentials import Credentials
from src.config import settings

fernet = Fernet(settings.token_encryption_key.encode("utf-8"))

def encrypt_credentials(credentials: Credentials) -> str:
    serialized_credentials = credentials.to_json().encode("utf-8")
    return fernet.encrypt(serialized_credentials).decode("utf-8")

def decrypt_credentials(
    encrypted_credentials: str,
) -> Credentials:
    try:
        serialized_credentials = fernet.decrypt(encrypted_credentials.encode("utf-8")).decode("utf-8")
    except InvalidToken as error:
        raise HTTPException(
            status_code=500,
            detail="Impossible de déchiffrer les credentials Gmail.",
        ) from error

    credentials_data = json.loads(serialized_credentials)

    return Credentials.from_authorized_user_info(credentials_data, scopes=credentials_data.get("scopes"))
