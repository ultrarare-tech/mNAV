package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

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

// Known SaylorTracker transactions (recent ones from the website)
var knownTransactions = []Transaction{
	{Date: "2025-05-26", BTCAmount: 4020, USDAmount: 427100000, AvgPrice: 106237},
	{Date: "2025-05-19", BTCAmount: 7390, USDAmount: 764900000, AvgPrice: 103498},
	{Date: "2025-05-12", BTCAmount: 13390, USDAmount: 1340000000, AvgPrice: 99856},
	{Date: "2025-05-05", BTCAmount: 1895, USDAmount: 180300000, AvgPrice: 95167},
	{Date: "2025-04-28", BTCAmount: 15355, USDAmount: 1420000000, AvgPrice: 92737},
	{Date: "2025-04-21", BTCAmount: 6556, USDAmount: 555800000, AvgPrice: 84785},
	{Date: "2025-04-14", BTCAmount: 3459, USDAmount: 285800000, AvgPrice: 82618},
	{Date: "2025-03-31", BTCAmount: 22048, USDAmount: 1920000000, AvgPrice: 86969},
	{Date: "2025-03-24", BTCAmount: 6911, USDAmount: 584100000, AvgPrice: 84529},
	{Date: "2025-03-17", BTCAmount: 130, USDAmount: 10700000, AvgPrice: 82981},
}

func main() {
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

	fmt.Println("=== MSTR Bitcoin Transaction Audit ===")
	fmt.Println()

	// Sort our transactions by date
	sort.Slice(ourData.Transactions, func(i, j int) bool {
		return ourData.Transactions[i].Date < ourData.Transactions[j].Date
	})

	// 1. Check for duplicate transactions
	fmt.Println("🔍 1. DUPLICATE TRANSACTION ANALYSIS:")
	duplicates := findDuplicates(ourData.Transactions)
	if len(duplicates) > 0 {
		fmt.Printf("Found %d potential duplicate groups:\n", len(duplicates))
		for i, group := range duplicates {
			fmt.Printf("Group %d:\n", i+1)
			for _, tx := range group {
				fmt.Printf("  - %s: %.0f BTC @ $%.0f (%s, %s)\n",
					tx.Date, tx.BTCAmount, tx.AvgPrice, tx.FilingType, tx.Source)
			}
			fmt.Println()
		}
	} else {
		fmt.Println("No obvious duplicates found")
	}
	fmt.Println()

	// 2. Check for cumulative vs incremental reporting
	fmt.Println("🔍 2. CUMULATIVE REPORTING ANALYSIS:")
	suspiciousLarge := findSuspiciouslyLargeTransactions(ourData.Transactions)
	if len(suspiciousLarge) > 0 {
		fmt.Printf("Found %d suspiciously large transactions (might be cumulative totals):\n", len(suspiciousLarge))
		for _, tx := range suspiciousLarge {
			fmt.Printf("  - %s: %.0f BTC for $%.0f M (%s, %s)\n",
				tx.Date, tx.BTCAmount, tx.USDAmount/1000000, tx.FilingType, tx.Source)
		}
	} else {
		fmt.Println("No suspiciously large transactions found")
	}
	fmt.Println()

	// 3. Compare with known SaylorTracker transactions
	fmt.Println("🔍 3. COMPARISON WITH SAYLORTRACKER DATA:")
	compareWithKnown(ourData.Transactions, knownTransactions)
	fmt.Println()

	// 4. Analyze transactions by filing type
	fmt.Println("🔍 4. FILING TYPE ANALYSIS:")
	analyzeByFilingType(ourData.Transactions)
	fmt.Println()

	// 5. Check for unrealistic price movements
	fmt.Println("🔍 5. PRICE ANOMALY ANALYSIS:")
	priceAnomalies := findPriceAnomalies(ourData.Transactions)
	if len(priceAnomalies) > 0 {
		fmt.Printf("Found %d transactions with unusual prices:\n", len(priceAnomalies))
		for _, tx := range priceAnomalies {
			fmt.Printf("  - %s: %.0f BTC @ $%.0f (%s)\n",
				tx.Date, tx.BTCAmount, tx.AvgPrice, tx.FilingType)
		}
	} else {
		fmt.Println("No obvious price anomalies found")
	}
	fmt.Println()

	// 6. Calculate running totals to identify where discrepancy starts
	fmt.Println("🔍 6. RUNNING TOTAL ANALYSIS:")
	analyzeRunningTotals(ourData.Transactions)
}

func findDuplicates(transactions []Transaction) [][]Transaction {
	var duplicates [][]Transaction
	seen := make(map[string][]Transaction)

	for _, tx := range transactions {
		key := fmt.Sprintf("%s-%.0f-%.0f", tx.Date, tx.BTCAmount, tx.USDAmount)
		seen[key] = append(seen[key], tx)
	}

	for _, group := range seen {
		if len(group) > 1 {
			duplicates = append(duplicates, group)
		}
	}

	return duplicates
}

