from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from app.api import orders, positions
from service.trading_service.app.api import accounts, connections
from service.trading_service.app.config import settings
from service.trading_service.app.db import Base, engine
from service.trading_service.app.protocol.errors import TradingServiceError

app = FastAPI(title=settings.app_name, version="0.1.0")

@app.exception_handler(TradingServiceError)
async def handle_service_error(request: Request, error: TradingServiceError):
    return JSONResponse(
        status_code=error.status_code,
        content={"error": error.__class__.__name__, "message": error.message}
    )

app.include_router(connections.router, prefix="/internal/v1", tags=["connections"])
app.include_router(accounts.router, prefix="/internal/v1", tags=["account"])
app.include_router(positions.router, prefix="/internal/v1", tags=["positions"])
app.include_router(orders.router, prefix="/internal/v1", tags=["orders"])


@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/ready")
def ready():
    return {"status": "ready"}


@app.on_event("startup")
def create_database_tables():
    Base.metadata.create_all(bind=engine)
