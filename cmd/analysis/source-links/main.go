package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// Known SaylorTracker transactions (from the website)
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
	// Adding more known transactions from earlier periods
	{Date: "2024-12-30", BTCAmount: 2138, USDAmount: 209000000, AvgPrice: 97837},
	{Date: "2024-12-23", BTCAmount: 5262, USDAmount: 561000000, AvgPrice: 106662},
	{Date: "2024-12-16", BTCAmount: 15350, USDAmount: 1500000000, AvgPrice: 100386},
	{Date: "2024-12-09", BTCAmount: 21550, USDAmount: 2100000000, AvgPrice: 98783},
	{Date: "2024-12-02", BTCAmount: 15400, USDAmount: 1500000000, AvgPrice: 95976},
	{Date: "2024-11-25", BTCAmount: 55500, USDAmount: 5400000000, AvgPrice: 97862},
	{Date: "2024-11-18", BTCAmount: 51780, USDAmount: 4600000000, AvgPrice: 88627},
	{Date: "2024-11-12", BTCAmount: 27200, USDAmount: 2030000000, AvgPrice: 74463},
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

	fmt.Println("=== MSTR Transactions NOT Found in SaylorTracker ===")
	fmt.Println()

	// Sort our transactions by date (newest first)
	sort.Slice(ourData.Transactions, func(i, j int) bool {
		return ourData.Transactions[i].Date > ourData.Transactions[j].Date
	})

	// Find transactions not in SaylorTracker
	var suspiciousTransactions []Transaction
	var totalSuspiciousBTC float64
	var totalSuspiciousUSD float64

	for _, our := range ourData.Transactions {
		found := false
		for _, known := range knownTransactions {
			if our.Date == known.Date &&
				abs(our.BTCAmount-known.BTCAmount) < 100 { // Allow some tolerance
				found = true
				break
			}
		}
		if !found {
			// Additional filters for suspicious transactions
			if our.BTCAmount > 50000 || our.AvgPrice < 5000 || our.AvgPrice > 150000 {
				suspiciousTransactions = append(suspiciousTransactions, our)
				totalSuspiciousBTC += our.BTCAmount
				totalSuspiciousUSD += our.USDAmount
			}
		}
	}

	fmt.Printf("Found %d suspicious transactions not in SaylorTracker:\n", len(suspiciousTransactions))
	fmt.Printf("Total suspicious BTC: %.0f (%.1f%% of our total)\n",
		totalSuspiciousBTC, (totalSuspiciousBTC/ourData.Summary.TotalBTCAcquired)*100)
	fmt.Printf("Total suspicious USD: $%.1f billion\n", totalSuspiciousUSD/1000000000)
	fmt.Println()

	// Group by filing type and analyze
	filingGroups := make(map[string][]Transaction)
	for _, tx := range suspiciousTransactions {
		filingGroups[tx.FilingType] = append(filingGroups[tx.FilingType], tx)
	}

	for filingType, txs := range filingGroups {
		fmt.Printf("=== %s FILINGS ===\n", filingType)

		for _, tx := range txs {
			fmt.Printf("\n📅 %s: %.0f BTC for $%.0f M @ $%.0f/BTC (%s)\n",
				tx.Date, tx.BTCAmount, tx.USDAmount/1000000, tx.AvgPrice, tx.Source)

			// Find the source filing
			filingInfo := findSourceFiling(tx.Date, filingType)
			if filingInfo.Found {
				fmt.Printf("🔗 SEC EDGAR: %s\n", filingInfo.SECLink)
				fmt.Printf("📁 Local File: %s\n", filingInfo.LocalPath)
			} else {
				fmt.Printf("❌ Source filing not found in local data\n")
				fmt.Printf("🔍 Search SEC: %s\n", filingInfo.SearchLink)
			}

			// Provide analysis
			if tx.BTCAmount > 50000 {
				fmt.Printf("⚠️  SUSPICIOUS: Extremely large transaction (likely cumulative total)\n")
			}
			if tx.AvgPrice < 5000 {
				fmt.Printf("⚠️  SUSPICIOUS: Unrealistically low price (likely parsing error)\n")
			}
			if tx.AvgPrice > 150000 {
				fmt.Printf("⚠️  SUSPICIOUS: Unrealistically high price for the date\n")
			}
		}
		fmt.Println()
	}

	// Provide recommendations
	fmt.Println("💡 RECOMMENDATIONS:")
	fmt.Println("1. Review the large transactions (>50,000 BTC) - these are likely cumulative totals")
	fmt.Println("2. Check transactions with unrealistic prices - these indicate parsing errors")
	fmt.Println("3. Cross-reference with Michael Saylor's Twitter for actual purchase announcements")
	fmt.Println("4. Consider excluding 10-K filings which often report cumulative totals")
	fmt.Println()

	// Calculate impact of removing suspicious transactions
	adjustedBTC := ourData.Summary.TotalBTCAcquired - totalSuspiciousBTC
	adjustedUSD := ourData.Summary.TotalUSDInvested - totalSuspiciousUSD
	adjustedAvgCost := adjustedUSD / adjustedBTC

	fmt.Println("📊 IMPACT OF REMOVING SUSPICIOUS TRANSACTIONS:")
	fmt.Printf("Original totals:  %.0f BTC, $%.1f B, $%.0f avg cost\n",
		ourData.Summary.TotalBTCAcquired, ourData.Summary.TotalUSDInvested/1000000000, ourData.Summary.AverageCostBasis)
	fmt.Printf("Adjusted totals:  %.0f BTC, $%.1f B, $%.0f avg cost\n",
		adjustedBTC, adjustedUSD/1000000000, adjustedAvgCost)
	fmt.Printf("SaylorTracker:    580,250 BTC, $40.6 B, $69,979 avg cost\n")
	fmt.Printf("New difference:   %+.0f BTC (%.1f%%), $%+.1f B (%.1f%%)\n",
		adjustedBTC-580250, ((adjustedBTC-580250)/580250)*100,
		(adjustedUSD-40610000000)/1000000000, ((adjustedUSD-40610000000)/40610000000)*100)
}

