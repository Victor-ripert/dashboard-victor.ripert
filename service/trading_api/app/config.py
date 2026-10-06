from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    app_name: str = "Alpaca Trading Service"
    environment: str = "development"

    database_url: str

    alpaca_client_id: str
    alpaca_client_secret: str
    alpaca_redirect_uri: str

    token_encryption_key: str
    oauth_state_secret: str

    internal_jwt_secret: str
    internal_jwt_issuer: str = "api-gateway"
    internal_jwt_audience: str = "trading-service"

    model_config = SettingsConfigDict(
        env_file=".env",
        case_sensitive=False,
        extra="ignore",
    )

@lru_cache
def get_settings() -> Settings:
    return Settings()


settings = get_settings()
