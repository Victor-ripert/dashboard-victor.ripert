from typing import Annotated

from alpaca.trading.client import TradingClient
from fastapi import APIRouter, Depends

from app.alpaca.mapper import map_account
from app.api.dependencies import (
    get_user_environment,
    get_user_trading_client,
)
from app.protocol.models import AccountView


router = APIRouter(prefix="/account")


@router.get("", response_model=AccountView)
def get_account(
    client: Annotated[
        TradingClient,
        Depends(get_user_trading_client),
    ],
    environment: Annotated[
        str,
        Depends(get_user_environment),
    ],
):
    account = client.get_account()

    return map_account(account, environment)