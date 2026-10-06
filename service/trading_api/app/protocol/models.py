from typing import Literal
from pydantic import BaseModel, Field, model_validator

BrokerEnvironment = Literal["paper", "live"]
OrderSide = Literal["buy", "sell"]
OrderType = Literal["market", "limit"]
TimeInForce = Literal["day", "gtc"]

class ConnectionStartRequest(BaseModel):
    environment: BrokerEnvironment = "paper"

class ConnectionStartResponse(BaseModel):
    authorization_url: str

class ConnectionStatus(BaseModel):
    connected: bool
    environment: BrokerEnvironment | None = None
    alpaca_account_id: str | None = None

class AccountView(BaseModel):
    id: str
    status: str
    currency: str
    cash: float
    buying_power: float
    portfolio_value: float
    environment: BrokerEnvironment

class PositionView(BaseModel):
    symbol: str
    quantity: float
    average_entry_price: float
    current_price: float
    market_value: float
    unrealized_profit_loss: float

class OrderCreate(BaseModel):
    symbol: str = Field(min_length=1, max_length=20)
    side: OrderSide
    quantity: float = Field(gt=0)
    order_type: OrderType = "market"
    time_in_force: TimeInForce = "day"
    limit_price: float | None = Field(default=None, gt=0)

    @model_validator(mode="after")
    def validate_limit_price(self):
        if self.order_type == "limit" and self.limit_price is None:
            raise ValueError(
                "limit_price is required for a limit order"
            )

        if self.order_type == "market" and self.limit_price is not None:
            raise ValueError(
                "limit_price must not be provided for a market order"
            )

        self.symbol = self.symbol.upper()
        return self

class OrderView(BaseModel):
    id: str
    symbol: str
    side: str
    quantity: float
    order_type: str
    status: str
    submitted_at: str | None = None
