import queue
import threading
from typing import Any

from config import settings
from ib_service import trading_service

IB_HOST = settings.IB_HOST
IB_PORT = settings.IB_PORT
CLIENT_ID = settings.IB_CLIENT_ID

ib = trading_service.ib
requests_q: queue.Queue[Any] = queue.Queue()
responses_q: queue.Queue[Any] = queue.Queue()


def _legacy_bridge_worker():
    """Bridges legacy requests_q / responses_q to trading_service."""
    trading_service.start()

    while True:
        try:
            req = requests_q.get(timeout=0.5)
        except queue.Empty:
            continue

        if req is None:
            break

        req_type = req.get("type")
        try:
            if req_type == "available_funds":
                val = trading_service.submit_task("available_funds", {})
                responses_q.put(val)

            elif req_type == "positions":
                val = trading_service.submit_task("positions", {})
                responses_q.put(val)

            elif req_type == "place_order":
                val = trading_service.submit_task(
                    "place_order",
                    {
                        "symbol": req["symbol"],
                        "action": req["action"],
                        "quantity": req["qty"],
                        "product_type": "STK",
                        "order_type": "MKT",
                    },
                )
                responses_q.put({
                    "orderId": val.get("order_id", val.get("orderId")),
                    "status": val.get("status"),
                })

            elif req_type == "order_status":
                val = trading_service.submit_task("order_status", {"orderId": req["orderId"]})
                responses_q.put(val)

            elif req_type == "close_position":
                val = trading_service.submit_task("close_position", {"symbol": req["symbol"]})
                responses_q.put(val)

            elif req_type == "shutdown":
                val = trading_service.submit_task("shutdown", {})
                responses_q.put(val)
                break

            else:
                responses_q.put({"error": f"Unknown request type: {req_type}"})

        except Exception as e:
            responses_q.put({"error": str(e)})


# Start the legacy bridge thread on import
t = threading.Thread(target=_legacy_bridge_worker, daemon=True, name="LegacyIBBridge")
t.start()
