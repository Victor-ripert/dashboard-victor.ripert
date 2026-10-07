class TradingServiceError(Exception):
    status_code = 400

    def __init__(self, message: str):
        self.message = message
        super().__init__(message)

class BrokerNotConnectedError(TradingServiceError):
    status_code = 404

class InvalidOAuthStateError(TradingServiceError):
    status_code = 400

class AlpacaAPIError(TradingServiceError):
    status_code = 502
