from typing import Annotated
from fastapi import Depends, Header, HTTPException
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from jose import JWTError, jwt
from service.trading_service.app.config import settings

bearer = HTTPBearer(auto_error=True)

def require_gateway(credentials: Annotated[HTTPAuthorizationCredentials, Depends(bearer)]) -> None:
    try:
        payload = jwt.decode(
            credentials.credentials,
            settings.internal_jwt_secret,
            algorithms=["HS256"],
            issuer=settings.internal_jwt_issuer,
            audience=settings.internal_jwt_audience,
        )
    except JWTError as error:
        raise HTTPException(
            status_code=401,
            detail="Invalid internal token",
        ) from error

    if payload.get("sub") != "api-gateway":
        raise HTTPException(
            status_code=403,
            detail="Invalid service identity",
        )


def get_current_user_id(
    x_user_id: Annotated[str, Header(alias="X-User-Id")],
    _: Annotated[None, Depends(require_gateway)],
) -> str:
    if not x_user_id:
        raise HTTPException(
            status_code=400,
            detail="Missing user ID",
        )

    return x_user_id
