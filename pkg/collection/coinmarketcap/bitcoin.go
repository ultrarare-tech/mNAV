package coinmarketcap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	apiEndpoint = "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest"
	bitcoinID   = "1" // CoinMarketCap ID for Bitcoin
)

// BitcoinPriceResponse represents the structured response from CoinMarketCap API
type BitcoinPriceResponse struct {
	Status struct {
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	} `json:"status"`
	Data map[string]struct {
		Symbol string `json:"symbol"`
		Quote  map[string]struct {
			Price            float64   `json:"price"`
			LastUpdated      time.Time `json:"last_updated"`
			PercentChange24h float64   `json:"percent_change_24h"`
		} `json:"quote"`
	} `json:"data"`
}

// BitcoinPrice represents the Bitcoin price information
type BitcoinPrice struct {
	Price            float64   // Current price in USD
	LastUpdated      time.Time // When the price was last updated
	PercentChange24h float64   // 24-hour percent change
}

// GetBitcoinPrice fetches the current Bitcoin price from CoinMarketCap
func GetBitcoinPrice() (*BitcoinPrice, error) {
	apiKey := os.Getenv("COINMARKETCAP_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("COINMARKETCAP_API_KEY environment variable is not set")
	}

	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set query parameters
	q := req.URL.Query()
	q.Add("id", bitcoinID)
	q.Add("convert", "USD")
	req.URL.RawQuery = q.Encode()

	// Set headers
	req.Header.Set("X-CMC_PRO_API_KEY", apiKey)
	req.Header.Set("Accept", "application/json")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to CoinMarketCap API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var response BitcoinPriceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response JSON: %w", err)
	}

	// Check for API error
	if response.Status.ErrorCode != 0 {
		return nil, fmt.Errorf("API returned an error: %s", response.Status.ErrorMessage)
	}

	// Extract Bitcoin data
	btcData, exists := response.Data[bitcoinID]
	if !exists {
		return nil, fmt.Errorf("bitcoin data not found in the response")
	}

	// Extract USD quote
	usdQuote, exists := btcData.Quote["USD"]
	if !exists {
		return nil, fmt.Errorf("USD quote not found in the response")
	}

	return &BitcoinPrice{
		Price:            usdQuote.Price,
		LastUpdated:      usdQuote.LastUpdated,
		PercentChange24h: usdQuote.PercentChange24h,
	}, nil
}
