from enum import Enum
from typing import Optional, List
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
    MKT = "MKT"       # Market Order
    LMT = "LMT"       # Limit Order
    STP = "STP"       # Stop Order

class PurchaseRequest(BaseModel):
    """Request schema for purchasing a financial product from the dashboard."""
    symbol: str = Field(..., description="Financial product ticker or symbol, e.g., 'AAPL', 'BTC', 'EURUSD'")
    quantity: float = Field(..., gt=0, description="Quantity to purchase (must be > 0)")
    product_type: ProductType = Field(default=ProductType.STK, description="Product category: STK, ETF, CRYPTO, CASH, OPT, FUT")
    order_type: OrderType = Field(default=OrderType.MKT, description="Order execution type: MKT (Market), LMT (Limit), STP (Stop)")
    price: Optional[float] = Field(default=None, description="Limit price or Stop price (required if order_type is LMT or STP)")
    currency: str = Field(default="USD", description="Currency (USD, EUR, GBP, etc.)")
    exchange: Optional[str] = Field(default=None, description="Target exchange (e.g. SMART, ISLAND, PAXOS, IDEALPRO)")

    @field_validator("symbol")
    @classmethod
    def clean_symbol(cls, v: str) -> str:
        cleaned = v.strip().upper()
        if not cleaned:
            raise ValueError("Symbol cannot be empty")
        return cleaned

    @field_validator("currency")
    @classmethod
    def clean_currency(cls, v: str) -> str:
        return v.strip().upper()

class PurchaseResponse(BaseModel):
    """Response returned upon purchasing a financial product."""
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
    """Real-time market quote for a financial product."""
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
    """Active portfolio position details."""
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
    """Financial account overview for the dashboard."""
    currency: str
    available_funds: float
    buying_power: float
    net_liquidation: float
    cash_balance: float
    mode: str  # "LIVE" or "MOCK"


class CloseRequest(BaseModel):
    """Request to close an existing position."""
    symbol: str

    @field_validator("symbol")
    @classmethod
    def clean_symbol(cls, v: str) -> str:
        return v.strip().upper()


class OrderRequest(BaseModel):
    """Legacy order request for backward compatibility."""
    symbol: str
    action: str
    qty: int
