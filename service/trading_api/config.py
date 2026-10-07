import os
from pathlib import Path
from dotenv import load_dotenv

load_dotenv(Path(__file__).resolve().parent / ".env")


class Settings:
    IB_HOST: str = os.getenv("IB_HOST", "127.0.0.1")
    IB_PORT: int = int(os.getenv("IB_PORT", "4002"))
    IB_CLIENT_ID: int = int(os.getenv("IB_CLIENT_ID", "1"))
    IB_TIMEOUT: float = float(os.getenv("IB_TIMEOUT", "10.0"))
    MOCK_MODE: bool = os.getenv("MOCK_MODE", "false").lower() in ("true", "1", "yes")
    AUTO_FALLBACK_MOCK: bool = os.getenv("AUTO_FALLBACK_MOCK", "true").lower() in ("true", "1", "yes")

    API_HOST: str = os.getenv("API_HOST", "0.0.0.0")
    API_PORT: int = int(os.getenv("API_PORT", "8000"))
    CORS_ORIGINS: list[str] = [o.strip() for o in os.getenv("CORS_ORIGINS", "*").split(",") if o.strip()]


settings = Settings()
