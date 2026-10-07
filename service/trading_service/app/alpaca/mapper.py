from typing import Any
from service.trading_service.app.protocol.models import AccountView, OrderView, PositionView


def enum_value(value: Any) -> str:
    if hasattr(value, "value"):
        return str(value.value)
    else :
        return str(value)


def map_account(account: Any, environment: str) -> AccountView:
    return AccountView(
        id=str(account.id),
        status=enum_value(account.status),
        currency=str(account.currency),
        cash=float(account.cash),
        buying_power=float(account.buying_power),
        portfolio_value=float(account.portfolio_value),
        environment=environment,
    )


def map_position(position: Any) -> PositionView:
    return PositionView(
        symbol=position.symbol,
        quantity=float(position.qty),
        average_entry_price=float(position.avg_entry_price),
        current_price=float(position.current_price),
        market_value=float(position.market_value),
        unrealized_profit_loss=float(position.unrealized_pl),
    )


def map_order(order: Any) -> OrderView:
    submitted_at = None

    if order.submitted_at is not None:
        submitted_at = order.submitted_at.isoformat()

    return OrderView(
        id=str(order.id),
        symbol=order.symbol,
        side=enum_value(order.side),
        quantity=float(order.qty),
        order_type=enum_value(order.type),
        status=enum_value(order.status),
        submitted_at=submitted_at
    )