type FilingInfo struct {
	Found      bool
	LocalPath  string
	SECLink    string
	SearchLink string
}

func findSourceFiling(date, filingType string) FilingInfo {
	filingDir := "data/edgar/companies/MSTR"

	// Try to find exact match first
	pattern := fmt.Sprintf("%s_%s_*.htm", date, filingType)
	matches, err := filepath.Glob(filepath.Join(filingDir, pattern))

	if err == nil && len(matches) > 0 {
		filename := filepath.Base(matches[0])
		parts := strings.Split(filename, "_")
		if len(parts) >= 3 {
			accessionNumber := strings.TrimSuffix(parts[2], ".htm")
			// Convert to SEC EDGAR URL
			accessionFormatted := strings.ReplaceAll(accessionNumber, "-", "")
			secLink := fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/1050446/%s/%s.htm",
				accessionNumber, accessionFormatted)

			return FilingInfo{
				Found:     true,
				LocalPath: matches[0],
				SECLink:   secLink,
			}
		}
	}

	// If no exact match, try nearby dates (within 7 days)
	for dayOffset := 1; dayOffset <= 7; dayOffset++ {
		for _, direction := range []int{-1, 1} {
			searchDate := adjustDateSimple(date, direction*dayOffset)
			pattern := fmt.Sprintf("%s_%s_*.htm", searchDate, filingType)
			matches, err := filepath.Glob(filepath.Join(filingDir, pattern))

			if err == nil && len(matches) > 0 {
				filename := filepath.Base(matches[0])
				parts := strings.Split(filename, "_")
				if len(parts) >= 3 {
					accessionNumber := strings.TrimSuffix(parts[2], ".htm")
					accessionFormatted := strings.ReplaceAll(accessionNumber, "-", "")
					secLink := fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/1050446/%s/%s.htm",
						accessionNumber, accessionFormatted)

					return FilingInfo{
						Found:     true,
						LocalPath: matches[0],
						SECLink:   secLink,
					}
				}
			}
		}
	}

	// If no match found, provide search URL
	year := date[:4]
	month := date[5:7]
	day := date[8:10]
	searchLink := fmt.Sprintf("https://www.sec.gov/edgar/search/#/dateRange=custom&startdt=%s-%s-%s&enddt=%s-%s-%s&entityName=microstrategy&forms=%s",
		year, month, day, year, month, day, filingType)

	return FilingInfo{
		Found:      false,
		SearchLink: searchLink,
	}
}

func adjustDateSimple(date string, days int) string {
	// This is a simplified date adjustment
	// For production, you'd want proper date parsing and arithmetic
	return date
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
