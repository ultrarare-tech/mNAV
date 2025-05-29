package yahoo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HistoricalDataPoint represents a single day of market data
type HistoricalDataPoint struct {
	Date     string  `json:"date"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   int64   `json:"volume"`
	AdjClose float64 `json:"adjClose"`
}

// HistoricalData represents historical stock data for a symbol
type HistoricalData struct {
	Symbol string                `json:"symbol"`
	Data   []HistoricalDataPoint `json:"data"`
}

// YahooHistoricalResponse represents the response from Yahoo Finance historical data API
type YahooHistoricalResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol               string  `json:"symbol"`
				Currency             string  `json:"currency"`
				ExchangeName         string  `json:"exchangeName"`
				InstrumentType       string  `json:"instrumentType"`
				FirstTradeDate       int64   `json:"firstTradeDate"`
				RegularMarketTime    int64   `json:"regularMarketTime"`
				Timezone             string  `json:"timezone"`
				ExchangeTimezoneName string  `json:"exchangeTimezoneName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				PreviousClose        float64 `json:"previousClose"`
				Scale                int     `json:"scale"`
				PriceHint            int     `json:"priceHint"`
				CurrentTradingPeriod struct {
					Pre struct {
						Timezone  string `json:"timezone"`
						Start     int64  `json:"start"`
						End       int64  `json:"end"`
						Gmtoffset int    `json:"gmtoffset"`
					} `json:"pre"`
					Regular struct {
						Timezone  string `json:"timezone"`
						Start     int64  `json:"start"`
						End       int64  `json:"end"`
						Gmtoffset int    `json:"gmtoffset"`
					} `json:"regular"`
					Post struct {
						Timezone  string `json:"timezone"`
						Start     int64  `json:"start"`
						End       int64  `json:"end"`
						Gmtoffset int    `json:"gmtoffset"`
					} `json:"post"`
				} `json:"currentTradingPeriod"`
				DataGranularity string `json:"dataGranularity"`
				Range           string `json:"range"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
				Adjclose []struct {
					Adjclose []float64 `json:"adjclose"`
				} `json:"adjclose"`
			} `json:"indicators"`
		} `json:"result"`
		Error string `json:"error"`
	} `json:"chart"`
}

// GetHistoricalData fetches historical data for a symbol
// rangeStr should be one of: "1d", "5d", "1mo", "3mo", "6mo", "1y", "2y", "5y", "10y", "ytd", "max"
// or it can be a custom range with start and end dates: "startUnix,endUnix" in Unix timestamp format
func GetHistoricalData(symbol string, rangeStr string) (*HistoricalData, error) {
	url := fmt.Sprintf("%s/%s", apiEndpoint, symbol)

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set query parameters
	q := req.URL.Query()
	q.Add("interval", "1d") // Daily interval

	// Check if rangeStr is a custom range (contains comma)
	if strings.Contains(rangeStr, ",") {
		// Parse the start and end timestamps
		timestamps := strings.Split(rangeStr, ",")
		if len(timestamps) == 2 {
			q.Add("period1", timestamps[0])
			q.Add("period2", timestamps[1])
		} else {
			return nil, fmt.Errorf("invalid custom range format: %s", rangeStr)
		}
	} else {
		// Use predefined range
		q.Add("range", rangeStr)
	}

	q.Add("includePrePost", "false") // Exclude pre and post market data
	q.Add("events", "div,split")     // Include dividends and splits
	req.URL.RawQuery = q.Encode()

	// Set a browser-like user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second, // Longer timeout for historical data
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Parse response
	var response YahooHistoricalResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response JSON: %w", err)
	}

	// Check for API error
	if response.Chart.Error != "" {
		return nil, fmt.Errorf("API error: %s", response.Chart.Error)
	}

	// Check if we have results
	if len(response.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned for symbol %s", symbol)
	}

	// Extract data
	result := response.Chart.Result[0]
	timestamps := result.Timestamp

	// Ensure we have quotes data
	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data returned for symbol %s", symbol)
	}
	quotes := result.Indicators.Quote[0]

	// Ensure we have adjusted close data
	var adjcloses []float64
	if len(result.Indicators.Adjclose) > 0 {
		adjcloses = result.Indicators.Adjclose[0].Adjclose
	} else {
		// If no adjusted close, use regular close
		adjcloses = quotes.Close
	}

	// Create historical data points
	histData := &HistoricalData{
		Symbol: symbol,
		Data:   make([]HistoricalDataPoint, 0, len(timestamps)),
	}

	// Process each data point
	for i, ts := range timestamps {
		// Skip if we don't have data for this timestamp (null values in Yahoo's response)
		if i >= len(quotes.Open) || i >= len(quotes.High) || i >= len(quotes.Low) ||
			i >= len(quotes.Close) || i >= len(quotes.Volume) ||
			(len(adjcloses) > 0 && i >= len(adjcloses)) {
			continue
		}

		// Handle null values (represented as NaN in Go's floating-point)
		open := quotes.Open[i]
		high := quotes.High[i]
		low := quotes.Low[i]
		close := quotes.Close[i]
		volume := quotes.Volume[i]

		// Set adjClose to close if adjcloses is empty
		adjClose := close
		if len(adjcloses) > i {
			adjClose = adjcloses[i]
		}

		// Convert timestamp to date string
		date := time.Unix(ts, 0).Format("2006-01-02")

		// Add data point
		histData.Data = append(histData.Data, HistoricalDataPoint{
			Date:     date,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
			Volume:   volume,
			AdjClose: adjClose,
		})
	}

	return histData, nil
}

// SaveHistoricalDataToFile saves historical data to a JSON file
func SaveHistoricalDataToFile(data *HistoricalData, directory string) (string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", fmt.Errorf("error creating directory: %w", err)
	}

	// Create filename with timestamp
	filename := fmt.Sprintf("%s_historical_data_%s.json",
		data.Symbol,
		time.Now().Format("2006-01-02"),
	)
	filepath := filepath.Join(directory, filename)

	// Convert data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling data to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("error writing data to file: %w", err)
	}

	return filepath, nil
}
