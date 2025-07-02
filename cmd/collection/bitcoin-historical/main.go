package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/collection/coindesk"
)

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
	fmt.Printf("🔗 Data source: CoinGecko (Free API)\n")
	fmt.Printf("💡 Using free CoinGecko API - no API key required!\n\n")

	// Fetch historical data using CoinGecko API
	histData, err := coindesk.GetHistoricalBitcoinPrices(*startDate, *endDate)
	if err != nil {
		log.Fatalf("❌ Error fetching historical data: %v", err)
	}

	fmt.Printf("✅ Fetched %d price points\n", len(histData.Data))

	// Save to file
	if err := saveHistoricalData(histData, *output); err != nil {
		log.Fatalf("❌ Error saving data: %v", err)
	}

	fmt.Printf("💾 Data saved to %s\n", *output)

	// Print sample of data
	if len(histData.Data) > 0 {
		fmt.Printf("\n📋 Sample data:\n")
		fmt.Printf("   First: %s - $%.2f\n", histData.Data[0].Date, histData.Data[0].Close)
		if len(histData.Data) > 1 {
			lastIdx := len(histData.Data) - 1
			fmt.Printf("   Last:  %s - $%.2f\n", histData.Data[lastIdx].Date, histData.Data[lastIdx].Close)
		}
	}

	fmt.Printf("\n🎉 Success! CoinGecko data collected successfully\n")
	fmt.Printf("📊 Source: %s\n", histData.Source)
	fmt.Printf("⏰ Generated at: %s\n", histData.FetchedAt.Format("2006-01-02 15:04:05"))
}

func saveHistoricalData(data *coindesk.HistoricalBitcoinData, outputDir string) error {
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
