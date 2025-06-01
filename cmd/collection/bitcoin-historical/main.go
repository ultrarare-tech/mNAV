package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

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

func main() {
	var (
		startDate = flag.String("start", "2020-08-11", "Start date (YYYY-MM-DD)")
		endDate   = flag.String("end", "", "End date (YYYY-MM-DD), defaults to today")
		output    = flag.String("output", "data/bitcoin-prices/historical", "Output directory")
	)
	flag.Parse()

	fmt.Printf("📊 BITCOIN HISTORICAL PRICE COLLECTOR\n")
	fmt.Printf("====================================\n\n")

	// Default end date to today
	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}

	fmt.Printf("📅 Fetching Bitcoin prices from %s to %s...\n", *startDate, *endDate)

	// Fetch historical data
	histData, err := fetchHistoricalBitcoinPrices(*startDate, *endDate)
	if err != nil {
		log.Fatalf("❌ Error fetching historical data: %v", err)
	}

	fmt.Printf("✅ Fetched %d price points\n", len(histData.Data))

	// Save to file
	if err := saveHistoricalData(histData, *output); err != nil {
		log.Fatalf("❌ Error saving data: %v", err)
	}

	fmt.Printf("💾 Data saved to %s\n", *output)
}

// fetchHistoricalBitcoinPrices fetches historical Bitcoin prices
// Using CoinGecko API as it's free and doesn't require authentication for historical data
func fetchHistoricalBitcoinPrices(startDate, endDate string) (*HistoricalBitcoinData, error) {
	// Convert dates to Unix timestamps
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	// CoinGecko API endpoint for historical data
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/bitcoin/market_chart/range?vs_currency=usd&from=%d&to=%d",
		start.Unix(), end.Unix())

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Prices [][]float64 `json:"prices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Convert to our format
	histData := &HistoricalBitcoinData{
		Symbol:    "BTC",
		StartDate: startDate,
		EndDate:   endDate,
		Data:      make([]HistoricalBitcoinPrice, 0, len(response.Prices)),
		Source:    "CoinGecko",
		FetchedAt: time.Now(),
	}

	// Process daily prices
	for _, price := range response.Prices {
		if len(price) >= 2 {
			timestamp := time.Unix(int64(price[0]/1000), 0)
			histData.Data = append(histData.Data, HistoricalBitcoinPrice{
				Date:  timestamp.Format("2006-01-02"),
				Close: price[1],
				// CoinGecko only provides close prices in this endpoint
				Open: price[1],
				High: price[1],
				Low:  price[1],
			})
		}
	}

	return histData, nil
}

func saveHistoricalData(data *HistoricalBitcoinData, outputDir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create filename
	filename := fmt.Sprintf("bitcoin_historical_%s_to_%s.json",
		data.StartDate, data.EndDate)
	filepath := filepath.Join(outputDir, filename)

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}
