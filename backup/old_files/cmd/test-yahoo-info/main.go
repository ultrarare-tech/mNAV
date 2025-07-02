package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/yahoo"
)

func main() {
	// Define command-line flags
	symbol := flag.String("symbol", "MSTR", "Stock symbol to fetch info for")
	flag.Parse()

	fmt.Printf("Testing Yahoo Finance Info API for %s\n", *symbol)
	fmt.Println("=========================================")

	// First test the existing working method to verify our setup
	fmt.Println("\n0. Testing existing GetStockPrice method...")
	existingStock, err := yahoo.GetStockPrice(*symbol)
	if err != nil {
		log.Printf("Error with existing method: %v", err)
	} else {
		fmt.Printf("Existing method works! Price: $%.2f, Shares: %.0f\n",
			existingStock.Price, existingStock.OutstandingShares)
	}

	// Add delay to avoid rate limiting
	fmt.Println("\nWaiting 2 seconds before next test...")
	time.Sleep(2 * time.Second)

	// Test GetStockInfo
	fmt.Println("\n1. Testing GetStockInfo...")
	info, err := yahoo.GetStockInfo(*symbol)
	if err != nil {
		log.Printf("Error getting stock info: %v", err)
	} else {
		fmt.Printf("Symbol: %s\n", info.Symbol)
		fmt.Printf("Name: %s\n", info.LongName)
		fmt.Printf("Currency: %s\n", info.Currency)
		fmt.Printf("Current Price: $%.2f\n", info.RegularMarketPrice)
		fmt.Printf("Market Cap: $%.2f billion\n", info.MarketCap/1e9)
		fmt.Printf("Shares Outstanding: %.0f (%.2f million)\n", info.SharesOutstanding, info.SharesOutstanding/1e6)
		fmt.Printf("Float Shares: %.0f (%.2f million)\n", info.FloatShares, info.FloatShares/1e6)
		fmt.Printf("Implied Shares: %.0f (%.2f million)\n", info.ImpliedSharesOutstanding, info.ImpliedSharesOutstanding/1e6)
		fmt.Printf("Enterprise Value: $%.2f billion\n", info.EnterpriseValue/1e9)
		fmt.Printf("Beta: %.2f\n", info.Beta)
		fmt.Printf("P/B Ratio: %.2f\n", info.PriceToBook)
		fmt.Printf("Trailing P/E: %.2f\n", info.TrailingPE)
		fmt.Printf("Forward P/E: %.2f\n", info.ForwardPE)
		fmt.Printf("50-Day Average: $%.2f\n", info.FiftyDayAverage)
		fmt.Printf("200-Day Average: $%.2f\n", info.TwoHundredDayAverage)
		fmt.Printf("Volume: %d\n", info.Volume)
		fmt.Printf("Average Volume: %d\n", info.AverageVolume)
		fmt.Printf("Last Updated: %s\n", info.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	// Test GetSharesOutstanding
	fmt.Println("\n2. Testing GetSharesOutstanding...")
	shares, err := yahoo.GetSharesOutstanding(*symbol)
	if err != nil {
		log.Printf("Error getting shares outstanding: %v", err)
	} else {
		fmt.Printf("Shares Outstanding: %.0f (%.2f million)\n", shares, shares/1e6)

		// Compare with existing method
		fmt.Println("\n3. Comparing with existing GetStockPrice method...")
		stockPrice, err := yahoo.GetStockPrice(*symbol)
		if err != nil {
			log.Printf("Error getting stock price: %v", err)
		} else {
			fmt.Printf("Price from GetStockPrice: $%.2f\n", stockPrice.Price)
			fmt.Printf("Shares from GetStockPrice: %.0f (%.2f million)\n", stockPrice.OutstandingShares, stockPrice.OutstandingShares/1e6)
			fmt.Printf("Market Cap from GetStockPrice: $%.2f billion\n", stockPrice.MarketCap/1e9)

			if info != nil {
				fmt.Printf("\nDifference in shares: %.0f (%.2f%%)\n",
					shares-stockPrice.OutstandingShares,
					((shares-stockPrice.OutstandingShares)/stockPrice.OutstandingShares)*100)
			}
		}
	}

	// Test multiple symbols
	fmt.Println("\n4. Testing multiple symbols...")
	symbols := []string{"MSTR", "MARA", "AAPL"}
	results, err := yahoo.GetMultipleStockInfo(symbols)
	if err != nil {
		log.Printf("Error getting multiple stock info: %v", err)
	} else {
		fmt.Println("\nSymbol | Name | Shares Outstanding | Market Cap")
		fmt.Println("-------|------|-------------------|------------")
		for symbol, info := range results {
			fmt.Printf("%-6s | %-20s | %15.0f | $%10.2fB\n",
				symbol,
				truncateString(info.LongName, 20),
				info.SharesOutstanding,
				info.MarketCap/1e9)
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
