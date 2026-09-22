import os
from pathlib import Path
from dotenv import load_dotenv

env_path = Path(__file__).resolve().parent / ".env"
load_dotenv(dotenv_path=env_path)


class Settings:
    # Interactive Brokers Connection
    IB_HOST: str = os.getenv("IB_HOST", "127.0.0.1")
    IB_PORT: int = int(os.getenv("IB_PORT", "4002"))  # 4002: IB Gateway paper, 7497: TWS paper
    IB_CLIENT_ID: int = int(os.getenv("IB_CLIENT_ID", "1"))
    IB_TIMEOUT: float = float(os.getenv("IB_TIMEOUT", "10.0"))

    # API Server Settings
    API_HOST: str = os.getenv("API_HOST", "0.0.0.0")
    API_PORT: int = int(os.getenv("API_PORT", "8000"))

    # CORS Settings (Allow dashboard frontend origins)
    CORS_ORIGINS: list[str] = [
        origin.strip()
        for origin in os.getenv("CORS_ORIGINS", "*").split(",")
        if origin.strip()
    ]


settings = Settings()
