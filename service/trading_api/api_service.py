from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException, Query, status
from fastapi.middleware.cors import CORSMiddleware

from config import settings
from ib_service import trading_service
from models import (
    AccountSummary,
    CloseRequest,
    OrderRequest,
    ProductType,
    PurchaseRequest,
    PurchaseResponse,
    QuoteResponse,
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    trading_service.start()
    yield
    trading_service.stop()


app = FastAPI(title="Trading Microservice", version="1.0.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS if settings.CORS_ORIGINS != ["*"] else ["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
@app.get("/api/v1/health")
def health_check():
    return {
        "status": "healthy",
        "service": "trading-api-microservice",
        "version": "1.0.0",
        "ib_connected": trading_service.is_connected,
        "mode": "MOCK" if trading_service.use_mock else "LIVE",
        "ib_host": settings.IB_HOST,
        "ib_port": settings.IB_PORT,
    }


@app.post("/api/v1/system/reconnect")
def reconnect():
    return trading_service.reconnect()


@app.get("/api/v1/products/catalog")
def get_catalog():
    return {
        "categories": [
            {"type": "STK", "name": "Stocks & Equities", "description": "US and Global publicly traded stocks"},
            {"type": "ETF", "name": "ETFs", "description": "Exchange-Traded Funds tracking indexes and sectors"},
            {"type": "CRYPTO", "name": "Cryptocurrency", "description": "Digital assets (BTC, ETH, SOL) via Paxos"},
            {"type": "CASH", "name": "Forex", "description": "Foreign currency pairs (EUR/USD, GBP/USD)"},
        ],
        "featured_products": [
            {"symbol": "AAPL", "name": "Apple Inc.", "product_type": "STK", "currency": "USD"},
            {"symbol": "TSLA", "name": "Tesla Inc.", "product_type": "STK", "currency": "USD"},
            {"symbol": "NVDA", "name": "NVIDIA Corp.", "product_type": "STK", "currency": "USD"},
            {"symbol": "MSFT", "name": "Microsoft Corp.", "product_type": "STK", "currency": "USD"},
            {"symbol": "SPY", "name": "SPDR S&P 500 ETF Trust", "product_type": "ETF", "currency": "USD"},
            {"symbol": "QQQ", "name": "Invesco QQQ Trust", "product_type": "ETF", "currency": "USD"},
            {"symbol": "BTC", "name": "Bitcoin", "product_type": "CRYPTO", "currency": "USD"},
            {"symbol": "ETH", "name": "Ethereum", "product_type": "CRYPTO", "currency": "USD"},
            {"symbol": "EURUSD", "name": "Euro / US Dollar", "product_type": "CASH", "currency": "USD"},
        ],
    }


@app.get("/api/v1/products/quote/{symbol}", response_model=QuoteResponse)
@app.get("/quote/{symbol}", response_model=QuoteResponse)
def get_quote(symbol: str, product_type: ProductType = Query(default=ProductType.STK), currency: str = Query(default="USD")):
    try:
        return trading_service.submit_task("quote", {"symbol": symbol, "product_type": product_type.value, "currency": currency})
    except Exception as e:
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))


@app.post("/api/v1/products/purchase", response_model=PurchaseResponse)
@app.post("/purchase", response_model=PurchaseResponse)
def purchase_product(req: PurchaseRequest):
    try:
        return trading_service.submit_task("purchase", {
            "symbol": req.symbol,
            "action": "BUY",
            "quantity": req.quantity,
            "product_type": req.product_type.value,
            "order_type": req.order_type.value,
            "price": req.price,
            "currency": req.currency,
            "exchange": req.exchange,
        })
    except ValueError as e:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(e))
    except TimeoutError as e:
        raise HTTPException(status_code=status.HTTP_504_GATEWAY_TIMEOUT, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))


@app.get("/api/v1/account", response_model=AccountSummary)
def get_account():
    try:
        return trading_service.submit_task("account_summary", {})
    except Exception as e:
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))


@app.get("/api/v1/available_funds")
@app.get("/available_funds")
def get_available_funds():
    try:
        funds = trading_service.submit_task("available_funds", {}, timeout=10.0)
        return {"currency": "EUR", "available_funds": funds}
    except Exception as e:
        return {"currency": "EUR", "available_funds": None, "error": str(e)}


@app.get("/api/v1/positions")
@app.get("/positions")
def get_positions():
    try:
        data = trading_service.submit_task("positions", {}, timeout=5.0)
        return {"positions": data}
    except Exception as e:
        return {"positions": [], "error": str(e)}


@app.post("/api/v1/positions/{symbol}/close")
def close_position_route(symbol: str):
    try:
        return trading_service.submit_task("close_position", {"symbol": symbol.upper().strip()})
    except Exception as e:
        raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))


@app.post("/close_position")
def close_position_legacy(req: CloseRequest):
    try:
        return trading_service.submit_task("close_position", {"symbol": req.symbol}, timeout=10.0)
    except Exception as e:
        return {"symbol": req.symbol, "closed": False, "error": str(e)}


@app.post("/place_order")
def place_order(req: OrderRequest):
    try:
        resp = trading_service.submit_task("place_order", {
            "symbol": req.symbol,
            "action": req.action,
            "quantity": req.qty,
            "product_type": "STK",
            "order_type": "MKT",
            "currency": "USD",
            "exchange": "SMART",
        }, timeout=5.0)
        return {"orderId": resp.get("order_id", resp.get("orderId")), "status": resp.get("status")}
    except Exception as e:
        return {"error": str(e)}


@app.get("/order_status/{order_id}")
def order_status(order_id: int):
    try:
        return trading_service.submit_task("order_status", {"orderId": order_id}, timeout=5.0)
    except Exception as e:
        return {"orderId": order_id, "error": str(e)}


@app.get("/shutdown_ib")
def shutdown_ib():
    try:
        return {"status": trading_service.submit_task("shutdown", {}, timeout=5.0)}
    except Exception as e:
        return {"status": "error", "message": str(e)}
