package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TradingAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTradingAPIClient(baseURL string) *TradingAPIClient {
	return &TradingAPIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type Position struct {
	ID       string  `json:"id"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
}

func (client *TradingAPIClient) GetPositions(
	ctx context.Context,
) ([]Position, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		client.baseURL+"/positions",
		nil,
	)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"trading api returned status %d",
			response.StatusCode,
		)
	}

	var positions []Position

	if err := json.NewDecoder(response.Body).Decode(&positions); err != nil {
		return nil, err
	}

	return positions, nil
}
