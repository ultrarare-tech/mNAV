package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/collection/yahoo"
)

func main() {
	var (
		ticker    = flag.String("ticker", "MSTR", "Stock ticker symbol")
		outputDir = flag.String("output-dir", "data/stock-prices", "Output directory for stock price data")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)

	flag.Parse()

	fmt.Printf("📈 STOCK PRICE COLLECTION\n")
	fmt.Printf("=========================\n\n")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	if *verbose {
		fmt.Printf("📊 Configuration:\n")
		fmt.Printf("   • Ticker: %s\n", *ticker)
		fmt.Printf("   • Output Directory: %s\n", *outputDir)
		fmt.Printf("\n")
	}

	// Fetch current stock price
	fmt.Printf("🔍 Fetching current stock price for %s...\n", *ticker)

	stockPrice, err := yahoo.GetStockPrice(*ticker)
	if err != nil {
		log.Fatalf("❌ Error fetching stock price: %v", err)
	}

	// Display results
	fmt.Printf("✅ Stock price retrieved successfully!\n\n")
	fmt.Printf("📊 %s Stock Information:\n", *ticker)
	fmt.Printf("   • Current Price: $%.2f\n", stockPrice.Price)
	fmt.Printf("   • Change: $%.2f (%.2f%%)\n", stockPrice.Change, stockPrice.ChangePercent)
	fmt.Printf("   • Volume: %s\n", formatNumber(stockPrice.Volume))
	fmt.Printf("   • Market Cap: $%s\n", formatMarketCap(stockPrice.MarketCap))
	fmt.Printf("   • Outstanding Shares: %s\n", formatNumber(int64(stockPrice.OutstandingShares)))
	fmt.Printf("   • Last Updated: %s\n", stockPrice.LastUpdated.Format("2006-01-02 15:04:05 MST"))

	// Save to file
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_stock_price_%s.json", *ticker, timestamp)
	filepath := filepath.Join(*outputDir, filename)

	// Create JSON output
	output := map[string]interface{}{
		"ticker":             *ticker,
		"timestamp":          time.Now().Format(time.RFC3339),
		"price":              stockPrice.Price,
		"change":             stockPrice.Change,
		"change_percent":     safeFloat(stockPrice.ChangePercent),
		"volume":             stockPrice.Volume,
		"market_cap":         stockPrice.MarketCap,
		"outstanding_shares": stockPrice.OutstandingShares,
		"last_updated":       stockPrice.LastUpdated.Format(time.RFC3339),
		"currency":           "USD",
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("❌ Error marshaling JSON: %v", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		log.Fatalf("❌ Error writing file: %v", err)
	}

	fmt.Printf("\n💾 Data saved to: %s\n", filepath)

	if *verbose {
		fmt.Printf("\n📄 JSON Output:\n%s\n", string(jsonData))
	}

	fmt.Printf("\n✅ Stock price collection complete!\n")
}

// formatNumber formats large numbers with commas
func formatNumber(n int64) string {
	if n == 0 {
		return "0"
	}

	str := fmt.Sprintf("%d", n)
	result := ""

	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(char)
	}

	return result
}

// formatMarketCap formats market cap in billions/millions
func formatMarketCap(marketCap float64) string {
	if marketCap == 0 {
		return "N/A"
	}

	if marketCap >= 1e9 {
		return fmt.Sprintf("%.2fB", marketCap/1e9)
	} else if marketCap >= 1e6 {
		return fmt.Sprintf("%.2fM", marketCap/1e6)
	} else if marketCap >= 1e3 {
		return fmt.Sprintf("%.2fK", marketCap/1e3)
	}

	return fmt.Sprintf("%.2f", marketCap)
}

// safeFloat safely handles float64 values, converting infinity and NaN to 0
func safeFloat(f float64) float64 {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return 0
	}
	return f
}
