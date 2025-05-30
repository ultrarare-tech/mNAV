package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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

	fmt.Println("=== 10-K vs 8-K Filing Analysis ===")
	fmt.Println()

	// Separate transactions by filing type
	var tenKTransactions []Transaction
	var eightKTransactions []Transaction
	var tenQTransactions []Transaction

	for _, tx := range ourData.Transactions {
		switch tx.FilingType {
		case "10-K":
			tenKTransactions = append(tenKTransactions, tx)
		case "8-K":
			eightKTransactions = append(eightKTransactions, tx)
		case "10-Q":
			tenQTransactions = append(tenQTransactions, tx)
		}
	}

	// Sort by date
	sort.Slice(tenKTransactions, func(i, j int) bool {
		return tenKTransactions[i].Date < tenKTransactions[j].Date
	})
	sort.Slice(eightKTransactions, func(i, j int) bool {
		return eightKTransactions[i].Date < eightKTransactions[j].Date
	})
	sort.Slice(tenQTransactions, func(i, j int) bool {
		return tenQTransactions[i].Date < tenQTransactions[j].Date
	})

	fmt.Printf("📊 FILING TYPE BREAKDOWN:\n")
	fmt.Printf("10-K (Annual Reports):    %d transactions, %.0f BTC, $%.1f B\n",
		len(tenKTransactions), sumBTC(tenKTransactions), sumUSD(tenKTransactions)/1000000000)
	fmt.Printf("8-K (Current Reports):    %d transactions, %.0f BTC, $%.1f B\n",
		len(eightKTransactions), sumBTC(eightKTransactions), sumUSD(eightKTransactions)/1000000000)
	fmt.Printf("10-Q (Quarterly Reports): %d transactions, %.0f BTC, $%.1f B\n",
		len(tenQTransactions), sumBTC(tenQTransactions), sumUSD(tenQTransactions)/1000000000)
	fmt.Println()

	// Analyze 10-K transactions in detail
	fmt.Println("🔍 DETAILED 10-K ANALYSIS:")
	if len(tenKTransactions) == 0 {
		fmt.Println("No 10-K transactions found")
	} else {
		for i, tx := range tenKTransactions {
			fmt.Printf("\n%d. 📅 %s: %.0f BTC for $%.0f M @ $%.0f/BTC (%s)\n",
				i+1, tx.Date, tx.BTCAmount, tx.USDAmount/1000000, tx.AvgPrice, tx.Source)

			// Check if this transaction has a corresponding 8-K
			hasCorresponding8K := false
			var corresponding8Ks []Transaction

			// Look for 8-K filings within 30 days before the 10-K
			for _, eightK := range eightKTransactions {
				if eightK.Date <= tx.Date &&
					daysBetween(eightK.Date, tx.Date) <= 30 &&
					abs(eightK.BTCAmount-tx.BTCAmount) < 1000 { // Allow some tolerance
					hasCorresponding8K = true
					corresponding8Ks = append(corresponding8Ks, eightK)
				}
			}

			if hasCorresponding8K {
				fmt.Printf("   ✅ Has corresponding 8-K filing(s):\n")
				for _, corr := range corresponding8Ks {
					fmt.Printf("      - %s: %.0f BTC @ $%.0f\n",
						corr.Date, corr.BTCAmount, corr.AvgPrice)
				}
			} else {
				fmt.Printf("   ❓ NO corresponding 8-K found - potentially unique information\n")

				// Check if it's a reasonable purchase amount or likely cumulative
				if tx.BTCAmount > 50000 {
					fmt.Printf("   ⚠️  LIKELY CUMULATIVE: Very large amount suggests total holdings\n")
				} else if tx.AvgPrice < 5000 || tx.AvgPrice > 150000 {
					fmt.Printf("   ⚠️  SUSPICIOUS PRICE: Unrealistic price suggests parsing error\n")
				} else {
					fmt.Printf("   💡 POTENTIAL UNIQUE PURCHASE: Reasonable amount and price\n")
				}
			}

			// Provide SEC link
			fmt.Printf("   🔗 SEC Filing: https://www.sec.gov/edgar/search/#/dateRange=custom&startdt=%s&enddt=%s&entityName=microstrategy&forms=10-K\n",
				tx.Date, tx.Date)
		}
	}
	fmt.Println()

	// Check for 10-Q transactions as well
	fmt.Println("🔍 DETAILED 10-Q ANALYSIS:")
	if len(tenQTransactions) == 0 {
		fmt.Println("No 10-Q transactions found")
	} else {
		for i, tx := range tenQTransactions {
			fmt.Printf("\n%d. 📅 %s: %.0f BTC for $%.0f M @ $%.0f/BTC (%s)\n",
				i+1, tx.Date, tx.BTCAmount, tx.USDAmount/1000000, tx.AvgPrice, tx.Source)

			// Check for suspicious characteristics
			if tx.BTCAmount > 50000 {
				fmt.Printf("   ⚠️  LIKELY CUMULATIVE: Very large amount suggests total holdings\n")
			} else if tx.AvgPrice < 5000 || tx.AvgPrice > 150000 {
				fmt.Printf("   ⚠️  SUSPICIOUS PRICE: Unrealistic price suggests parsing error\n")
			} else {
				fmt.Printf("   💡 POTENTIAL VALID PURCHASE: Reasonable amount and price\n")
			}

			fmt.Printf("   🔗 SEC Filing: https://www.sec.gov/edgar/search/#/dateRange=custom&startdt=%s&enddt=%s&entityName=microstrategy&forms=10-Q\n",
				tx.Date, tx.Date)
		}
	}
	fmt.Println()

	// Timeline analysis
	fmt.Println("📅 TIMELINE ANALYSIS:")
	fmt.Println("Looking for gaps where 10-K/10-Q might have unique information...")

	// Create a timeline of all transactions
	allTransactions := append(append(tenKTransactions, eightKTransactions...), tenQTransactions...)
	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].Date < allTransactions[j].Date
	})

	// Group by year and analyze
	yearGroups := make(map[string][]Transaction)
	for _, tx := range allTransactions {
		year := tx.Date[:4]
		yearGroups[year] = append(yearGroups[year], tx)
	}

	for year := 2020; year <= 2025; year++ {
		yearStr := fmt.Sprintf("%d", year)
		if txs, exists := yearGroups[yearStr]; exists {
			fmt.Printf("\n%s:\n", yearStr)

			var year8K, year10K, year10Q []Transaction
			for _, tx := range txs {
				switch tx.FilingType {
				case "8-K":
					year8K = append(year8K, tx)
				case "10-K":
					year10K = append(year10K, tx)
				case "10-Q":
					year10Q = append(year10Q, tx)
				}
			}

			fmt.Printf("  8-K:  %d transactions, %.0f BTC\n", len(year8K), sumBTC(year8K))
			fmt.Printf("  10-K: %d transactions, %.0f BTC\n", len(year10K), sumBTC(year10K))
			fmt.Printf("  10-Q: %d transactions, %.0f BTC\n", len(year10Q), sumBTC(year10Q))

			// Check for years with only 10-K/10-Q transactions
			if len(year8K) == 0 && (len(year10K) > 0 || len(year10Q) > 0) {
				fmt.Printf("  ⚠️  NO 8-K transactions - 10-K/10-Q might contain unique info\n")
			}
		}
	}

	fmt.Println()
	fmt.Println("💡 CONCLUSIONS:")
	fmt.Println("1. 10-K filings often contain cumulative totals rather than individual purchases")
	fmt.Println("2. Most legitimate purchases are announced via 8-K filings first")
	fmt.Println("3. 10-Q filings may contain quarterly summaries or catch-up information")
	fmt.Println("4. Large amounts (>50,000 BTC) in 10-K/10-Q are likely cumulative totals")
	fmt.Println("5. Unrealistic prices indicate parsing errors in cumulative reporting")
}

func sumBTC(transactions []Transaction) float64 {
	total := 0.0
	for _, tx := range transactions {
		total += tx.BTCAmount
	}
	return total
}

func sumUSD(transactions []Transaction) float64 {
	total := 0.0
	for _, tx := range transactions {
		total += tx.USDAmount
	}
	return total
}

func daysBetween(date1, date2 string) int {
	// Simple day calculation (this is approximate)
	// In production, you'd use proper date parsing
	if date1 > date2 {
		return 0
	}

	// Extract year, month, day
	y1, m1, d1 := parseDate(date1)
	y2, m2, d2 := parseDate(date2)

	// Simple approximation
	days1 := y1*365 + m1*30 + d1
	days2 := y2*365 + m2*30 + d2

	return days2 - days1
}

func parseDate(date string) (int, int, int) {
	// Parse YYYY-MM-DD format
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return 0, 0, 0
	}

	year := parseInt(parts[0])
	month := parseInt(parts[1])
	day := parseInt(parts[2])

	return year, month, day
}

func parseInt(s string) int {
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
