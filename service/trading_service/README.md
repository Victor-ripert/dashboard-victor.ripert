# Trading API

FastAPI microservice for trading operations (Interactive Brokers & mock simulation).

## Setup & Run

```bash
pip install -r requirements.txt
uvicorn api_service:app --reload --port 8000
```

Or using Docker:

```bash
docker compose up --build
```

## Tests

```bash
pytest
```
