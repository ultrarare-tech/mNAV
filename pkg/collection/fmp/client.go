package fmp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Financial Modeling Prep API client
type Client struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

// NewClient creates a new FMP client
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: "https://financialmodelingprep.com/api/v3",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HistoricalPrice represents a historical price data point
type HistoricalPrice struct {
	Date             string  `json:"date"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	AdjClose         float64 `json:"adjClose"`
	Volume           int64   `json:"volume"`
	UnadjustedVolume int64   `json:"unadjustedVolume"`
	Change           float64 `json:"change"`
	ChangePercent    float64 `json:"changePercent"`
	VWAP             float64 `json:"vwap"`
	Label            string  `json:"label"`
	ChangeOverTime   float64 `json:"changeOverTime"`
}

// HistoricalData represents the response from the historical price endpoint
type HistoricalData struct {
	Symbol     string            `json:"symbol"`
	Historical []HistoricalPrice `json:"historical"`
}

// CompanyProfile represents company profile information
type CompanyProfile struct {
	Symbol            string  `json:"symbol"`
	Price             float64 `json:"price"`
	Beta              float64 `json:"beta"`
	VolAvg            int64   `json:"volAvg"`
	MktCap            int64   `json:"mktCap"`
	LastDiv           float64 `json:"lastDiv"`
	Range             string  `json:"range"`
	Changes           float64 `json:"changes"`
	CompanyName       string  `json:"companyName"`
	Currency          string  `json:"currency"`
	CIK               string  `json:"cik"`
	ISIN              string  `json:"isin"`
	CUSIP             string  `json:"cusip"`
	Exchange          string  `json:"exchange"`
	ExchangeShortName string  `json:"exchangeShortName"`
	Industry          string  `json:"industry"`
	Website           string  `json:"website"`
	Description       string  `json:"description"`
	CEO               string  `json:"ceo"`
	Sector            string  `json:"sector"`
	Country           string  `json:"country"`
	FullTimeEmployees string  `json:"fullTimeEmployees"`
	Phone             string  `json:"phone"`
	Address           string  `json:"address"`
	City              string  `json:"city"`
	State             string  `json:"state"`
	Zip               string  `json:"zip"`
	DCFDiff           float64 `json:"dcfDiff"`
	DCF               float64 `json:"dcf"`
	Image             string  `json:"image"`
	IPODate           string  `json:"ipoDate"`
	DefaultImage      bool    `json:"defaultImage"`
	IsEtf             bool    `json:"isEtf"`
	IsActivelyTrading bool    `json:"isActivelyTrading"`
	IsAdr             bool    `json:"isAdr"`
	IsFund            bool    `json:"isFund"`
}

// GetHistoricalData fetches historical price data for a symbol
func (c *Client) GetHistoricalData(symbol, startDate, endDate string) (*HistoricalData, error) {
	url := fmt.Sprintf("%s/historical-price-full/%s?from=%s&to=%s&apikey=%s",
		c.BaseURL, symbol, startDate, endDate, c.APIKey)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var data HistoricalData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &data, nil
}

// GetCompanyProfile fetches company profile including current price and market cap
func (c *Client) GetCompanyProfile(symbol string) (*CompanyProfile, error) {
	url := fmt.Sprintf("%s/profile/%s?apikey=%s", c.BaseURL, symbol, c.APIKey)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var profiles []CompanyProfile
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profile data found for symbol %s", symbol)
	}

	return &profiles[0], nil
}

// GetCurrentPrice fetches the current stock price
func (c *Client) GetCurrentPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("%s/quote-short/%s?apikey=%s", c.BaseURL, symbol, c.APIKey)

	resp, err := c.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var quotes []struct {
		Symbol string  `json:"symbol"`
		Price  float64 `json:"price"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&quotes); err != nil {
		return 0, fmt.Errorf("error parsing response: %w", err)
	}

	if len(quotes) == 0 {
		return 0, fmt.Errorf("no quote data found for symbol %s", symbol)
	}

	return quotes[0].Price, nil
}
