package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
)

// SaylorTrackerData represents the data from SaylorTracker website
type SaylorTrackerData struct {
	TotalBTC         float64 `json:"total_btc"`
	TotalUSDInvested float64 `json:"total_usd_invested"`
	AverageCostBasis float64 `json:"average_cost_basis"`
	LastUpdated      string  `json:"last_updated"`
}

// Our transaction structure
type Transaction struct {
	Date       string  `json:"date"`
	BTCAmount  float64 `json:"btc_amount"`
	USDAmount  float64 `json:"usd_amount"`
	AvgPrice   float64 `json:"avg_price"`
	FilingType string  `json:"filing_type"`
	Source     string  `json:"source"`
}

type OurData struct {
	Summary struct {
		TotalTransactions  int     `json:"total_transactions"`
		TotalBTCAcquired   float64 `json:"total_btc_acquired"`
		TotalUSDInvested   float64 `json:"total_usd_invested"`
		AverageCostBasis   float64 `json:"average_cost_basis"`
		FirstPurchaseDate  string  `json:"first_purchase_date"`
		LatestPurchaseDate string  `json:"latest_purchase_date"`
	} `json:"summary"`
	Transactions []Transaction `json:"transactions"`
}

func main() {
	// SaylorTracker data as of the website (May 26, 2025)
	saylorData := SaylorTrackerData{
		TotalBTC:         580250,
		TotalUSDInvested: 40610000000, // ~$40.61 billion
		AverageCostBasis: 69979,       // ~$69,979 per bitcoin
		LastUpdated:      "2025-05-26",
	}

	// Load our data
	file, err := os.Open("data/analysis/mstr_bitcoin_transactions.json")
	if err != nil {
		log.Fatalf("Error opening our data file: %v", err)
	}
	defer file.Close()

	var ourData OurData
	if err := json.NewDecoder(file).Decode(&ourData); err != nil {
		log.Fatalf("Error decoding our data: %v", err)
	}

	fmt.Println("=== MSTR Bitcoin Holdings Comparison ===")
	fmt.Println()

	fmt.Println("📊 SUMMARY COMPARISON:")
	fmt.Printf("SaylorTracker Total BTC:    %.0f BTC\n", saylorData.TotalBTC)
	fmt.Printf("Our Analysis Total BTC:     %.0f BTC\n", ourData.Summary.TotalBTCAcquired)
	btcDiff := ourData.Summary.TotalBTCAcquired - saylorData.TotalBTC
	fmt.Printf("Difference:                 %+.0f BTC (%.2f%%)\n", btcDiff, (btcDiff/saylorData.TotalBTC)*100)
	fmt.Println()

	fmt.Printf("SaylorTracker Total USD:    $%.0f billion\n", saylorData.TotalUSDInvested/1000000000)
	fmt.Printf("Our Analysis Total USD:     $%.1f billion\n", ourData.Summary.TotalUSDInvested/1000000000)
	usdDiff := ourData.Summary.TotalUSDInvested - saylorData.TotalUSDInvested
	fmt.Printf("Difference:                 $%+.1f billion (%.2f%%)\n", usdDiff/1000000000, (usdDiff/saylorData.TotalUSDInvested)*100)
	fmt.Println()

	fmt.Printf("SaylorTracker Avg Cost:     $%.2f\n", saylorData.AverageCostBasis)
	fmt.Printf("Our Analysis Avg Cost:      $%.2f\n", ourData.Summary.AverageCostBasis)
	costDiff := ourData.Summary.AverageCostBasis - saylorData.AverageCostBasis
	fmt.Printf("Difference:                 $%+.2f (%.2f%%)\n", costDiff, (costDiff/saylorData.AverageCostBasis)*100)
	fmt.Println()

	fmt.Println("📅 DATE RANGE COMPARISON:")
	fmt.Printf("SaylorTracker Last Update:  %s\n", saylorData.LastUpdated)
	fmt.Printf("Our Latest Transaction:     %s\n", ourData.Summary.LatestPurchaseDate)
	fmt.Printf("Our First Transaction:      %s\n", ourData.Summary.FirstPurchaseDate)
	fmt.Printf("Our Total Transactions:     %d\n", ourData.Summary.TotalTransactions)
	fmt.Println()

	// Analyze potential causes of discrepancy
	fmt.Println("🔍 POTENTIAL CAUSES OF DISCREPANCY:")
	fmt.Println()

	if btcDiff > 0 {
		fmt.Printf("1. OVER-COUNTING: We have %.0f MORE BTC than SaylorTracker\n", btcDiff)
		fmt.Println("   Possible causes:")
		fmt.Println("   - Double-counting transactions from multiple filings")
		fmt.Println("   - Including cumulative totals instead of incremental purchases")
		fmt.Println("   - Parsing errors creating phantom transactions")
		fmt.Println("   - Including non-purchase transactions (transfers, etc.)")
	} else {
		fmt.Printf("1. UNDER-COUNTING: We have %.0f FEWER BTC than SaylorTracker\n", -btcDiff)
		fmt.Println("   Possible causes:")
		fmt.Println("   - Missing recent transactions not yet in SEC filings")
		fmt.Println("   - Failed to parse some transactions from filings")
		fmt.Println("   - SaylorTracker includes data from Twitter/press releases")
	}
	fmt.Println()

	// Analyze transaction sources
	grokCount := 0
	regexCount := 0
	for _, tx := range ourData.Transactions {
		if tx.Source == "Grok AI" {
			grokCount++
		} else {
			regexCount++
		}
	}

	fmt.Println("2. DATA SOURCE ANALYSIS:")
	fmt.Printf("   - Grok AI extracted:     %d transactions\n", grokCount)
	fmt.Printf("   - Regex extracted:       %d transactions\n", regexCount)
	fmt.Printf("   - Total transactions:    %d\n", len(ourData.Transactions))
	fmt.Println("   - SaylorTracker uses: Twitter + SEC filings + press releases")
	fmt.Println("   - We only use: SEC filings")
	fmt.Println()

	// Check for suspicious transactions
	fmt.Println("3. SUSPICIOUS TRANSACTIONS (potential parsing errors):")
	suspiciousCount := 0
	for _, tx := range ourData.Transactions {
		if tx.AvgPrice < 1000 || tx.AvgPrice > 150000 {
			fmt.Printf("   - %s: %.0f BTC at $%.0f (filing: %s)\n",
				tx.Date, tx.BTCAmount, tx.AvgPrice, tx.FilingType)
			suspiciousCount++
		}
	}
	if suspiciousCount == 0 {
		fmt.Println("   - No obviously suspicious price data found")
	}
	fmt.Println()

	// Recent transactions analysis
	fmt.Println("4. RECENT TRANSACTIONS (May 2025):")
	recentBTC := 0.0
	recentUSD := 0.0
	for _, tx := range ourData.Transactions {
		if tx.Date >= "2025-05-01" {
			recentBTC += tx.BTCAmount
			recentUSD += tx.USDAmount
			fmt.Printf("   - %s: %.0f BTC for $%.0f M\n",
				tx.Date, tx.BTCAmount, tx.USDAmount/1000000)
		}
	}
	fmt.Printf("   Total May 2025: %.0f BTC for $%.0f M\n", recentBTC, recentUSD/1000000)
	fmt.Println()

	// Recommendations
	fmt.Println("💡 RECOMMENDATIONS:")
	fmt.Println("1. Cross-reference with Michael Saylor's Twitter announcements")
	fmt.Println("2. Check for duplicate transactions across different filing types")
	fmt.Println("3. Verify cumulative vs incremental reporting in filings")
	fmt.Println("4. Compare individual transaction dates and amounts with SaylorTracker")
	fmt.Println("5. Consider that SaylorTracker may have more recent data from Twitter")
	fmt.Println()

	// Generate detailed transaction comparison
	fmt.Println("📋 TRANSACTION TIMELINE (Last 10 transactions):")
	sort.Slice(ourData.Transactions, func(i, j int) bool {
		return ourData.Transactions[i].Date > ourData.Transactions[j].Date
	})

	for i, tx := range ourData.Transactions {
		if i >= 10 {
			break
		}
		fmt.Printf("%s: %8.0f BTC @ $%6.0f = $%8.0f M (%s)\n",
			tx.Date, tx.BTCAmount, tx.AvgPrice, tx.USDAmount/1000000, tx.Source)
	}
}
