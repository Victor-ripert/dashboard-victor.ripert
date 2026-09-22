import logging
from contextlib import asynccontextmanager
from typing import Any, Dict, List, Optional

from fastapi import FastAPI, HTTPException, Query, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from config import settings
from ib_service import trading_service
from models import (
    AccountSummary,
    CloseRequest,
    OrderRequest,
    PositionItem,
    ProductType,
    PurchaseRequest,
    PurchaseResponse,
    QuoteResponse,
)

logger = logging.getLogger("trading_microservice")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Start the background IB trading service
    trading_service.start()
    logger.info("Trading API Microservice initialized successfully.")
    yield
    # Shutdown: Cleanly disconnect
    trading_service.stop()
    logger.info("Trading API Microservice shut down.")


app = FastAPI(
    title="Trading Microservice",
    description=(
        "Microservice API for financial dashboards allowing users to browse products, "
        "retrieve real-time quotes, purchase financial products (Stocks, ETFs, Crypto, Forex), "
        "track active positions, and monitor account funds via Interactive Brokers or simulation."
    ),
    version="1.0.0",
    lifespan=lifespan,
)

# Enable CORS for dashboard web clients (React, Vue, Vite, Next.js, etc.)
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS if settings.CORS_ORIGINS != ["*"] else ["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ==========================================
# 1. Health & System Endpoints
# ==========================================

@app.get("/health", tags=["System"])
@app.get("/api/v1/health", tags=["System"])
def health_check():
    """Health check endpoint for container orchestrators and dashboard status monitors."""
    return {
        "status": "healthy",
        "service": "trading-api-microservice",
        "version": "1.0.0",
        "ib_connected": trading_service.is_connected,
        "mode": "MOCK" if trading_service.use_mock else "LIVE",
        "ib_host": settings.IB_HOST,
        "ib_port": settings.IB_PORT,
    }


@app.post("/api/v1/system/reconnect", tags=["System"])
def reconnect_ib():
    """Trigger manual reconnection to Interactive Brokers Gateway / TWS."""
    return trading_service.reconnect()


# ==========================================
# 2. Financial Products & Purchasing (Main)
# ==========================================

@app.get("/api/v1/products/catalog", tags=["Products"])
def get_product_catalog():
    """Returns supported product categories and featured financial assets for the dashboard."""
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


@app.get("/api/v1/products/quote/{symbol}", response_model=QuoteResponse, tags=["Products"])
@app.get("/quote/{symbol}", response_model=QuoteResponse, tags=["Products"])
def get_product_quote(
    symbol: str,
    product_type: ProductType = Query(default=ProductType.STK, description="Product category"),
    currency: str = Query(default="USD", description="Currency"),
):
    """Retrieve the latest market quote (price, bid, ask, volume) for a financial product."""
    try:
        quote = trading_service.submit_task(
            "quote",
            {
                "symbol": symbol.upper().strip(),
                "product_type": product_type.value,
                "currency": currency.upper().strip(),
            },
        )
        return quote
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to fetch market quote for {symbol}: {str(e)}",
        )


@app.post("/api/v1/products/purchase", response_model=PurchaseResponse, tags=["Trading"])
@app.post("/purchase", response_model=PurchaseResponse, tags=["Trading"])
def purchase_financial_product(req: PurchaseRequest):
    """Purchase a financial product (Stock, ETF, Crypto, Forex) from the dashboard.

    Validates parameters, checks available funds, routes the order to Interactive Brokers
    (or simulation engine in mock mode), and returns the execution receipt.
    """
    try:
        result = trading_service.submit_task(
            "purchase",
            {
                "symbol": req.symbol,
                "action": "BUY",
                "quantity": req.quantity,
                "product_type": req.product_type.value,
                "order_type": req.order_type.value,
                "price": req.price,
                "currency": req.currency,
                "exchange": req.exchange,
            },
        )
        return result
    except ValueError as val_err:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(val_err))
    except TimeoutError as timeout_err:
        raise HTTPException(status_code=status.HTTP_504_GATEWAY_TIMEOUT, detail=str(timeout_err))
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to purchase product {req.symbol}: {str(e)}",
        )


