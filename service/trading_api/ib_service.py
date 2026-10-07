import asyncio
import concurrent.futures
import logging
import queue
import threading
import time
from datetime import datetime, timezone
from typing import Any

from ib_insync import IB, Contract, Crypto, Forex, LimitOrder, MarketOrder, Stock, StopOrder
from config import settings

logger = logging.getLogger("trading_microservice")


class MockTradingEngine:
    def __init__(self, initial_balance: float = 100000.0, currency: str = "EUR"):
        self.currency = currency
        self.available_funds = initial_balance
        self.cash_balance = initial_balance
        self.positions: dict[str, dict[str, Any]] = {}
        self.orders: dict[str, dict[str, Any]] = {}
        self.order_counter = 1000
        self.prices: dict[str, float] = {
            "AAPL": 185.50, "TSLA": 245.20, "MSFT": 420.10, "NVDA": 128.90,
            "BTC": 64500.00, "ETH": 3450.00, "EURUSD": 1.0850,
        }

    def get_price(self, symbol: str) -> float:
        sym = symbol.upper()
        if sym in self.prices:
            return self.prices[sym]
        return round(float(sum(ord(c) for c in sym) % 300 + 25.5), 2)

    def get_quote(self, symbol: str, product_type: str = "STK", currency: str = "USD") -> dict[str, Any]:
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

    def get_account_summary(self) -> dict[str, Any]:
        portfolio_val = sum(pos["position"] * self.get_price(sym) for sym, pos in self.positions.items())
        return {
            "currency": self.currency,
            "available_funds": round(self.available_funds, 2),
            "buying_power": round(self.available_funds * 2, 2),
            "net_liquidation": round(self.cash_balance + portfolio_val, 2),
            "cash_balance": round(self.cash_balance, 2),
            "mode": "MOCK",
        }

    def get_positions(self) -> list[dict[str, Any]]:
        results = []
        for sym, p in self.positions.items():
            if p["position"] == 0:
                continue
            curr_price = self.get_price(sym)
            market_val = p["position"] * curr_price
            pnl = (curr_price - p["avg_cost"]) * p["position"]
            results.append({
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
        return results

    def execute_order(self, symbol: str, action: str, quantity: float, product_type: str = "STK",
                      order_type: str = "MKT", price: float | None = None, currency: str = "USD",
                      exchange: str | None = None) -> dict[str, Any]:
        self.order_counter += 1
        order_id = self.order_counter
        curr_price = price if (order_type == "LMT" and price is not None) else self.get_price(symbol)
        total_cost = quantity * curr_price
        sym = symbol.upper()
        act = action.upper()

        if act == "BUY":
            if total_cost > self.available_funds:
                raise ValueError(f"Insufficient funds: required {total_cost:.2f} {currency}, available {self.available_funds:.2f} {self.currency}")
            self.available_funds -= total_cost
            self.cash_balance -= total_cost
            existing = self.positions.get(sym, {"position": 0.0, "avg_cost": 0.0})
            new_qty = existing["position"] + quantity
            new_avg = ((existing["position"] * existing["avg_cost"]) + total_cost) / new_qty
            self.positions[sym] = {"position": new_qty, "avg_cost": new_avg, "sec_type": product_type, "currency": currency}
        elif act == "SELL":
            existing = self.positions.get(sym, {"position": 0.0, "avg_cost": 0.0})
            if existing["position"] < quantity:
                raise ValueError(f"Cannot sell {quantity} of {sym}: current position is {existing['position']}")
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
            "action": act,
            "quantity": float(quantity),
            "order_type": order_type,
            "price": curr_price,
            "status": "FILLED",
            "filled_quantity": float(quantity),
            "avg_fill_price": curr_price,
            "total_cost": round(total_cost, 2),
            "currency": currency,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "message": f"Simulated {act} {quantity} {sym} @ {curr_price:.2f} {currency}",
        }
        self.orders[str(order_id)] = order_record
        return order_record

    def close_position(self, symbol: str) -> dict[str, Any]:
        sym = symbol.upper()
        pos = self.positions.get(sym)
        if not pos or pos["position"] == 0:
            return {"symbol": sym, "closed": False, "reason": "no position"}
        qty = pos["position"]
        act = "SELL" if qty > 0 else "BUY"
        self.execute_order(sym, act, abs(qty), pos.get("sec_type", "STK"), "MKT", currency=pos.get("currency", "USD"))
        return {"symbol": sym, "closed": True, "finalStatus": "FILLED", "filled": abs(qty)}

    def get_order_status(self, order_id: int | str) -> dict[str, Any]:
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
    def __init__(self):
        self.ib = IB()
        self.mock_engine = MockTradingEngine()
        self.is_connected = False
        self.use_mock = settings.MOCK_MODE
        self._task_queue: queue.Queue = queue.Queue()
        self._worker_thread: threading.Thread | None = None
        self._stop_event = threading.Event()

    def start(self):
        if self._worker_thread and self._worker_thread.is_alive():
            return
        self._stop_event.clear()
        self._worker_thread = threading.Thread(target=self._run_worker, daemon=True, name="IBWorker")
        self._worker_thread.start()

    def _attempt_connect(self) -> bool:
        try:
            self.ib.connect(settings.IB_HOST, settings.IB_PORT, clientId=settings.IB_CLIENT_ID, timeout=min(settings.IB_TIMEOUT, 3.0))
            self.is_connected = self.ib.isConnected()
            if self.is_connected:
                self.use_mock = False
                return True
        except Exception:
            if settings.AUTO_FALLBACK_MOCK or settings.MOCK_MODE:
                self.use_mock = True
        return False

    def _run_worker(self):
        asyncio.set_event_loop(asyncio.new_event_loop())
        if not self.use_mock:
            self._attempt_connect()

        last_retry = time.time()
        while not self._stop_event.is_set():
            if not self.is_connected and not self.use_mock and time.time() - last_retry > 5.0:
                last_retry = time.time()
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
                future.set_exception(ex)
            finally:
                self._task_queue.task_done()

        if self.is_connected:
            try:
                self.ib.disconnect()
            except Exception:
                pass
            self.is_connected = False

    def submit_task(self, action: str, payload: dict[str, Any], timeout: float | None = None) -> Any:
        if self.use_mock:
            return self._dispatch_mock(action, payload)

        if not self._worker_thread or not self._worker_thread.is_alive():
            self.start()

        fut = concurrent.futures.Future()
        self._task_queue.put((action, payload, fut))
        return fut.result(timeout=timeout or settings.IB_TIMEOUT)

    def _build_contract(self, symbol: str, product_type: str = "STK", currency: str = "USD", exchange: str | None = None) -> Contract:
        ptype = product_type.upper()
        sym = symbol.upper()
        curr = currency.upper()
        exch = exchange or ("PAXOS" if ptype == "CRYPTO" else ("IDEALPRO" if ptype == "CASH" else "SMART"))

        if ptype in ("STK", "ETF"):
            return Stock(sym, exch, curr)
        if ptype == "CRYPTO":
            return Crypto(sym, exch, curr)
        if ptype == "CASH":
            return Forex(sym) if len(sym) == 6 else Contract(secType="CASH", symbol=sym, currency=curr, exchange=exch)
        return Contract(secType=ptype, symbol=sym, currency=curr, exchange=exch)

    def _dispatch_action(self, action: str, payload: dict[str, Any]) -> Any:
        return self._dispatch_mock(action, payload) if self.use_mock else self._dispatch_live(action, payload)

    def _dispatch_mock(self, action: str, payload: dict[str, Any]) -> Any:
        if action == "available_funds":
            return self.mock_engine.get_account_summary()["available_funds"]
        if action == "account_summary":
            return self.mock_engine.get_account_summary()
        if action == "positions":
            return self.mock_engine.get_positions()
        if action in ("place_order", "purchase"):
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
        if action == "order_status":
            return self.mock_engine.get_order_status(payload["orderId"])
        if action == "close_position":
            return self.mock_engine.close_position(payload["symbol"])
        if action == "quote":
            return self.mock_engine.get_quote(payload["symbol"], payload.get("product_type", "STK"), payload.get("currency", "USD"))
        if action == "shutdown":
            return "mock_shutdown"
        raise ValueError(f"Unknown action: {action}")

    def _dispatch_live(self, action: str, payload: dict[str, Any]) -> Any:
        if not self.ib.isConnected():
            raise ConnectionError("Not connected to Interactive Brokers Gateway/TWS")

        if action == "available_funds":
            vals = self.ib.accountValues()
            return next((float(v.value) for v in vals if v.tag == "AvailableFunds" and v.currency in ("EUR", "USD")), 0.0)

        if action == "account_summary":
            vals = {v.tag: float(v.value) for v in self.ib.accountValues() if v.value.replace('.', '', 1).isdigit()}
            return {
                "currency": "EUR",
                "available_funds": vals.get("AvailableFunds", 0.0),
                "buying_power": vals.get("BuyingPower", 0.0),
                "net_liquidation": vals.get("NetLiquidation", 0.0),
                "cash_balance": vals.get("TotalCashBalance", 0.0),
                "mode": "LIVE",
            }

        if action == "positions":
            return [{
                "account": p.account,
                "symbol": p.contract.symbol,
                "sec_type": p.contract.secType,
                "currency": p.contract.currency,
                "position": float(p.position),
                "avg_cost": float(p.avgCost),
                "market_price": None,
                "market_value": round(float(p.position) * float(p.avgCost), 2),
                "unrealized_pnl": 0.0,
            } for p in self.ib.positions()]

        if action in ("place_order", "purchase"):
            contract = self._build_contract(payload["symbol"], payload.get("product_type", "STK"), payload.get("currency", "USD"), payload.get("exchange"))
            self.ib.qualifyContracts(contract)
            act, qty = payload["action"].upper(), payload["quantity"]
            otype = payload.get("order_type", "MKT").upper()
            price = payload.get("price")

            if otype == "LMT":
                order = LimitOrder(act, qty, price)
            elif otype == "STP":
                order = StopOrder(act, qty, price)
            else:
                order = MarketOrder(act, qty)

            trade = self.ib.placeOrder(contract, order)
            self.ib.sleep(0.5)
            status = trade.orderStatus
            return {
                "order_id": trade.order.orderId,
                "symbol": payload["symbol"],
                "product_type": payload.get("product_type", "STK"),
                "action": act,
                "quantity": float(qty),
                "order_type": otype,
                "price": price,
                "status": status.status,
                "filled_quantity": float(status.filled),
                "avg_fill_price": float(status.avgFillPrice) if status.avgFillPrice else None,
                "total_cost": None,
                "currency": payload.get("currency", "USD"),
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "message": f"Order placed: {status.status}",
            }

        if action == "order_status":
            trade = next((t for t in self.ib.trades() if t.order.orderId == payload["orderId"]), None)
            if not trade:
                return {"orderId": payload["orderId"], "found": False}
            return {
                "orderId": payload["orderId"],
                "found": True,
                "status": trade.orderStatus.status,
                "filled": float(trade.orderStatus.filled),
                "remaining": float(trade.orderStatus.remaining),
                "avgFillPrice": float(trade.orderStatus.avgFillPrice),
                "symbol": trade.contract.symbol,
            }

        if action == "close_position":
            target = next((p for p in self.ib.positions() if p.contract.symbol.upper() == payload["symbol"].upper()), None)
            if not target or target.position == 0:
                return {"symbol": payload["symbol"], "closed": False, "reason": "no position"}
            order = MarketOrder("SELL" if target.position > 0 else "BUY", abs(target.position))
            trade = self.ib.placeOrder(target.contract, order)
            start_wait = time.time()
            while not trade.isDone() and (time.time() - start_wait < 5.0):
                self.ib.sleep(0.5)
            return {"symbol": payload["symbol"], "closed": trade.isDone(), "finalStatus": trade.orderStatus.status, "filled": float(trade.orderStatus.filled)}

        if action == "quote":
            contract = self._build_contract(payload["symbol"], payload.get("product_type", "STK"), payload.get("currency", "USD"))
            self.ib.qualifyContracts(contract)
            tickers = self.ib.reqTickers(contract)
            if tickers:
                t = tickers[0]
                return {
                    "symbol": payload["symbol"].upper(),
                    "product_type": payload.get("product_type", "STK"),
                    "currency": payload.get("currency", "USD"),
                    "last_price": t.last if t.last == t.last else None,
                    "bid": t.bid if t.bid == t.bid else None,
                    "ask": t.ask if t.ask == t.ask else None,
                    "high": t.high if t.high == t.high else None,
                    "low": t.low if t.low == t.low else None,
                    "volume": float(t.volume) if t.volume == t.volume else None,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }
            return self.mock_engine.get_quote(payload["symbol"], payload.get("product_type", "STK"), payload.get("currency", "USD"))

        if action == "shutdown":
            self.ib.disconnect()
            self.is_connected = False
            return "disconnected"

        raise ValueError(f"Unknown action: {action}")

    def reconnect(self) -> dict[str, Any]:
        try:
            if self.ib.isConnected():
                self.ib.disconnect()
            self.ib.connect(settings.IB_HOST, settings.IB_PORT, clientId=settings.IB_CLIENT_ID, timeout=settings.IB_TIMEOUT)
            self.is_connected = self.ib.isConnected()
            self.use_mock = False
            return {"connected": True, "mode": "LIVE", "server_version": self.ib.client.serverVersion()}
        except Exception as e:
            return {"connected": False, "mode": "MOCK" if self.use_mock else "DISCONNECTED", "error": str(e)}

    def stop(self):
        self._stop_event.set()
        self._task_queue.put(None)
        if self._worker_thread:
            self._worker_thread.join(timeout=2.0)
            self._worker_thread = None


trading_service = IBTradingService()
