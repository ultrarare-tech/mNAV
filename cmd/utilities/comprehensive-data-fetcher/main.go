package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/collection/alphavantage"
	"github.com/jeffreykibler/mNAV/pkg/collection/external"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

func main() {
	var (
		symbol    = flag.String("symbol", "MSTR", "Stock symbol to analyze")
		outputDir = flag.String("output", "data/analysis", "Output directory")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	fmt.Printf("📊 COMPREHENSIVE DATA FETCHER - Option C Implementation\n")
	fmt.Printf("======================================================\n\n")
	fmt.Printf("🔍 Fetching comprehensive data for %s...\n", *symbol)
	fmt.Printf("📈 Sources: SaylorTracker-style aggregation + Alpha Vantage\n\n")

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	// Initialize clients
	saylorClient := external.NewSaylorTrackerClient()
	alphaVantageClient := alphavantage.NewSharesOutstandingClient()

	// Fetch comprehensive Bitcoin transaction data
	fmt.Printf("🪙 Fetching comprehensive Bitcoin transaction history...\n")
	bitcoinData, err := saylorClient.GetComprehensiveMSTRData()
	if err != nil {
		log.Fatalf("❌ Error fetching Bitcoin data: %v", err)
	}

	if *verbose {
		fmt.Printf("   ✅ Found %d Bitcoin transactions\n", len(bitcoinData.Transactions))
		fmt.Printf("   📊 Total Bitcoin: %.0f BTC\n", bitcoinData.TotalBitcoin)
		fmt.Printf("   💰 Total Investment: $%.2f billion\n", bitcoinData.TotalInvestment/1_000_000_000)
		fmt.Printf("   📈 Average Price: $%.2f per BTC\n", bitcoinData.AveragePrice)
	}

	// Fetch historical shares outstanding data
	fmt.Printf("📈 Fetching historical shares outstanding from Alpha Vantage...\n")
	sharesData, err := alphaVantageClient.GetHistoricalSharesOutstanding(*symbol)
	if err != nil {
		log.Printf("⚠️  Warning: Could not fetch shares data from Alpha Vantage: %v", err)
		fmt.Printf("   📝 Using estimated shares outstanding data\n")
	} else {
		if *verbose {
			fmt.Printf("   ✅ Found %d historical shares data points\n", len(sharesData.HistoricalData))
			fmt.Printf("   📊 Current Shares Outstanding: %.0f million\n", sharesData.CurrentSharesOutstanding/1_000_000)
		}
	}

	// Convert to standard format
	standardData, err := saylorClient.ConvertToStandardFormat(bitcoinData)
	if err != nil {
		log.Fatalf("❌ Error converting data: %v", err)
	}

	// Enhance with shares outstanding data
	if sharesData != nil {
		enhanceWithSharesData(standardData, sharesData)
	}

	// Save comprehensive data
	if err := saveComprehensiveData(standardData, sharesData, bitcoinData, *outputDir, *symbol); err != nil {
		log.Fatalf("❌ Error saving data: %v", err)
	}

	fmt.Printf("\n🎉 Comprehensive data collection complete!\n")
	fmt.Printf("📁 Data saved to: %s\n", *outputDir)
	fmt.Printf("📊 Ready for enhanced mNAV analysis with full historical context\n")
}

// enhanceWithSharesData adds shares outstanding information to Bitcoin analysis
func enhanceWithSharesData(bitcoinData *models.ComprehensiveBitcoinAnalysis, sharesData *alphavantage.HistoricalSharesData) {
	// Add shares outstanding metadata
	if bitcoinData.Metadata == nil {
		bitcoinData.Metadata = make(map[string]interface{})
	}

	bitcoinData.Metadata["shares_outstanding_source"] = sharesData.Source
	bitcoinData.Metadata["current_shares_outstanding"] = sharesData.CurrentSharesOutstanding
	bitcoinData.Metadata["shares_data_points"] = len(sharesData.HistoricalData)

	// Create shares outstanding lookup for transaction dates
	sharesLookup := make(map[string]float64)
	for _, dataPoint := range sharesData.HistoricalData {
		sharesLookup[dataPoint.Date] = dataPoint.SharesOutstanding
	}

	// Enhance transactions with nearest shares outstanding data
	for i := range bitcoinData.AllTransactions {
		tx := &bitcoinData.AllTransactions[i]

		// Use the transaction date directly (already time.Time)
		txDate := tx.Date

		var nearestShares float64
		var nearestDiff time.Duration = time.Hour * 24 * 365 * 10 // 10 years

		for dateStr, shares := range sharesLookup {
			shareDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}

			diff := txDate.Sub(shareDate)
			if diff < 0 {
				diff = -diff
			}

			if diff < nearestDiff {
				nearestDiff = diff
				nearestShares = shares
			}
		}

		if nearestShares > 0 {
			if tx.Metadata == nil {
				tx.Metadata = make(map[string]interface{})
			}
			tx.Metadata["shares_outstanding"] = nearestShares
			tx.Metadata["bitcoin_per_share"] = tx.TotalBTCAfter / nearestShares
		}
	}
}

