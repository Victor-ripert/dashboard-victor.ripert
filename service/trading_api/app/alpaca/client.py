from alpaca.trading.client import TradingClient

def set_trading_client(access_token: str, environment: str) -> TradingClient:
    return TradingClient(oauth_token=access_token, paper=environment == "paper")
