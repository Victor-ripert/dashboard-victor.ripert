from cryptography.fernet import Fernet
from service.trading_service.app.config import settings

encrypted = Fernet(settings.token_encryption_key.encode())

def encrypt_token(token: str) -> str:
    return encrypted.encrypt(token.encode()).decode()

def decrypt_token(encrypted_token: str) -> str:
    return encrypted.decrypt(encrypted_token.encode()).decode()
