package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/collection/scraper"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

func main() {
	var (
		outputDir = flag.String("output", "data/analysis", "Output directory")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	fmt.Printf("📊 MSTR BITCOIN HOLDINGS FETCHER\n")
	fmt.Printf("================================\n\n")

	// Fetch MSTR Bitcoin holdings data
	fmt.Printf("🔍 Fetching MSTR Bitcoin holdings from BitBo.io...\n")
	holdings, err := scraper.GetMSTRBitcoinHoldings()
	if err != nil {
		log.Fatalf("❌ Error fetching holdings: %v", err)
	}

	fmt.Printf("✅ Retrieved MSTR Bitcoin data:\n")
	fmt.Printf("   • Total Bitcoin: %.0f BTC\n", holdings.TotalBTC)
	fmt.Printf("   • Average Price: $%.2f per BTC\n", holdings.AveragePriceUSD)
	fmt.Printf("   • Purchase Transactions: %d\n", len(holdings.Purchases))

	if *verbose {
		fmt.Printf("\n📅 Purchase History:\n")
		for i, purchase := range holdings.Purchases {
			if i < 10 { // Show first 10 for brevity
				fmt.Printf("   %s: %.0f BTC for $%.2fM (%.0f total BTC)\n",
					purchase.Date, purchase.BTCPurchased, purchase.AmountUSD/1000000, purchase.TotalBitcoin)
			}
		}
		if len(holdings.Purchases) > 10 {
			fmt.Printf("   ... and %d more transactions\n", len(holdings.Purchases)-10)
		}
	}

	// Convert to our standard transaction format
	transactions := make([]models.BitcoinTransaction, 0, len(holdings.Purchases))
	for _, purchase := range holdings.Purchases {
		// Parse date
		date, err := parseFlexibleDate(purchase.Date)
		if err != nil {
			if *verbose {
				fmt.Printf("   ⚠️  Could not parse date '%s': %v\n", purchase.Date, err)
			}
			continue
		}

		tx := models.BitcoinTransaction{
			Date:            date,
			FilingType:      "Web Scraping",
			FilingURL:       "https://treasuries.bitbo.io/microstrategy/",
			BTCPurchased:    purchase.BTCPurchased,
			USDSpent:        purchase.AmountUSD,
			AvgPriceUSD:     purchase.AmountUSD / purchase.BTCPurchased,
			TotalBTCAfter:   purchase.TotalBitcoin,
			ExtractedText:   fmt.Sprintf("Scraped data: %s, %.0f BTC, $%.2fM", purchase.Date, purchase.BTCPurchased, purchase.AmountUSD/1000000),
			ConfidenceScore: 0.8, // Web scraping has good accuracy but not as high as SEC filings
		}
		transactions = append(transactions, tx)
	}

	fmt.Printf("\n✅ Converted %d purchases to standard transaction format\n", len(transactions))

	// Save the data
	if err := saveTransactionData(transactions, holdings, *outputDir); err != nil {
		log.Fatalf("❌ Error saving data: %v", err)
	}

	fmt.Printf("💾 Data saved to %s\n", *outputDir)
	fmt.Printf("\n🎉 MSTR Bitcoin holdings data collection complete!\n")
}

func parseFlexibleDate(dateStr string) (time.Time, error) {
	// Try multiple date formats
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"01/02/06",
		"1/2/06",
		"2006/01/02",
		"2006/1/2",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func saveTransactionData(transactions []models.BitcoinTransaction, holdings *scraper.MSTRHoldings, outputDir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create comprehensive analysis file
	analysisData := map[string]interface{}{
		"symbol":          "MSTR",
		"fetched_at":      time.Now(),
		"source":          "BitBo.io scraping",
		"total_btc":       holdings.TotalBTC,
		"average_price":   holdings.AveragePriceUSD,
		"total_cost":      holdings.TotalCostUSD,
		"allTransactions": transactions,
		"summary": map[string]interface{}{
			"transaction_count":    len(transactions),
			"earliest_transaction": getEarliestDate(transactions),
			"latest_transaction":   getLatestDate(transactions),
			"total_btc_acquired":   getTotalBTC(transactions),
			"total_usd_spent":      getTotalUSD(transactions),
		},
	}

	// Save comprehensive analysis
	filename := "MSTR_comprehensive_bitcoin_analysis.json"
	filePath := filepath.Join(outputDir, filename)

	jsonData, err := json.MarshalIndent(analysisData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("   ✅ Saved comprehensive analysis: %s\n", filename)

	// Also save raw holdings data
	holdingsFilename := "MSTR_bitcoin_holdings_raw.json"
	holdingsPath := filepath.Join(outputDir, holdingsFilename)

	holdingsData, err := json.MarshalIndent(holdings, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling holdings: %w", err)
	}

	if err := os.WriteFile(holdingsPath, holdingsData, 0644); err != nil {
		return fmt.Errorf("error writing holdings file: %w", err)
	}

	fmt.Printf("   ✅ Saved raw holdings data: %s\n", holdingsFilename)

	return nil
}

func getEarliestDate(transactions []models.BitcoinTransaction) string {
	if len(transactions) == 0 {
		return ""
	}
	earliest := transactions[0].Date
	for _, tx := range transactions {
		if tx.Date.Before(earliest) {
			earliest = tx.Date
		}
	}
	return earliest.Format("2006-01-02")
}

func getLatestDate(transactions []models.BitcoinTransaction) string {
	if len(transactions) == 0 {
		return ""
	}
	latest := transactions[0].Date
	for _, tx := range transactions {
		if tx.Date.After(latest) {
			latest = tx.Date
		}
	}
	return latest.Format("2006-01-02")
}

func getTotalBTC(transactions []models.BitcoinTransaction) float64 {
	total := 0.0
	for _, tx := range transactions {
		total += tx.BTCPurchased
	}
	return total
}

func getTotalUSD(transactions []models.BitcoinTransaction) float64 {
	total := 0.0
	for _, tx := range transactions {
		total += tx.USDSpent
	}
	return total
}
