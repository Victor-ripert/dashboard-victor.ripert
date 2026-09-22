# Trading API Microservice (`service/trading_api`)

Independent microservice providing trading operations, market quotes, and financial product purchasing for the dashboard frontend. Connects to Interactive Brokers (IB Gateway / TWS) with automatic in-memory simulation fallback.

---

## Service Architecture

```
service/trading_api/
├── .venv/                 # Dedicated Python virtual environment
├── requirements.txt       # Python dependencies
├── Dockerfile             # Container build specification
├── docker-compose.yml     # Standalone Docker compose runner
├── .env.example           # Environment template
├── .env                   # Local configuration
├── api_service.py         # FastAPI endpoints (purchase, quotes, positions, account, CORS)
├── ib_service.py          # Future-based async IB worker & simulation engine
├── models.py              # Pydantic data schemas
├── config.py              # Environment configuration loader
├── ib_worker.py           # Legacy backward-compatibility bridge
├── tests/                 # Unit and integration test suite
│   ├── __init__.py
│   └── test_api.py
└── README.md              # Service documentation
```

---

## Independent Launch & Development

This service can be developed, tested, and launched independently of any other services.

### 1. Local Python Setup

```bash
cd service/trading_api

# Activate virtual environment
source .venv/bin/activate

# Install dependencies (if needed)
pip install -r requirements.txt

# Run service
uvicorn api_service:app --host 0.0.0.0 --port 8000 --reload
```

Interactive Docs:
- Swagger UI: `http://localhost:8000/docs`
- ReDoc: `http://localhost:8000/redoc`
- Health: `http://localhost:8000/health`

### 2. Standalone Docker Launch (with IB Gateway)

When launching via Docker Compose, **IB Gateway** (`ghcr.io/gnzsnz/ib-gateway`) is automatically started alongside `trading-api`:

```bash
cd service/trading_api

# Build and start both IB Gateway & Trading API
docker compose up --build -d

# View live logs
docker compose logs -f

# Shut down
docker compose down
```

Credentials for IB Gateway can be configured in `.env`:
- `TWS_USERID`: IB paper trading username (defaults to `edemo` for built-in demo)
- `TWS_PASSWORD`: IB paper trading password (defaults to `demouser`)
- `TRADING_MODE`: `paper`
- `READ_ONLY_API`: `no` (allows order placement)

### 3. Run Tests

```bash
cd service/trading_api
.venv/bin/pytest -v tests/
```

---

## Purchase API Endpoint

### `POST /api/v1/products/purchase` (or `/purchase`)

```json
{
  "symbol": "AAPL",
  "quantity": 10,
  "product_type": "STK",
  "order_type": "MKT",
  "currency": "USD"
}
```

Supports:
- `STK` (Stocks / Equities)
- `ETF` (Exchange Traded Funds)
- `CRYPTO` (Crypto assets like BTC, ETH via Paxos)
- `CASH` (Forex pairs like EURUSD)
- `OPT` / `FUT` (Options and Futures)
