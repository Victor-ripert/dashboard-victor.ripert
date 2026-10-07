from datetime import datetime, timedelta, timezone
from urllib.parse import urlencode

import httpx
from jose import JWTError, jwt

from service.trading_service.app.config import settings
from service.trading_service.app.protocol.errors import AlpacaAPIError, InvalidOAuthStateError


AUTHORIZE_URL = "https://app.alpaca.markets/oauth/authorize"
TOKEN_URL = "https://api.alpaca.markets/oauth/token"


def create_oauth_state(
    user_id: str,
    environment: str,
) -> str:
    now = datetime.now(timezone.utc)

    payload = {
        "sub": user_id,
        "environment": environment,
        "iat": now,
        "exp": now + timedelta(minutes=10),
        "aud": "alpaca-oauth",
        "iss": "trading-service",
    }

    return jwt.encode(
        payload,
        settings.oauth_state_secret,
        algorithm="HS256",
    )


def decode_oauth_state(state: str) -> tuple[str, str]:
    try:
        payload = jwt.decode(
            state,
            settings.oauth_state_secret,
            algorithms=["HS256"],
            audience="alpaca-oauth",
            issuer="trading-service",
        )
    except JWTError as error:
        raise InvalidOAuthStateError(
            "OAuth state is invalid or expired"
        ) from error

    user_id = payload.get("sub")
    environment = payload.get("environment")

    if not user_id or environment not in {"paper", "live"}:
        raise InvalidOAuthStateError("Invalid OAuth state")

    return user_id, environment


def build_authorization_url(
    user_id: str,
    environment: str,
) -> str:
    state = create_oauth_state(user_id, environment)

    parameters = {
        "response_type": "code",
        "client_id": settings.alpaca_client_id,
        "redirect_uri": settings.alpaca_redirect_uri,
        "state": state,
        "scope": "trading",
        "env": environment,
    }

    return f"{AUTHORIZE_URL}?{urlencode(parameters)}"


async def exchange_code(code: str) -> str:
    data = {
        "grant_type": "authorization_code",
        "code": code,
        "client_id": settings.alpaca_client_id,
        "client_secret": settings.alpaca_client_secret,
        "redirect_uri": settings.alpaca_redirect_uri,
    }

    async with httpx.AsyncClient(timeout=10) as client:
        response = await client.post(
            TOKEN_URL,
            data=data,
            headers={
                "Content-Type":
                    "application/x-www-form-urlencoded"
            },
        )

    if response.is_error:
        raise AlpacaAPIError(
            f"OAuth exchange failed: {response.status_code}"
        )

    access_token = response.json().get("access_token")

    if not access_token:
        raise AlpacaAPIError(
            "Alpaca did not return an access token"
        )

    return access_token