// saveComprehensiveData saves all collected data in multiple formats
func saveComprehensiveData(standardData *models.ComprehensiveBitcoinAnalysis, sharesData *alphavantage.HistoricalSharesData, rawBitcoinData *external.SaylorTrackerResponse, outputDir, symbol string) error {
	timestamp := time.Now().Format("2006-01-02")

	// Save standard Bitcoin transaction data
	bitcoinFilename := fmt.Sprintf("%s_comprehensive_bitcoin_analysis.json", symbol)
	bitcoinPath := filepath.Join(outputDir, bitcoinFilename)

	bitcoinJSON, err := json.MarshalIndent(standardData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling Bitcoin data: %w", err)
	}

	if err := os.WriteFile(bitcoinPath, bitcoinJSON, 0644); err != nil {
		return fmt.Errorf("error writing Bitcoin data: %w", err)
	}
	fmt.Printf("   ✅ Saved: %s\n", bitcoinFilename)

	// Save shares outstanding data if available
	if sharesData != nil {
		sharesFilename := fmt.Sprintf("%s_shares_outstanding_%s.json", symbol, timestamp)
		sharesPath := filepath.Join(outputDir, sharesFilename)

		sharesJSON, err := json.MarshalIndent(sharesData, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling shares data: %w", err)
		}

		if err := os.WriteFile(sharesPath, sharesJSON, 0644); err != nil {
			return fmt.Errorf("error writing shares data: %w", err)
		}
		fmt.Printf("   ✅ Saved: %s\n", sharesFilename)
	}

	// Save raw comprehensive data
	rawFilename := fmt.Sprintf("%s_raw_comprehensive_data_%s.json", symbol, timestamp)
	rawPath := filepath.Join(outputDir, rawFilename)

	rawJSON, err := json.MarshalIndent(rawBitcoinData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling raw data: %w", err)
	}

	if err := os.WriteFile(rawPath, rawJSON, 0644); err != nil {
		return fmt.Errorf("error writing raw data: %w", err)
	}
	fmt.Printf("   ✅ Saved: %s\n", rawFilename)

	// Create combined summary
	summary := map[string]interface{}{
		"symbol":                symbol,
		"last_updated":          time.Now().Format("2006-01-02T15:04:05Z"),
		"data_sources":          []string{"SaylorTracker-style aggregation", "Alpha Vantage API"},
		"bitcoin_transactions":  len(standardData.AllTransactions),
		"total_bitcoin":         standardData.TotalBTC,
		"total_investment_usd":  standardData.TotalInvestmentUSD,
		"average_price_usd":     standardData.AveragePrice,
		"shares_data_available": sharesData != nil,
	}

	if sharesData != nil {
		summary["current_shares_outstanding"] = sharesData.CurrentSharesOutstanding
		summary["shares_data_points"] = len(sharesData.HistoricalData)
		summary["bitcoin_per_share_current"] = standardData.TotalBTC / sharesData.CurrentSharesOutstanding
	}

	summaryFilename := fmt.Sprintf("%s_comprehensive_summary_%s.json", symbol, timestamp)
	summaryPath := filepath.Join(outputDir, summaryFilename)

	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling summary: %w", err)
	}

	if err := os.WriteFile(summaryPath, summaryJSON, 0644); err != nil {
		return fmt.Errorf("error writing summary: %w", err)
	}
	fmt.Printf("   ✅ Saved: %s\n", summaryFilename)

	return nil
}
