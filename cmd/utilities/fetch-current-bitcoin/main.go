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

// Use the structures from coindesk package

func main() {
	var (
		days      = flag.Int("days", 7, "Number of days to fetch (default: 7)")
		outputDir = flag.String("output", "data/bitcoin-prices/historical", "Output directory")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *verbose {
		fmt.Printf("₿ CURRENT BITCOIN PRICE FETCHER\n")
		fmt.Printf("===============================\n\n")
		fmt.Printf("📅 Days to fetch: %d\n", *days)
		fmt.Printf("💾 Output Directory: %s\n\n", *outputDir)
	}

	// Create CoinGecko client
	client := coindesk.NewClient()

	// Get current price first
	if *verbose {
		fmt.Printf("🔄 Fetching current Bitcoin price...\n")
	}

	currentPrice, err := client.GetCurrentPrice()
	if err != nil {
		log.Fatalf("❌ Error fetching current Bitcoin price: %v", err)
	}

	if *verbose {
		fmt.Printf("✅ Current Bitcoin price: $%.2f\n", currentPrice.Bitcoin.USD)
	}

	// Calculate date range for historical data
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -*days)

	if *verbose {
		fmt.Printf("🔄 Fetching historical data from %s to %s...\n",
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	}

	// Get historical data
	historicalData, err := client.GetHistoricalPrices(
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)
	if err != nil {
		log.Fatalf("❌ Error fetching historical Bitcoin data: %v", err)
	}

	if *verbose {
		fmt.Printf("✅ Fetched %d historical data points\n", len(historicalData.Data))
	}

	// Add today's current price as the latest data point if not already included
	todayStr := time.Now().Format("2006-01-02")
	hasToday := false
	for _, point := range historicalData.Data {
		if point.Date == todayStr {
			hasToday = true
			break
		}
	}

	if !hasToday {
		todayPoint := coindesk.HistoricalBitcoinPrice{
			Date:   todayStr,
			Open:   currentPrice.Bitcoin.USD,
			High:   currentPrice.Bitcoin.USD,
			Low:    currentPrice.Bitcoin.USD,
			Close:  currentPrice.Bitcoin.USD,
			Volume: 0, // Volume not available from current price API
		}
		historicalData.Data = append(historicalData.Data, todayPoint)
		if *verbose {
			fmt.Printf("✅ Added today's current price: $%.2f\n", currentPrice.Bitcoin.USD)
		}
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	// Save the data
	filename := fmt.Sprintf("bitcoin_current_%s_%ddays.json",
		time.Now().Format("2006-01-02"), *days)
	outputFile := filepath.Join(*outputDir, filename)

	jsonData, err := json.MarshalIndent(historicalData, "", "  ")
	if err != nil {
		log.Fatalf("❌ Error marshaling data: %v", err)
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		log.Fatalf("❌ Error writing file: %v", err)
	}

	if *verbose {
		fmt.Printf("💾 Bitcoin data saved to: %s\n", outputFile)
		fmt.Printf("✅ Bitcoin price fetch complete!\n")
	} else {
		fmt.Printf("Bitcoin data saved to: %s\n", outputFile)
	}
}
