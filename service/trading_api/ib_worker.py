import queue
import threading
from typing import Any
from ib_service import trading_service

requests_q: queue.Queue[Any] = queue.Queue()
responses_q: queue.Queue[Any] = queue.Queue()

def _bridge_worker():
    trading_service.start()
    while True:
        try:
            req = requests_q.get(timeout=0.5)
        except queue.Empty:
            continue
        if req is None:
            break
        try:
            req_type = req.get("type")
            res = trading_service.submit_task(req_type, req.get("payload", req))
            responses_q.put(res)
        except Exception as e:
            responses_q.put({"error": str(e)})


threading.Thread(target=_bridge_worker, daemon=True, name="IBBridge").start()
