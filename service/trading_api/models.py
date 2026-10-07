from enum import Enum
from typing import Optional
from pydantic import BaseModel, Field, field_validator


class ProductType(str, Enum):
    STK = "STK"
    ETF = "ETF"
    CRYPTO = "CRYPTO"
    CASH = "CASH"
    OPT = "OPT"
    FUT = "FUT"


class OrderAction(str, Enum):
    BUY = "BUY"
    SELL = "SELL"


class OrderType(str, Enum):
    MKT = "MKT"
    LMT = "LMT"
    STP = "STP"


class PurchaseRequest(BaseModel):
    symbol: str
    quantity: float = Field(..., gt=0)
    product_type: ProductType = ProductType.STK
    order_type: OrderType = OrderType.MKT
    price: Optional[float] = None
    currency: str = "USD"
    exchange: Optional[str] = None

    @field_validator("symbol", "currency")
    @classmethod
    def format_uppercase(cls, v: str) -> str:
        cleaned = v.strip().upper()
        if not cleaned:
            raise ValueError("Value cannot be empty")
        return cleaned


class PurchaseResponse(BaseModel):
    order_id: int | str
    symbol: str
    product_type: str
    action: str
    quantity: float
    order_type: str
    price: Optional[float] = None
    status: str
    filled_quantity: float = 0.0
    avg_fill_price: Optional[float] = None
    total_cost: Optional[float] = None
    currency: str
    timestamp: str
    message: Optional[str] = None


class QuoteResponse(BaseModel):
    symbol: str
    product_type: str
    currency: str
    last_price: Optional[float] = None
    bid: Optional[float] = None
    ask: Optional[float] = None
    high: Optional[float] = None
    low: Optional[float] = None
    volume: Optional[float] = None
    timestamp: str


class PositionItem(BaseModel):
    account: str
    symbol: str
    sec_type: str
    currency: str
    position: float
    avg_cost: float
    market_price: Optional[float] = None
    market_value: Optional[float] = None
    unrealized_pnl: Optional[float] = None


class AccountSummary(BaseModel):
    currency: str
    available_funds: float
    buying_power: float
    net_liquidation: float
    cash_balance: float
    mode: str


class CloseRequest(BaseModel):
    symbol: str

    @field_validator("symbol")
    @classmethod
    def clean_symbol(cls, v: str) -> str:
        return v.strip().upper()


class OrderRequest(BaseModel):
    symbol: str
    action: str
    qty: int