# ==========================================
# 3. Account & Portfolio Endpoints
# ==========================================

@app.get("/api/v1/account", response_model=AccountSummary, tags=["Account"])
def get_account_summary():
    """Returns comprehensive account details: available funds, buying power, equity, and cash balance."""
    try:
        return trading_service.submit_task("account_summary", {})
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to retrieve account summary: {str(e)}",
        )


@app.get("/api/v1/available_funds", tags=["Account"])
def get_available_funds_v1():
    """Returns available purchasing funds."""
    try:
        funds = trading_service.submit_task("available_funds", {})
        return {"currency": "EUR", "available_funds": funds}
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get available funds: {str(e)}",
        )


@app.get("/api/v1/positions", tags=["Portfolio"])
def get_positions():
    """Returns all open portfolio positions for display on the dashboard."""
    try:
        positions = trading_service.submit_task("positions", {})
        return {"positions": positions}
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get positions: {str(e)}",
        )


@app.post("/api/v1/positions/{symbol}/close", tags=["Portfolio"])
def close_position(symbol: str):
    """Closes all holdings of the specified financial product."""
    try:
        return trading_service.submit_task("close_position", {"symbol": symbol.upper().strip()})
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to close position for {symbol}: {str(e)}",
        )


# ==========================================
# 4. Backward Compatibility Endpoints
# ==========================================

@app.get("/available_funds", tags=["Legacy Compatibility"])
def get_available_funds():
    """Legacy endpoint returning available funds."""
    try:
        funds = trading_service.submit_task("available_funds", {}, timeout=10.0)
        return {"currency": "EUR", "available_funds": funds}
    except TimeoutError:
        return {"currency": "EUR", "available_funds": None, "error": "timeout"}
    except Exception as e:
        return {"currency": "EUR", "available_funds": None, "error": str(e)}


@app.get("/positions", tags=["Legacy Compatibility"])
def get_positions():
    """Legacy endpoint returning current positions."""
    try:
        data = trading_service.submit_task("positions", {}, timeout=5.0)
        return {"positions": data}
    except TimeoutError:
        return {"positions": [], "error": "timeout"}
    except Exception as e:
        return {"positions": [], "error": str(e)}


@app.post("/place_order", tags=["Legacy Compatibility"])
def place_order(req: OrderRequest):
    """Legacy order placement endpoint."""
    try:
        resp = trading_service.submit_task(
            "place_order",
            {
                "symbol": req.symbol,
                "action": req.action,
                "quantity": req.qty,
                "product_type": "STK",
                "order_type": "MKT",
                "currency": "USD",
                "exchange": "SMART",
            },
            timeout=5.0,
        )
        return {
            "orderId": resp.get("order_id", resp.get("orderId")),
            "status": resp.get("status"),
        }
    except TimeoutError:
        return {"error": "timeout"}
    except Exception as e:
        return {"error": str(e)}


@app.get("/order_status/{order_id}", tags=["Legacy Compatibility"])
def order_status(order_id: int):
    """Legacy order status lookup endpoint."""
    try:
        resp = trading_service.submit_task("order_status", {"orderId": order_id}, timeout=5.0)
        return resp
    except TimeoutError:
        return {"orderId": order_id, "error": "timeout"}
    except Exception as e:
        return {"orderId": order_id, "error": str(e)}


@app.post("/close_position", tags=["Legacy Compatibility"])
def close_position(req: CloseRequest):
    """Legacy close position endpoint."""
    try:
        resp = trading_service.submit_task("close_position", {"symbol": req.symbol}, timeout=10.0)
        return resp
    except TimeoutError:
        return {"symbol": req.symbol, "closed": False, "error": "timeout"}
    except Exception as e:
        return {"symbol": req.symbol, "closed": False, "error": str(e)}


@app.get("/shutdown_ib", tags=["Legacy Compatibility"])
def shutdown_ib():
    """Legacy shutdown endpoint."""
    try:
        msg = trading_service.submit_task("shutdown", {}, timeout=5.0)
        return {"status": msg}
    except TimeoutError:
        return {"status": "timeout"}
    except Exception as e:
        return {"status": "error", "message": str(e)}
