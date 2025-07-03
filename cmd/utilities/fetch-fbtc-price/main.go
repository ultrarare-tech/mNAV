package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"

	"github.com/ultrarare-tech/mNAV/pkg/collection/yahoo"
)

func main() {
	var (
		symbol  = flag.String("symbol", "FBTC", "Stock symbol to fetch")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
		format  = flag.String("format", "simple", "Output format: simple, json")
	)
	flag.Parse()

	if *verbose {
		fmt.Printf("📈 Fetching current price for %s...\n", *symbol)
	}

	// Fetch stock price using Yahoo Finance
	stockPrice, err := yahoo.GetStockPrice(*symbol)
	if err != nil {
		log.Fatalf("❌ Error fetching stock price: %v", err)
	}

	// Debug output to see what we got
	if *verbose {
		fmt.Printf("Debug - Raw values:\n")
		fmt.Printf("  Price: %f\n", stockPrice.Price)
		fmt.Printf("  Change: %f\n", stockPrice.Change)
		fmt.Printf("  ChangePercent: %f\n", stockPrice.ChangePercent)
		fmt.Printf("  Volume: %d\n", stockPrice.Volume)
		fmt.Printf("  MarketCap: %f\n", stockPrice.MarketCap)
	}

	// Handle infinite or NaN values
	if math.IsInf(stockPrice.Price, 0) || math.IsNaN(stockPrice.Price) {
		log.Fatalf("❌ Invalid price data received: %f", stockPrice.Price)
	}

	switch *format {
	case "json":
		// Clean up any infinite or NaN values before JSON marshaling
		cleanStockPrice := cleanInvalidValues(stockPrice)
		jsonData, err := json.MarshalIndent(cleanStockPrice, "", "  ")
		if err != nil {
			log.Fatalf("❌ Error marshaling to JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	case "simple":
		fmt.Printf("%.2f", stockPrice.Price)
	default:
		fmt.Printf("%s: $%.2f\n", stockPrice.Symbol, stockPrice.Price)
		if *verbose {
			fmt.Printf("   Change: $%.2f (%.2f%%)\n", stockPrice.Change, stockPrice.ChangePercent)
			fmt.Printf("   Volume: %d\n", stockPrice.Volume)
			fmt.Printf("   Last Updated: %s\n", stockPrice.LastUpdated.Format("2006-01-02 15:04:05"))
		}
	}
}

// cleanInvalidValues removes infinite and NaN values from the struct
func cleanInvalidValues(sp *yahoo.StockPrice) *yahoo.StockPrice {
	clean := *sp

	if math.IsInf(clean.Price, 0) || math.IsNaN(clean.Price) {
		clean.Price = 0
	}
	if math.IsInf(clean.Change, 0) || math.IsNaN(clean.Change) {
		clean.Change = 0
	}
	if math.IsInf(clean.ChangePercent, 0) || math.IsNaN(clean.ChangePercent) {
		clean.ChangePercent = 0
	}
	if math.IsInf(clean.MarketCap, 0) || math.IsNaN(clean.MarketCap) {
		clean.MarketCap = 0
	}
	if math.IsInf(clean.OutstandingShares, 0) || math.IsNaN(clean.OutstandingShares) {
		clean.OutstandingShares = 0
	}

	return &clean
}
