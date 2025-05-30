package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/collection/coinmarketcap"
)

func main() {
	var (
		outputDir = flag.String("output-dir", "data/bitcoin-prices", "Output directory for Bitcoin price data")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)

	flag.Parse()

	fmt.Printf("₿ BITCOIN PRICE COLLECTION\n")
	fmt.Printf("==========================\n\n")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	if *verbose {
		fmt.Printf("📊 Configuration:\n")
		fmt.Printf("   • Output Directory: %s\n", *outputDir)
		fmt.Printf("\n")
	}

	// Check if API key is set
	if os.Getenv("COINMARKETCAP_API_KEY") == "" {
		fmt.Printf("⚠️  COINMARKETCAP_API_KEY not set, trying free alternative...\n")
		// For demo purposes, we'll use a mock price
		mockBitcoinPrice()
		return
	}

	// Fetch current Bitcoin price
	fmt.Printf("🔍 Fetching current Bitcoin price...\n")

	bitcoinPrice, err := coinmarketcap.GetBitcoinPrice()
	if err != nil {
		log.Fatalf("❌ Error fetching Bitcoin price: %v", err)
	}

	// Display results
	fmt.Printf("✅ Bitcoin price retrieved successfully!\n\n")
	fmt.Printf("₿ Bitcoin Information:\n")
	fmt.Printf("   • Current Price: $%.2f\n", bitcoinPrice.Price)
	fmt.Printf("   • 24h Change: %.2f%%\n", bitcoinPrice.PercentChange24h)
	fmt.Printf("   • Last Updated: %s\n", bitcoinPrice.LastUpdated.Format("2006-01-02 15:04:05 MST"))

	// Save to file
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("BTC_price_%s.json", timestamp)
	filepath := filepath.Join(*outputDir, filename)

	// Create JSON output
	output := map[string]interface{}{
		"symbol":             "BTC",
		"timestamp":          time.Now().Format(time.RFC3339),
		"price":              bitcoinPrice.Price,
		"percent_change_24h": bitcoinPrice.PercentChange24h,
		"last_updated":       bitcoinPrice.LastUpdated.Format(time.RFC3339),
		"currency":           "USD",
		"source":             "CoinMarketCap",
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

	fmt.Printf("\n✅ Bitcoin price collection complete!\n")
}

// mockBitcoinPrice creates a mock Bitcoin price for demo purposes
func mockBitcoinPrice() {
	fmt.Printf("🔍 Using mock Bitcoin price for demonstration...\n")

	// Mock current Bitcoin price (approximate)
	mockPrice := 67500.00
	mockChange := 2.5

	fmt.Printf("✅ Mock Bitcoin price retrieved!\n\n")
	fmt.Printf("₿ Bitcoin Information (Mock):\n")
	fmt.Printf("   • Current Price: $%.2f\n", mockPrice)
	fmt.Printf("   • 24h Change: %.2f%%\n", mockChange)
	fmt.Printf("   • Last Updated: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))

	// Create output directory
	outputDir := "data/bitcoin-prices"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	// Save mock data
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("BTC_price_%s.json", timestamp)
	filepath := filepath.Join(outputDir, filename)

	output := map[string]interface{}{
		"symbol":             "BTC",
		"timestamp":          time.Now().Format(time.RFC3339),
		"price":              mockPrice,
		"percent_change_24h": mockChange,
		"last_updated":       time.Now().Format(time.RFC3339),
		"currency":           "USD",
		"source":             "Mock Data",
		"note":               "Mock data used - set COINMARKETCAP_API_KEY for real data",
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("❌ Error marshaling JSON: %v", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		log.Fatalf("❌ Error writing file: %v", err)
	}

	fmt.Printf("\n💾 Mock data saved to: %s\n", filepath)
	fmt.Printf("\n✅ Bitcoin price collection complete (mock mode)!\n")
}
