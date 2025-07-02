package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/ultrarare-tech/mNAV/pkg/scraper"
)

func main() {
	// Define command-line flags
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	csvOutput := flag.Bool("csv", false, "Output in CSV format")
	latestOnly := flag.Bool("latest", false, "Show only latest purchase")
	summaryOnly := flag.Bool("summary", false, "Show only summary data")
	flag.Parse()

	// Fetch MSTR Bitcoin holdings
	holdings, err := scraper.GetMSTRBitcoinHoldings()
	if err != nil {
		log.Fatalf("Error fetching MSTR Bitcoin holdings: %v", err)
	}

	// Output based on flags
	if *jsonOutput {
		// JSON output
		jsonData, err := json.MarshalIndent(holdings, "", "  ")
		if err != nil {
			log.Fatalf("Error generating JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return
	}

	if *csvOutput {
		// CSV output
		fmt.Println("Date,BTC Purchased,Amount USD,Total Bitcoin,Total USD Spent")
		for _, purchase := range holdings.Purchases {
			fmt.Printf("%s,%.2f,%.2f,%.2f,%.2f\n",
				purchase.Date,
				purchase.BTCPurchased,
				purchase.AmountUSD,
				purchase.TotalBitcoin,
				purchase.TotalUSDSpent)
		}
		return
	}

	if *latestOnly {
		// Show only latest purchase
		if len(holdings.Purchases) > 0 {
			latest := holdings.Purchases[0]
			fmt.Println("Latest MicroStrategy Bitcoin Purchase:")
			fmt.Printf("Date: %s\n", latest.Date)
			fmt.Printf("BTC Purchased: %.2f\n", latest.BTCPurchased)
			fmt.Printf("Amount Spent: $%.2f\n", latest.AmountUSD)
			fmt.Printf("Total BTC After Purchase: %.2f\n", latest.TotalBitcoin)
			fmt.Printf("Total USD Spent: $%.2f\n", latest.TotalUSDSpent)
		} else {
			fmt.Println("No purchase data found")
		}
		return
	}

	if *summaryOnly {
		// Show only summary
		fmt.Println("MicroStrategy Bitcoin Holdings Summary:")
		fmt.Printf("Total BTC: %.2f\n", holdings.TotalBTC)
		fmt.Printf("Current Value: $%.2f\n", holdings.ValueToday)
		fmt.Printf("Percentage of 21M Supply: %.4f%%\n", holdings.PercentageOfSupply*100)
		fmt.Printf("Average Purchase Price: $%.2f\n", holdings.AveragePriceUSD)
		fmt.Printf("Total Cost: $%.2f\n", holdings.TotalCostUSD)
		fmt.Printf("Last Updated: %s\n", holdings.LastUpdated.Format("2006-01-02 15:04:05"))
		return
	}

	// Default output (summary + purchase history)
	fmt.Println("MicroStrategy Bitcoin Holdings:")
	fmt.Printf("Total BTC: %.2f\n", holdings.TotalBTC)
	fmt.Printf("Current Value: $%.2f\n", holdings.ValueToday)
	fmt.Printf("Percentage of 21M Supply: %.4f%%\n", holdings.PercentageOfSupply*100)
	fmt.Printf("Average Purchase Price: $%.2f\n", holdings.AveragePriceUSD)
	fmt.Printf("Total Cost: $%.2f\n", holdings.TotalCostUSD)
	fmt.Printf("Last Updated: %s\n\n", holdings.LastUpdated.Format("2006-01-02 15:04:05"))

	fmt.Println("Purchase History:")
	fmt.Println("---------------------------------------------------")
	fmt.Printf("%-12s %-12s %-12s %-12s %-12s\n", "Date", "BTC", "Amount", "Total BTC", "Total USD")
	fmt.Println("---------------------------------------------------")

	for _, purchase := range holdings.Purchases {
		fmt.Printf("%-12s %-12.2f $%-11.2f %-12.2f $%-11.2f\n",
			purchase.Date,
			purchase.BTCPurchased,
			purchase.AmountUSD,
			purchase.TotalBitcoin,
			purchase.TotalUSDSpent)
	}

	fmt.Println("---------------------------------------------------")
	fmt.Printf("Total Purchases: %d\n", len(holdings.Purchases))
}
