import asyncio
import concurrent.futures
import logging
import queue
import threading
import time
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional

from ib_insync import IB, Contract, Crypto, Forex, Future, LimitOrder, MarketOrder, Option, Stock, StopOrder

from config import settings
from models import (
    AccountSummary,
    CloseRequest,
    OrderAction,
    OrderType,
    PositionItem,
    ProductType,
    PurchaseRequest,
    PurchaseResponse,
    QuoteResponse,
)

logger = logging.getLogger("trading_microservice")
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")


class MockTradingEngine:
    """In-memory simulation engine for financial product purchasing and portfolio management

    when IB Gateway is not running or mock mode is enabled.
    """

    def __init__(self, initial_balance: float = 100000.0, currency: str = "EUR"):
        self.currency = currency
        self.available_funds = initial_balance
        self.cash_balance = initial_balance
        self.positions: Dict[str, Dict[str, Any]] = {}
        self.orders: Dict[str, Dict[str, Any]] = {}
        self.order_counter = 1000

        # Base mock prices for financial products
        self.product_prices: Dict[str, float] = {
            "AAPL": 185.50,
            "TSLA": 245.20,
            "MSFT": 420.10,
            "GOOGL": 175.80,
            "NVDA": 128.90,
            "AMZN": 182.40,
            "SPY": 550.00,
            "QQQ": 480.00,
            "BTC": 64500.00,
            "ETH": 3450.00,
            "SOL": 155.00,
            "EURUSD": 1.0850,
            "GBPUSD": 1.2950,
            "USDJPY": 155.20,
        }

    def get_price(self, symbol: str) -> float:
        sym = symbol.upper()
        if sym in self.product_prices:
            return self.product_prices[sym]
        # Deterministic mock price for any custom financial product symbol
        val = sum(ord(c) for c in sym) % 300 + 25.5
        return round(float(val), 2)

    def get_quote(self, symbol: str, product_type: str = "STK", currency: str = "USD") -> Dict[str, Any]:
        price = self.get_price(symbol)
        spread = round(price * 0.001, 2) if price > 1 else 0.0002
        return {
            "symbol": symbol.upper(),
            "product_type": product_type.upper(),
            "currency": currency.upper(),
            "last_price": price,
            "bid": round(price - spread / 2, 2),
            "ask": round(price + spread / 2, 2),
            "high": round(price * 1.015, 2),
            "low": round(price * 0.985, 2),
            "volume": 1250000.0,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

    def get_account_summary(self) -> Dict[str, Any]:
        unrealized_pnl = 0.0
        portfolio_val = 0.0
        for sym, pos in self.positions.items():
            curr_price = self.get_price(sym)
            market_val = pos["position"] * curr_price
            pnl = (curr_price - pos["avg_cost"]) * pos["position"]
            portfolio_val += market_val
            unrealized_pnl += pnl

        net_liq = self.cash_balance + portfolio_val
        return {
            "currency": self.currency,
            "available_funds": round(self.available_funds, 2),
            "buying_power": round(self.available_funds * 2, 2),
            "net_liquidation": round(net_liq, 2),
            "cash_balance": round(self.cash_balance, 2),
            "mode": "MOCK",
        }

    def get_positions(self) -> List[Dict[str, Any]]:
        result = []
        for sym, p in self.positions.items():
            if p["position"] == 0:
                continue
            curr_price = self.get_price(sym)
            market_val = p["position"] * curr_price
            pnl = (curr_price - p["avg_cost"]) * p["position"]
            result.append({
                "account": "MOCK-ACC-101",
                "symbol": sym,
                "sec_type": p["sec_type"],
                "currency": p["currency"],
                "position": float(p["position"]),
                "avg_cost": round(float(p["avg_cost"]), 4),
                "market_price": round(float(curr_price), 4),
                "market_value": round(float(market_val), 2),
                "unrealized_pnl": round(float(pnl), 2),
            })
        return result

    def execute_order(
        self,
        symbol: str,
        action: str,
        quantity: float,
        product_type: str = "STK",
        order_type: str = "MKT",
        price: Optional[float] = None,
        currency: str = "USD",
        exchange: Optional[str] = None,
    ) -> Dict[str, Any]:
        self.order_counter += 1
        order_id = self.order_counter
        curr_price = price if (order_type == "LMT" and price is not None) else self.get_price(symbol)
        total_cost = quantity * curr_price

        action_upper = action.upper()
        sym = symbol.upper()

        if action_upper == "BUY":
            if total_cost > self.available_funds:
                raise ValueError(
                    f"Insufficient funds to purchase {quantity} of {sym}. "
                    f"Required: {total_cost:.2f} {currency}, Available: {self.available_funds:.2f} {self.currency}"
                )
            self.available_funds -= total_cost
            self.cash_balance -= total_cost

            # Update position
            existing = self.positions.get(sym, {"position": 0.0, "avg_cost": 0.0})
            old_qty = existing["position"]
            old_cost = existing["avg_cost"]
            new_qty = old_qty + quantity
            new_avg_cost = ((old_qty * old_cost) + (quantity * curr_price)) / new_qty

            self.positions[sym] = {
                "position": new_qty,
                "avg_cost": new_avg_cost,
                "sec_type": product_type,
                "currency": currency,
            }

        elif action_upper == "SELL":
            existing = self.positions.get(sym, {"position": 0.0, "avg_cost": 0.0})
            if existing["position"] < quantity:
                raise ValueError(
                    f"Cannot sell {quantity} of {sym}: current position is only {existing['position']}"
                )
            self.available_funds += total_cost
            self.cash_balance += total_cost

            new_qty = existing["position"] - quantity
            if new_qty == 0:
                self.positions.pop(sym, None)
            else:
                self.positions[sym]["position"] = new_qty

        order_record = {
            "order_id": order_id,
            "symbol": sym,
            "product_type": product_type,
            "action": action_upper,
            "quantity": float(quantity),
            "order_type": order_type,
            "price": curr_price,
            "status": "FILLED",
            "filled_quantity": float(quantity),
            "avg_fill_price": curr_price,
            "total_cost": round(total_cost, 2),
            "currency": currency,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "message": f"Successfully executed simulated {action_upper} for {quantity} {sym} @ {curr_price:.2f} {currency}",
        }
        self.orders[str(order_id)] = order_record
        return order_record

    def close_position(self, symbol: str) -> Dict[str, Any]:
        sym = symbol.upper()
        existing = self.positions.get(sym)
        if not existing or existing["position"] == 0:
            return {"symbol": sym, "closed": False, "reason": "no position"}

        qty = existing["position"]
        action = "SELL" if qty > 0 else "BUY"
        res = self.execute_order(
            symbol=sym,
            action=action,
            quantity=abs(qty),
            product_type=existing.get("sec_type", "STK"),
            order_type="MKT",
            currency=existing.get("currency", "USD"),
        )
        return {
            "symbol": sym,
            "closed": True,
            "finalStatus": "FILLED",
            "filled": abs(qty),
            "message": f"Position for {sym} fully closed.",
        }

    def get_order_status(self, order_id: int | str) -> Dict[str, Any]:
        order = self.orders.get(str(order_id))
        if not order:
            return {"orderId": order_id, "found": False}
        return {
            "orderId": order_id,
            "found": True,
            "status": order["status"],
            "filled": order["filled_quantity"],
            "remaining": 0.0,
            "avgFillPrice": order.get("avg_fill_price"),
            "symbol": order.get("symbol"),
        }


class IBTradingService:
    """Thread-safe trading service that encapsulates communication with Interactive Brokers

    (TWS / IB Gateway) and falls back to a Mock Trading Engine when IB is not running.
    """

    def __init__(self):
        self.ib = IB()
        self.mock_engine = MockTradingEngine()
        self.is_connected = False
        self.use_mock = settings.MOCK_MODE
        self._task_queue: queue.Queue = queue.Queue()
        self._worker_thread: Optional[threading.Thread] = None
        self._stop_event = threading.Event()
        self._loop: Optional[asyncio.AbstractEventLoop] = None

    def start(self):
        """Starts the dedicated background worker thread and attempts initial connection."""
        if self._worker_thread and self._worker_thread.is_alive():
            return
        self._stop_event.clear()
        self._worker_thread = threading.Thread(target=self._run_worker, daemon=True, name="IBTradingWorker")
        self._worker_thread.start()
        logger.info("IBTradingService worker thread started.")

    def _attempt_connect(self) -> bool:
        """Attempts connection to Interactive Brokers Gateway / TWS."""
        try:
            logger.info(
                f"Attempting connection to Interactive Brokers at {settings.IB_HOST}:{settings.IB_PORT} (client_id={settings.IB_CLIENT_ID})..."
            )
            self.ib.connect(
                settings.IB_HOST,
                settings.IB_PORT,
                clientId=settings.IB_CLIENT_ID,
                timeout=min(settings.IB_TIMEOUT, 3.0),
            )
            self.is_connected = self.ib.isConnected()
            if self.is_connected:
                self.use_mock = False
                logger.info(f"Connected to Interactive Brokers! Server version: {self.ib.client.serverVersion()}")
                return True
        except Exception as e:
            logger.info(f"Interactive Brokers not ready yet ({e}).")
            if settings.AUTO_FALLBACK_MOCK or settings.MOCK_MODE:
                self.use_mock = True
            else:
                self.is_connected = False
        return False

    def _run_worker(self):
        """Background worker thread event loop."""
        self._loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self._loop)

        if not self.use_mock:
            self._attempt_connect()

        last_retry_time = time.time()

        while not self._stop_event.is_set():
            # Periodically retry connection to IB Gateway if it was starting up
            if not self.is_connected and not settings.MOCK_MODE:
                if time.time() - last_retry_time > 5.0:
                    last_retry_time = time.time()
                    self._attempt_connect()

            try:
                task = self._task_queue.get(timeout=0.2)
            except queue.Empty:
                if self.is_connected:
                    self.ib.sleep(0.05)
                continue

            if task is None:
                break

            action, payload, future = task
            try:
                result = self._dispatch_action(action, payload)
                future.set_result(result)
            except Exception as ex:
                logger.exception(f"Error handling action '{action}': {ex}")
                future.set_exception(ex)
            finally:
                self._task_queue.task_done()

        # Clean disconnect
        if self.is_connected:
            try:
                self.ib.disconnect()
            except Exception:
                pass
            self.is_connected = False

        logger.info("IBTradingService worker loop finished.")

    def submit_task(self, action: str, payload: Dict[str, Any], timeout: Optional[float] = None) -> Any:
        """Thread-safe request dispatcher using Futures."""
        if not self._worker_thread or not self._worker_thread.is_alive():
            self.start()
            time.sleep(0.1)

        effective_timeout = timeout or settings.IB_TIMEOUT
        fut = concurrent.futures.Future()
        self._task_queue.put((action, payload, fut))

        try:
            return fut.result(timeout=effective_timeout)
        except concurrent.futures.TimeoutError:
            raise TimeoutError(f"Operation '{action}' timed out after {effective_timeout}s")

    def _build_contract(
        self,
        symbol: str,
        product_type: str = "STK",
        currency: str = "USD",
        exchange: Optional[str] = None,
    ) -> Contract:
        ptype = product_type.upper()
        sym = symbol.upper()
        curr = currency.upper()

        if ptype in ("STK", "ETF"):
            exch = exchange or "SMART"
            return Stock(sym, exch, curr)
        elif ptype == "CRYPTO":
            exch = exchange or "PAXOS"
            return Crypto(sym, exch, curr)
        elif ptype == "CASH":
            exch = exchange or "IDEALPRO"
            if len(sym) == 6 and not currency:
                return Forex(sym)
            return Contract(secType="CASH", symbol=sym, currency=curr, exchange=exch)
        else:
            exch = exchange or "SMART"
            return Contract(secType=ptype, symbol=sym, currency=curr, exchange=exch)

    def _dispatch_action(self, action: str, payload: Dict[str, Any]) -> Any:
        """Executes operations on either IB or Mock engine."""
        if self.use_mock:
            return self._dispatch_mock(action, payload)
        else:
            return self._dispatch_live(action, payload)

    def _dispatch_mock(self, action: str, payload: Dict[str, Any]) -> Any:
        if action == "available_funds":
            summary = self.mock_engine.get_account_summary()
            return summary["available_funds"]

        elif action == "account_summary":
            return self.mock_engine.get_account_summary()

        elif action == "positions":
            return self.mock_engine.get_positions()

        elif action in ("place_order", "purchase"):
            return self.mock_engine.execute_order(
                symbol=payload["symbol"],
                action=payload["action"],
                quantity=payload["quantity"],
                product_type=payload.get("product_type", "STK"),
                order_type=payload.get("order_type", "MKT"),
                price=payload.get("price"),
                currency=payload.get("currency", "USD"),
                exchange=payload.get("exchange"),
            )

        elif action == "order_status":
            return self.mock_engine.get_order_status(payload["orderId"])

        elif action == "close_position":
            return self.mock_engine.close_position(payload["symbol"])

        elif action == "quote":
            return self.mock_engine.get_quote(
                symbol=payload["symbol"],
                product_type=payload.get("product_type", "STK"),
                currency=payload.get("currency", "USD"),
            )

        elif action == "shutdown":
            return "mock_shutdown"

        raise ValueError(f"Unknown action: {action}")

    def _dispatch_live(self, action: str, payload: Dict[str, Any]) -> Any:
        if not self.ib.isConnected():
            raise ConnectionError("Not connected to Interactive Brokers Gateway/TWS")

        if action == "available_funds":
            vals = self.ib.accountValues()
            val = next(
                (float(v.value) for v in vals if v.tag == "AvailableFunds" and v.currency in ("EUR", "USD")),
                None,
            )
            return val

        elif action == "account_summary":
            vals = self.ib.accountValues()
            curr = "EUR"
            avail = 0.0
            buying_power = 0.0
            net_liq = 0.0
            cash = 0.0

            for v in vals:
                if v.tag == "AvailableFunds":
                    avail = float(v.value)
                    curr = v.currency
                elif v.tag == "BuyingPower":
                    buying_power = float(v.value)
                elif v.tag == "NetLiquidation":
                    net_liq = float(v.value)
                elif v.tag == "TotalCashBalance":
                    cash = float(v.value)

            return {
                "currency": curr,
                "available_funds": avail,
                "buying_power": buying_power,
                "net_liquidation": net_liq,
                "cash_balance": cash,
                "mode": "LIVE",
            }

        elif action == "positions":
            poss = self.ib.positions()
            data = []
            for p in poss:
                data.append({
                    "account": p.account,
                    "symbol": p.contract.symbol,
                    "sec_type": p.contract.secType,
                    "currency": p.contract.currency,
                    "position": float(p.position),
                    "avg_cost": float(p.avgCost),
                    "market_price": None,
                    "market_value": round(float(p.position) * float(p.avgCost), 2),
                    "unrealized_pnl": 0.0,
                })
            return data

        elif action in ("place_order", "purchase"):
            symbol = payload["symbol"]
            action_type = payload["action"].upper()
            qty = payload["quantity"]
            ptype = payload.get("product_type", "STK")
            order_type = payload.get("order_type", "MKT").upper()
            price = payload.get("price")
            currency = payload.get("currency", "USD")
            exchange = payload.get("exchange")

            contract = self._build_contract(symbol, ptype, currency, exchange)
            self.ib.qualifyContracts(contract)

            if order_type == "LMT":
                if price is None:
                    raise ValueError("Price is required for Limit Order")
                order = LimitOrder(action_type, qty, price)
            elif order_type == "STP":
                if price is None:
                    raise ValueError("Price is required for Stop Order")
                order = StopOrder(action_type, qty, price)
            else:
                order = MarketOrder(action_type, qty)

            trade = self.ib.placeOrder(contract, order)
            self.ib.sleep(0.5)

            return {
                "order_id": trade.order.orderId,
                "symbol": symbol,
                "product_type": ptype,
                "action": action_type,
                "quantity": float(qty),
                "order_type": order_type,
                "price": price,
                "status": trade.orderStatus.status,
                "filled_quantity": float(trade.orderStatus.filled),
                "avg_fill_price": float(trade.orderStatus.avgFillPrice) if trade.orderStatus.avgFillPrice else None,
                "total_cost": None,
                "currency": currency,
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "message": f"Order placed with status: {trade.orderStatus.status}",
            }

        elif action == "order_status":
            order_id = payload["orderId"]
            target = None
            for tr in self.ib.trades():
                if tr.order.orderId == order_id:
                    target = tr
                    break

            if target is None:
                return {"orderId": order_id, "found": False}
            return {
                "orderId": order_id,
                "found": True,
                "status": target.orderStatus.status,
                "filled": float(target.orderStatus.filled),
                "remaining": float(target.orderStatus.remaining),
                "avgFillPrice": float(target.orderStatus.avgFillPrice),
                "symbol": target.contract.symbol,
            }

        elif action == "close_position":
            symbol = payload["symbol"]
            target = None
            for p in self.ib.positions():
                if p.contract.symbol.upper() == symbol.upper():
                    target = p
                    break

            if target is None or target.position == 0:
                return {"symbol": symbol, "closed": False, "reason": "no position"}
            else:
                qty = abs(target.position)
                act = "SELL" if target.position > 0 else "BUY"
                order = MarketOrder(act, qty)
                trade = self.ib.placeOrder(target.contract, order)

                # Wait for execution with 5s timeout
                start_wait = time.time()
                while not trade.isDone() and (time.time() - start_wait < 5.0):
                    self.ib.sleep(0.5)

                return {
                    "symbol": symbol,
                    "closed": trade.isDone(),
                    "finalStatus": trade.orderStatus.status,
                    "filled": float(trade.orderStatus.filled),
                    "message": f"Position close order status: {trade.orderStatus.status}",
                }

        elif action == "quote":
            symbol = payload["symbol"]
            ptype = payload.get("product_type", "STK")
            currency = payload.get("currency", "USD")
            contract = self._build_contract(symbol, ptype, currency)
            self.ib.qualifyContracts(contract)

            tickers = self.ib.reqTickers(contract)
            if tickers:
                t = tickers[0]
                return {
                    "symbol": symbol.upper(),
                    "product_type": ptype,
                    "currency": currency,
                    "last_price": t.last if t.last == t.last else None,
                    "bid": t.bid if t.bid == t.bid else None,
                    "ask": t.ask if t.ask == t.ask else None,
                    "high": t.high if t.high == t.high else None,
                    "low": t.low if t.low == t.low else None,
                    "volume": float(t.volume) if t.volume == t.volume else None,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }
            return self.mock_engine.get_quote(symbol, ptype, currency)

        elif action == "shutdown":
            self.ib.disconnect()
            self.is_connected = False
            return "disconnected"

        raise ValueError(f"Unknown action: {action}")

    def reconnect(self) -> Dict[str, Any]:
        """Manually triggers a reconnection attempt to Interactive Brokers."""
        try:
            if self.ib.isConnected():
                self.ib.disconnect()
            self.ib.connect(
                settings.IB_HOST,
                settings.IB_PORT,
                clientId=settings.IB_CLIENT_ID,
                timeout=settings.IB_TIMEOUT,
            )
            self.is_connected = self.ib.isConnected()
            self.use_mock = False
            return {"connected": True, "mode": "LIVE", "server_version": self.ib.client.serverVersion()}
        except Exception as e:
            return {"connected": False, "mode": "MOCK" if self.use_mock else "DISCONNECTED", "error": str(e)}

    def stop(self):
        """Clean shutdown of service and worker thread."""
        self._stop_event.set()
        self._task_queue.put(None)
        if self._worker_thread:
            self._worker_thread.join(timeout=2.0)
            self._worker_thread = None


# Singleton instance
trading_service = IBTradingService()
