import pytest
from fastapi.testclient import TestClient

from api_service import app
from ib_service import trading_service


@pytest.fixture(scope="session", autouse=True)
def init_trading():
    trading_service.use_mock = True
    trading_service.start()
    yield
    trading_service.stop()


@pytest.fixture
def client():
    with TestClient(app) as test_client:
        yield test_client


def test_health_check(client):
    response = client.get("/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "healthy"
    assert data["service"] == "trading-api-microservice"
    assert data["mode"] in ("MOCK", "LIVE")


def test_products_catalog(client):
    response = client.get("/api/v1/products/catalog")
    assert response.status_code == 200
    data = response.json()
    assert "categories" in data
    assert "featured_products" in data
    category_types = [c["type"] for c in data["categories"]]
    assert "STK" in category_types
    assert "CRYPTO" in category_types


def test_product_quote(client):
    response = client.get("/api/v1/products/quote/AAPL?product_type=STK&currency=USD")
    assert response.status_code == 200
    data = response.json()
    assert data["symbol"] == "AAPL"
    assert data["last_price"] > 0
    assert data["bid"] is not None
    assert data["ask"] is not None


def test_purchase_stock(client):
    # Purchase 10 shares of Apple stock
    payload = {
        "symbol": "AAPL",
        "quantity": 10.0,
        "product_type": "STK",
        "order_type": "MKT",
        "currency": "USD"
    }
    response = client.post("/api/v1/products/purchase", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["symbol"] == "AAPL"
    assert data["action"] == "BUY"
    assert data["quantity"] == 10.0
    assert data["status"] == "FILLED"
    assert data["filled_quantity"] == 10.0
    assert data["total_cost"] > 0

    # Verify positions endpoint reflects this purchase
    pos_resp = client.get("/api/v1/positions")
    assert pos_resp.status_code == 200
    positions = pos_resp.json()["positions"]
    aapl_pos = next((p for p in positions if p["symbol"] == "AAPL"), None)
    assert aapl_pos is not None
    assert aapl_pos["position"] >= 10.0


def test_purchase_crypto(client):
    # Purchase 0.5 Bitcoin
    payload = {
        "symbol": "BTC",
        "quantity": 0.5,
        "product_type": "CRYPTO",
        "order_type": "MKT",
        "currency": "USD"
    }
    response = client.post("/purchase", json=payload)  # test alias route
    assert response.status_code == 200
    data = response.json()
    assert data["symbol"] == "BTC"
    assert data["product_type"] == "CRYPTO"
    assert data["quantity"] == 0.5
    assert data["status"] == "FILLED"


def test_purchase_limit_order(client):
    # Place a limit order purchase
    payload = {
        "symbol": "MSFT",
        "quantity": 5.0,
        "product_type": "STK",
        "order_type": "LMT",
        "price": 400.0,
        "currency": "USD"
    }
    response = client.post("/api/v1/products/purchase", json=payload)
    assert response.status_code == 200
    data = response.json()
    assert data["symbol"] == "MSFT"
    assert data["order_type"] == "LMT"
    assert data["price"] == 400.0
    assert data["total_cost"] == 5.0 * 400.0


def test_purchase_validation_error(client):
    # Negative / zero quantity should fail validation
    payload = {
        "symbol": "AAPL",
        "quantity": 0,
        "product_type": "STK"
    }
    response = client.post("/api/v1/products/purchase", json=payload)
    assert response.status_code == 422  # Pydantic validation error


def test_account_summary(client):
    response = client.get("/api/v1/account")
    assert response.status_code == 200
    data = response.json()
    assert "available_funds" in data
    assert "buying_power" in data
    assert "net_liquidation" in data
    assert data["available_funds"] > 0


def test_close_position(client):
    # Ensure position exists
    client.post("/api/v1/products/purchase", json={
        "symbol": "TSLA",
        "quantity": 4.0,
        "product_type": "STK"
    })

    # Close TSLA position
    close_resp = client.post("/api/v1/positions/TSLA/close")
    assert close_resp.status_code == 200
    data = close_resp.json()
    assert data["symbol"] == "TSLA"
    assert data["closed"] is True

    # Position should now be 0 or removed
    pos_resp = client.get("/api/v1/positions")
    positions = pos_resp.json()["positions"]
    tsla_pos = next((p for p in positions if p["symbol"] == "TSLA"), None)
    assert tsla_pos is None or tsla_pos["position"] == 0


def test_legacy_endpoints(client):
    # Legacy available_funds
    funds_resp = client.get("/available_funds")
    assert funds_resp.status_code == 200
    assert "available_funds" in funds_resp.json()

    # Legacy place_order
    order_resp = client.post("/place_order", json={
        "symbol": "NVDA",
        "action": "BUY",
        "qty": 5
    })
    assert order_resp.status_code == 200
    order_data = order_resp.json()
    assert "orderId" in order_data

    # Legacy order_status
    order_id = order_data["orderId"]
    status_resp = client.get(f"/order_status/{order_id}")
    assert status_resp.status_code == 200
    assert status_resp.json()["found"] is True

    # Legacy positions
    legacy_pos = client.get("/positions")
    assert legacy_pos.status_code == 200
    assert "positions" in legacy_pos.json()

    # Legacy close_position
    close_resp = client.post("/close_position", json={"symbol": "NVDA"})
    assert close_resp.status_code == 200
    assert close_resp.json()["closed"] is True