func findSuspiciouslyLargeTransactions(transactions []Transaction) []Transaction {
	var suspicious []Transaction

	for _, tx := range transactions {
		// Flag transactions larger than 50,000 BTC as potentially cumulative
		if tx.BTCAmount > 50000 {
			suspicious = append(suspicious, tx)
		}
	}

	return suspicious
}

func compareWithKnown(ourTxs []Transaction, knownTxs []Transaction) {
	fmt.Println("Recent transactions comparison:")

	for _, known := range knownTxs {
		found := false
		for _, our := range ourTxs {
			if our.Date == known.Date &&
				abs(our.BTCAmount-known.BTCAmount) < 10 &&
				abs(our.AvgPrice-known.AvgPrice) < 1000 {
				fmt.Printf("✅ %s: MATCH - %.0f BTC @ $%.0f\n",
					known.Date, known.BTCAmount, known.AvgPrice)
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("❌ %s: MISSING - %.0f BTC @ $%.0f\n",
				known.Date, known.BTCAmount, known.AvgPrice)
		}
	}

	// Check for transactions we have that SaylorTracker doesn't
	fmt.Println("\nTransactions we have that might not be in SaylorTracker:")
	for _, our := range ourTxs {
		if our.Date >= "2025-03-01" {
			found := false
			for _, known := range knownTxs {
				if our.Date == known.Date &&
					abs(our.BTCAmount-known.BTCAmount) < 10 {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("⚠️  %s: %.0f BTC @ $%.0f (%s, %s)\n",
					our.Date, our.BTCAmount, our.AvgPrice, our.FilingType, our.Source)
			}
		}
	}
}

func analyzeByFilingType(transactions []Transaction) {
	filingStats := make(map[string]struct {
		count    int
		totalBTC float64
		totalUSD float64
	})

	for _, tx := range transactions {
		stats := filingStats[tx.FilingType]
		stats.count++
		stats.totalBTC += tx.BTCAmount
		stats.totalUSD += tx.USDAmount
		filingStats[tx.FilingType] = stats
	}

	for filingType, stats := range filingStats {
		fmt.Printf("%s: %d transactions, %.0f BTC, $%.1f B\n",
			filingType, stats.count, stats.totalBTC, stats.totalUSD/1000000000)
	}
}

func findPriceAnomalies(transactions []Transaction) []Transaction {
	var anomalies []Transaction

	for i, tx := range transactions {
		// Parse date to get approximate market price expectations
		date, err := time.Parse("2006-01-02", tx.Date)
		if err != nil {
			continue
		}

		year := date.Year()
		var expectedMin, expectedMax float64

		switch {
		case year <= 2020:
			expectedMin, expectedMax = 5000, 30000
		case year == 2021:
			expectedMin, expectedMax = 20000, 70000
		case year == 2022:
			expectedMin, expectedMax = 15000, 50000
		case year == 2023:
			expectedMin, expectedMax = 20000, 45000
		case year == 2024:
			expectedMin, expectedMax = 40000, 75000
		case year >= 2025:
			expectedMin, expectedMax = 80000, 120000
		}

		if tx.AvgPrice < expectedMin || tx.AvgPrice > expectedMax {
			anomalies = append(anomalies, tx)
		}

		// Also check for dramatic price changes between consecutive transactions
		if i > 0 {
			prevTx := transactions[i-1]
			priceChange := abs(tx.AvgPrice-prevTx.AvgPrice) / prevTx.AvgPrice
			if priceChange > 0.5 { // 50% price change
				// This might indicate parsing errors
			}
		}
	}

	return anomalies
}

func analyzeRunningTotals(transactions []Transaction) {
	runningBTC := 0.0
	runningUSD := 0.0

	fmt.Println("Year-by-year accumulation:")
	currentYear := ""
	yearBTC := 0.0
	yearUSD := 0.0

	for _, tx := range transactions {
		year := tx.Date[:4]
		if year != currentYear {
			if currentYear != "" {
				fmt.Printf("%s: +%.0f BTC, +$%.1f B (Total: %.0f BTC, $%.1f B)\n",
					currentYear, yearBTC, yearUSD/1000000000, runningBTC, runningUSD/1000000000)
			}
			currentYear = year
			yearBTC = 0
			yearUSD = 0
		}

		runningBTC += tx.BTCAmount
		runningUSD += tx.USDAmount
		yearBTC += tx.BTCAmount
		yearUSD += tx.USDAmount
	}

	if currentYear != "" {
		fmt.Printf("%s: +%.0f BTC, +$%.1f B (Total: %.0f BTC, $%.1f B)\n",
			currentYear, yearBTC, yearUSD/1000000000, runningBTC, runningUSD/1000000000)
	}

	fmt.Printf("\nFinal totals: %.0f BTC, $%.1f B\n", runningBTC, runningUSD/1000000000)
	fmt.Printf("SaylorTracker: 580,250 BTC, $40.6 B\n")
	fmt.Printf("Difference: %+.0f BTC, $%+.1f B\n",
		runningBTC-580250, (runningUSD-40610000000)/1000000000)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
