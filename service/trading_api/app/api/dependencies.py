from typing import Annotated
from alpaca.trading.client import TradingClient
from fastapi import Depends
from sqlalchemy.orm import Session
from app.alpaca.client import set_trading_client
from app.db import get_db
from app.protocol.errors import BrokerNotConnectedError
from app.connections.connections import ConnectionRepository
from app.sec.encryption import decrypt_token
from app.sec.auth import get_current_user_id


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
        raise BrokerNotConnectedError("No active Alpaca connection")

    access_token = decrypt_token(
        connection.encrypted_access_token
    )

    return set_trading_client(
        access_token=access_token,
        environment=connection.environment,
    )


def get_user_environment(
    user_id: Annotated[str, Depends(get_current_user_id)],
    repository: Annotated[
        ConnectionRepository,
        Depends(get_connection_repository)
    ]
) -> str:
    connection = repository.get_active(user_id)

    if connection is None:
        raise BrokerNotConnectedError("No active Alpaca connection")
    else:
        return connection.environment
