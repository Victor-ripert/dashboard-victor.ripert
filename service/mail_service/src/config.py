from functools import lru_cache
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    database_url: str
    gmail_client_id: str
    gmail_client_secret: str
    gmail_redirect_uri: str
    token_encryption_key: str
    jwt_secret: str
    jwt_issuer: str
    jwt_audience: str
    cors_origins: str = "http://localhost:3000"
    model_config = SettingsConfigDict(env_file=".env",extra="ignore")

    @property
    def allowed_origins(self) -> list[str]:
        return [
            origin.strip()
            for origin in self.cors_origins.split(",")
        ]

@lru_cache
def get_settings() -> Settings:
    return Settings()


settings = get_settings()
