from typing import Annotated
from alpaca.trading.client import TradingClient
from alpaca.trading.enums import OrderSide, TimeInForce
from alpaca.trading.requests import LimitOrderRequest, MarketOrderRequest
from fastapi import APIRouter, Depends
from service.trading_service.app.alpaca.mapper import map_order
from service.trading_service.app.api.dependencies import get_user_trading_client
from service.trading_service.app.protocol.models import OrderCreate, OrderView


router = APIRouter(prefix="/orders")


def build_order_request(body: OrderCreate):
    side = (
        OrderSide.BUY
        if body.side == "buy"
        else OrderSide.SELL
    )
    time_in_force = (
        TimeInForce.DAY
        if body.time_in_force == "day"
        else TimeInForce.GTC
    )

    common_fields = {
        "symbol": body.symbol,
        "qty": body.quantity,
        "side": side,
        "time_in_force": time_in_force,
    }

    if body.order_type == "limit":
        return LimitOrderRequest(**common_fields, limit_price=body.limit_price)
    else:
        return MarketOrderRequest(**common_fields)

@router.post("", response_model=OrderView)
def place_order(body: OrderCreate, client: Annotated[TradingClient, Depends(get_user_trading_client)]):
    order_request = build_order_request(body)
    order = client.submit_order(order_data=order_request)
    return map_order(order)

@router.delete("/{order_id}")
def cancel_order(
    order_id: str,
    client: Annotated[
        TradingClient,
        Depends(get_user_trading_client)
    ]
):
    client.cancel_order_by_id(order_id)
    return {"order_id": order_id, "cancelled": True}
