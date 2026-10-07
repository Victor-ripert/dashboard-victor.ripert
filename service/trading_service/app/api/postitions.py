from typing import Annotated

from alpaca.trading.client import TradingClient
from fastapi import Depends
from sqlalchemy.orm import Session

from service.trading_service.app.alpaca.client import create_trading_client
from service.trading_service.app.db import get_db
from service.trading_service.app.protocol.errors import BrokerNotConnectedError
from service.trading_service.app.connections.connections import ConnectionRepository
from service.trading_service.app.sec.encryption import decrypt_token
from service.trading_service.app.sec.auth import get_current_user_id


def get_connection_repository(
    db: Annotated[Session, Depends(get_db)],
) -> ConnectionRepository:
    return ConnectionRepository(db)


def get_user_trading_client(
    user_id: Annotated[str, Depends(get_current_user_id)],
    repository: Annotated[
        ConnectionRepository,
        Depends(get_connection_repository),
    ],
) -> TradingClient:
    connection = repository.get_active(user_id)

    if connection is None:
        raise BrokerNotConnectedError(
            "No active Alpaca connection"
        )

    access_token = decrypt_token(
        connection.encrypted_access_token
    )

    return create_trading_client(
        access_token=access_token,
        environment=connection.environment,
    )


def get_user_environment(
    user_id: Annotated[str, Depends(get_current_user_id)],
    repository: Annotated[
        ConnectionRepository,
        Depends(get_connection_repository),
    ],
) -> str:
    connection = repository.get_active(user_id)

    if connection is None:
        raise BrokerNotConnectedError(
            "No active Alpaca connection"
        )

    return connection.environment
