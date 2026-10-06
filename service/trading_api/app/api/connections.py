from typing import Annotated

from fastapi import APIRouter, Depends, Query

from app.alpaca.client import create_trading_client
from app.alpaca.oauth import (
    build_authorization_url,
    decode_oauth_state,
    exchange_code,
)
from app.api.dependencies import (
    get_connection_repository,
)
from app.protocol.models import (
    ConnectionStartRequest,
    ConnectionStartResponse,
    ConnectionStatus,
)
from app.connections.connections import ConnectionRepository
from app.sec.encryption import encrypt_token
from app.sec.auth import (
    get_current_user_id,
    require_gateway,
)


router = APIRouter(prefix="/connections/alpaca")


@router.post(
    "/start",
    response_model=ConnectionStartResponse,
)
def start_connection(
    body: ConnectionStartRequest,
    user_id: Annotated[
        str,
        Depends(get_current_user_id),
    ],
):
    return ConnectionStartResponse(
        authorization_url=build_authorization_url(
            user_id=user_id,
            environment=body.environment,
        )
    )


@router.post(
    "/callback",
    dependencies=[Depends(require_gateway)],
)
async def complete_connection(
    code: Annotated[str, Query()],
    state: Annotated[str, Query()],
    repository: Annotated[
        ConnectionRepository,
        Depends(get_connection_repository),
    ],
):
    user_id, environment = decode_oauth_state(state)
    access_token = await exchange_code(code)

    client = create_trading_client(
        access_token=access_token,
        environment=environment,
    )

    account = client.get_account()

    repository.save(
        user_id=user_id,
        environment=environment,
        encrypted_access_token=encrypt_token(
            access_token
        ),
        alpaca_account_id=str(account.id),
    )

    return {
        "connected": True,
        "environment": environment,
    }


@router.get(
    "",
    response_model=ConnectionStatus,
)
def get_connection_status(
    user_id: Annotated[
        str,
        Depends(get_current_user_id),
    ],
    repository: Annotated[
        ConnectionRepository,
        Depends(get_connection_repository),
    ],
):
    connection = repository.get_active(user_id)

    if connection is None:
        return ConnectionStatus(connected=False)

    return ConnectionStatus(
        connected=True,
        environment=connection.environment,
        alpaca_account_id=connection.alpaca_account_id,
    )


@router.delete("")
def disconnect(
    user_id: Annotated[
        str,
        Depends(get_current_user_id),
    ],
    repository: Annotated[
        ConnectionRepository,
        Depends(get_connection_repository),
    ],
):
    disconnected = repository.disconnect(user_id)

    return {"disconnected": disconnected}