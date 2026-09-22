# Financial Dashboard Multi-Service Architecture

A modular multi-service architecture designed for financial dashboards. The `service/` directory hosts isolated microservices, each with its own technology stack, environment, dependencies, and container lifecycle.

Services can be developed, tested, and launched **independently** from their respective directories, or orchestrated **collectively** using Docker Compose at the root.

---

## Directory Structure

```
dashboard/
├── docker-compose.yml              # Root orchestrator for all services
├── README.md                       # Project overview and architecture guide
├── .gitignore                      # Shared git ignore rules
│
└── service/                        # Microservices directory
    │
    ├── trading_api/                # Interactive Brokers & Product Purchasing Microservice
    │   ├── .venv/                  # Independent Python virtual environment
    │   ├── requirements.txt        # Clean Python dependencies
    │   ├── Dockerfile              # Independent Dockerfile
    │   ├── docker-compose.yml      # Independent Docker Compose runner
    │   ├── .env.example            # Environment variables template
    │   ├── .env                    # Local environment config
    │   ├── api_service.py          # FastAPI application (purchasing, quotes, account)
    │   ├── ib_service.py           # Thread-safe IB connection & mock trading engine
    │   ├── models.py               # Pydantic schemas for orders, quotes, products
    │   ├── config.py               # Settings loader
    │   ├── ib_worker.py            # Legacy backward-compatibility bridge
    │   ├── tests/                  # Service-specific test suite
    │   │   ├── __init__.py
    │   │   └── test_api.py
    │   └── README.md               # Detailed service documentation
    │
    ├── (future_service_1)/         # e.g., dashboard-frontend (React, Next.js, Vue)
    │   ├── Dockerfile
    │   └── ...
    │
    └── (future_service_2)/         # e.g., market-analytics, auth-service, etc.
        ├── Dockerfile
        └── ...
```

---

## How to Run Services

### Option A: Launch All Services (Root Orchestration)

From the project root (`/home/antonin-m/epitech/s5/dashboard`), Docker Compose will spin up **IB Gateway** (`ghcr.io/gnzsnz/ib-gateway`) and the **Trading API** automatically:

```bash
# Build and launch IB Gateway and all microservices in the background
docker compose up --build -d

# View consolidated logs
docker compose logs -f

# Shut down all services
docker compose down
```

---

### Option B: Launch `trading_api` Independently

Each service inside `service/` is 100% self-contained and can be run independently:

#### With Docker Compose:
```bash
cd service/trading_api
docker compose up --build -d
```

#### Locally with its own `.venv`:
```bash
cd service/trading_api

# Activate the service virtual environment
source .venv/bin/activate

# Run FastAPI via uvicorn
uvicorn api_service:app --host 0.0.0.0 --port 8000 --reload
```

- **Swagger Documentation**: [http://localhost:8000/docs](http://localhost:8000/docs)
- **Health Check**: [http://localhost:8000/health](http://localhost:8000/health)

#### Run Service Tests:
```bash
cd service/trading_api
.venv/bin/pytest -v tests/
```

---

## Adding a New Service to `service/`

To add a new microservice with a different tech stack (e.g. Node.js, Go, Rust, or Python):

1. **Create the service directory**:
   ```bash
   mkdir -p service/my_new_service
   ```
2. **Add its own stack and files**:
   - `Dockerfile`
   - `docker-compose.yml` (for standalone execution)
   - `.env.example`
   - Dependencies file (e.g. `package.json`, `go.mod`, or `requirements.txt`)
   - Source code and tests
3. **Register the service in the root `docker-compose.yml`** so it can optionally be started alongside other services with `docker compose up`.

---

## Current Service: `trading_api`

- **Purpose**: Provides order execution, portfolio management, and market quotes for the dashboard.
- **Main Endpoint**: `POST /api/v1/products/purchase` (purchases Stocks, ETFs, Crypto, Forex).
- **Interactive Brokers**: Connects via `ib_insync` with automatic fallback to an in-memory simulation engine if IB Gateway is not running.
- **CORS**: Enabled by default to accept requests from web dashboards.
- See [`service/trading_api/README.md`](file:///home/antonin-m/epitech/s5/dashboard/service/trading_api/README.md) for full endpoint specifications and frontend integration examples.
