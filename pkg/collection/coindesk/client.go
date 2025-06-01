package coindesk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	coinGeckoCurrentEndpoint    = "https://api.coingecko.com/api/v3/simple/price"
	coinGeckoHistoricalEndpoint = "https://api.coingecko.com/api/v3/coins/bitcoin/market_chart/range"
)

// CoinGeckoMarketChartResponse represents the response from CoinGecko market chart endpoint
type CoinGeckoMarketChartResponse struct {
	Prices       [][]float64 `json:"prices"`
	MarketCaps   [][]float64 `json:"market_caps"`
	TotalVolumes [][]float64 `json:"total_volumes"`
}

// CurrentPriceResponse represents the response from CoinGecko current price endpoint
type CurrentPriceResponse struct {
	Bitcoin struct {
		USD float64 `json:"usd"`
	} `json:"bitcoin"`
}

// HistoricalBitcoinPrice represents a historical Bitcoin price point
type HistoricalBitcoinPrice struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// HistoricalBitcoinData represents the complete historical data
type HistoricalBitcoinData struct {
	Symbol    string                   `json:"symbol"`
	StartDate string                   `json:"start_date"`
	EndDate   string                   `json:"end_date"`
	Data      []HistoricalBitcoinPrice `json:"data"`
	Source    string                   `json:"source"`
	FetchedAt time.Time                `json:"fetched_at"`
}

// Client represents a crypto API client
type Client struct {
	client *http.Client
}

// NewClient creates a new crypto client
func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetCurrentPrice fetches the current Bitcoin price from CoinGecko
func (c *Client) GetCurrentPrice() (*CurrentPriceResponse, error) {
	url := fmt.Sprintf("%s?ids=bitcoin&vs_currencies=usd", coinGeckoCurrentEndpoint)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var priceResp CurrentPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &priceResp, nil
}

// GetHistoricalPrices fetches historical Bitcoin prices from CoinGecko
func (c *Client) GetHistoricalPrices(startDate, endDate string) (*HistoricalBitcoinData, error) {
	// Validate date formats
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %w", err)
	}

	// Check if date range exceeds 365 days (CoinGecko free tier limit)
	maxStart := time.Now().AddDate(0, 0, -365)
	if start.Before(maxStart) {
		adjustedStart := maxStart.Format("2006-01-02")
		fmt.Printf("⚠️  CoinGecko free tier limited to 365 days\n")
		fmt.Printf("   Adjusting start date from %s to %s\n", startDate, adjustedStart)
		startDate = adjustedStart
		start = maxStart
	}

	// Build URL with UNIX timestamps
	url := fmt.Sprintf("%s?vs_currency=usd&from=%d&to=%d",
		coinGeckoHistoricalEndpoint, start.Unix(), end.Unix())

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var marketResp CoinGeckoMarketChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&marketResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Convert to our standard format
	histData := &HistoricalBitcoinData{
		Symbol:    "BTC",
		StartDate: startDate,
		EndDate:   endDate,
		Data:      make([]HistoricalBitcoinPrice, 0, len(marketResp.Prices)),
		Source:    "CoinGecko",
		FetchedAt: time.Now(),
	}

	// Process the data (CoinGecko returns timestamps and prices)
	for i, priceData := range marketResp.Prices {
		if len(priceData) >= 2 {
			timestamp := time.Unix(int64(priceData[0]/1000), 0)
			price := priceData[1]

			// Get volume if available
			volume := 0.0
			if i < len(marketResp.TotalVolumes) && len(marketResp.TotalVolumes[i]) >= 2 {
				volume = marketResp.TotalVolumes[i][1]
			}

			histData.Data = append(histData.Data, HistoricalBitcoinPrice{
				Date:   timestamp.Format("2006-01-02"),
				Close:  price,
				Open:   price, // CoinGecko doesn't provide OHLC in this endpoint
				High:   price,
				Low:    price,
				Volume: volume,
			})
		}
	}

	return histData, nil
}

// GetHistoricalBitcoinPrices is a standalone function for compatibility
func GetHistoricalBitcoinPrices(startDate, endDate string) (*HistoricalBitcoinData, error) {
	client := NewClient()
	return client.GetHistoricalPrices(startDate, endDate)
}
