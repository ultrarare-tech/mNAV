package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// SaylorTrackerData represents the data from SaylorTracker website
type SaylorTrackerData struct {
	TotalBTC          float64 `json:"totalBtc"`
	TotalUSD          float64 `json:"totalUsd"`
	AveragePrice      float64 `json:"averagePrice"`
	CurrentBTCPrice   float64 `json:"currentBtcPrice"`
	CurrentValue      float64 `json:"currentValue"`
	SharesOutstanding float64 `json:"sharesOutstanding"`
	StockPrice        float64 `json:"stockPrice"`
	MarketCap         float64 `json:"marketCap"`
	NAVPremium        float64 `json:"navPremium"`
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

type OurAnalysis struct {
	TotalBTC          float64 `json:"totalBtc"`
	TotalUSD          float64 `json:"totalUsd"`
	AveragePrice      float64 `json:"averagePrice"`
	TotalTransactions int     `json:"totalTransactions"`
	FirstTransaction  string  `json:"firstTransaction"`
	LastTransaction   string  `json:"lastTransaction"`
}

type ComparisonResult struct {
	GeneratedAt   time.Time         `json:"generatedAt"`
	SaylorTracker SaylorTrackerData `json:"saylorTracker"`
	OurExtraction OurAnalysis       `json:"ourExtraction"`
	Discrepancies Discrepancies     `json:"discrepancies"`
	Analysis      string            `json:"analysis"`
}

type Discrepancies struct {
	BTCDifference          float64 `json:"btcDifference"`
	BTCDifferencePercent   float64 `json:"btcDifferencePercent"`
	USDDifference          float64 `json:"usdDifference"`
	USDDifferencePercent   float64 `json:"usdDifferencePercent"`
	PriceDifference        float64 `json:"priceDifference"`
	PriceDifferencePercent float64 `json:"priceDifferencePercent"`
}

func main() {
	fmt.Printf("🔍 SAYLOR TRACKER COMPARISON ANALYSIS\n")
	fmt.Printf("=====================================\n\n")

	// SaylorTracker.com data (as of the search results)
	saylorData := SaylorTrackerData{
		TotalBTC:          580250,
		TotalUSD:          40610000000,  // $40.61 billion
		AveragePrice:      69979,        // $69,979 per BTC
		CurrentBTCPrice:   106153.57,    // $106,153.57
		CurrentValue:      61658538201,  // $61.66 billion
		SharesOutstanding: 256990000,    // 256.99M shares
		StockPrice:        370.63,       // $370.63
		MarketCap:         101330000000, // $101.33B
		NAVPremium:        1.64,         // 1.64x
	}

	// Load our analysis
	ourData, err := loadOurAnalysis("data/analysis/MSTR_comprehensive_bitcoin_analysis.json")
	if err != nil {
		log.Fatalf("❌ Error loading our analysis: %v", err)
	}

	// Calculate discrepancies
	discrepancies := calculateDiscrepancies(saylorData, ourData)

	// Create comparison result
	comparison := ComparisonResult{
		GeneratedAt:   time.Now(),
		SaylorTracker: saylorData,
		OurExtraction: ourData,
		Discrepancies: discrepancies,
		Analysis:      generateAnalysis(discrepancies),
	}

	// Display results
	displayComparison(comparison)

	// Save comparison
	if err := saveComparison(comparison, "data/analysis/saylor_tracker_comparison.json"); err != nil {
		log.Printf("⚠️  Warning: Could not save comparison: %v", err)
	} else {
		fmt.Printf("\n💾 Comparison saved to: data/analysis/saylor_tracker_comparison.json\n")
	}
}

func loadOurAnalysis(filePath string) (OurAnalysis, error) {
	var analysis struct {
		Summary struct {
			TotalBTC         float64 `json:"totalBtc"`
			TotalUSD         float64 `json:"totalUsd"`
			AveragePrice     float64 `json:"averagePrice"`
			FirstTransaction string  `json:"firstTransaction"`
			LastTransaction  string  `json:"lastTransaction"`
		} `json:"summary"`
		TotalTransactions int `json:"totalTransactions"`
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return OurAnalysis{}, err
	}

	if err := json.Unmarshal(data, &analysis); err != nil {
		return OurAnalysis{}, err
	}

	return OurAnalysis{
		TotalBTC:          analysis.Summary.TotalBTC,
		TotalUSD:          analysis.Summary.TotalUSD,
		AveragePrice:      analysis.Summary.AveragePrice,
		TotalTransactions: analysis.TotalTransactions,
		FirstTransaction:  analysis.Summary.FirstTransaction,
		LastTransaction:   analysis.Summary.LastTransaction,
	}, nil
}

func calculateDiscrepancies(saylor SaylorTrackerData, our OurAnalysis) Discrepancies {
	btcDiff := our.TotalBTC - saylor.TotalBTC
	btcDiffPercent := (btcDiff / saylor.TotalBTC) * 100

	usdDiff := our.TotalUSD - saylor.TotalUSD
	usdDiffPercent := (usdDiff / saylor.TotalUSD) * 100

	priceDiff := our.AveragePrice - saylor.AveragePrice
	priceDiffPercent := (priceDiff / saylor.AveragePrice) * 100

	return Discrepancies{
		BTCDifference:          btcDiff,
		BTCDifferencePercent:   btcDiffPercent,
		USDDifference:          usdDiff,
		USDDifferencePercent:   usdDiffPercent,
		PriceDifference:        priceDiff,
		PriceDifferencePercent: priceDiffPercent,
	}
}

func generateAnalysis(disc Discrepancies) string {
	analysis := "DISCREPANCY ANALYSIS:\n\n"

	if disc.BTCDifferencePercent > 10 {
		analysis += "🚨 SIGNIFICANT BTC DISCREPANCY: Our extraction shows significantly more BTC than SaylorTracker. "
		analysis += "This suggests potential over-counting, possibly due to:\n"
		analysis += "- Cumulative totals being misinterpreted as individual purchases\n"
		analysis += "- 10-K annual reports containing summary data rather than new transactions\n"
		analysis += "- Duplicate transactions across different filing types\n\n"
	} else if disc.BTCDifferencePercent > 5 {
		analysis += "⚠️  MODERATE BTC DISCREPANCY: Some over-counting detected. Review needed.\n\n"
	} else {
		analysis += "✅ BTC COUNTS ALIGN: Good agreement between sources.\n\n"
	}

	if disc.USDDifferencePercent > 10 {
		analysis += "💰 SIGNIFICANT USD DISCREPANCY: Investment amounts differ substantially.\n\n"
	}

	if disc.PriceDifferencePercent > 10 {
		analysis += "📊 AVERAGE PRICE DISCREPANCY: Different calculation methods or data sources.\n\n"
	}

	analysis += "RECOMMENDATIONS:\n"
	analysis += "1. Review 10-K filings for cumulative vs. individual transaction data\n"
	analysis += "2. Cross-reference with Michael Saylor's Twitter announcements\n"
	analysis += "3. Implement duplicate detection across filing types\n"
	analysis += "4. Validate against on-chain Bitcoin wallet data if available"

	return analysis
}

func displayComparison(comp ComparisonResult) {
	fmt.Printf("📊 COMPARISON RESULTS\n")
	fmt.Printf("====================\n\n")

	fmt.Printf("🌐 SAYLOR TRACKER DATA (saylortracker.com):\n")
	fmt.Printf("   Total BTC: %.0f BTC\n", comp.SaylorTracker.TotalBTC)
	fmt.Printf("   Total Invested: $%.2f billion\n", comp.SaylorTracker.TotalUSD/1e9)
	fmt.Printf("   Average Price: $%.2f per BTC\n", comp.SaylorTracker.AveragePrice)
	fmt.Printf("   Current BTC Price: $%.2f\n", comp.SaylorTracker.CurrentBTCPrice)
	fmt.Printf("   Current Portfolio Value: $%.2f billion\n", comp.SaylorTracker.CurrentValue/1e9)
	fmt.Printf("   Stock Price: $%.2f\n", comp.SaylorTracker.StockPrice)
	fmt.Printf("   Market Cap: $%.2f billion\n", comp.SaylorTracker.MarketCap/1e9)
	fmt.Printf("   NAV Premium: %.2fx\n", comp.SaylorTracker.NAVPremium)

	fmt.Printf("\n🔍 OUR EXTRACTION DATA:\n")
	fmt.Printf("   Total BTC: %.0f BTC\n", comp.OurExtraction.TotalBTC)
	fmt.Printf("   Total Invested: $%.2f billion\n", comp.OurExtraction.TotalUSD/1e9)
	fmt.Printf("   Average Price: $%.2f per BTC\n", comp.OurExtraction.AveragePrice)
	fmt.Printf("   Total Transactions: %d\n", comp.OurExtraction.TotalTransactions)
	fmt.Printf("   First Transaction: %s\n", comp.OurExtraction.FirstTransaction)
	fmt.Printf("   Last Transaction: %s\n", comp.OurExtraction.LastTransaction)

	fmt.Printf("\n⚖️  DISCREPANCIES:\n")
	fmt.Printf("   BTC Difference: %+.0f BTC (%+.1f%%)\n",
		comp.Discrepancies.BTCDifference, comp.Discrepancies.BTCDifferencePercent)
	fmt.Printf("   USD Difference: %+$.2f billion (%+.1f%%)\n",
		comp.Discrepancies.USDDifference/1e9, comp.Discrepancies.USDDifferencePercent)
	fmt.Printf("   Price Difference: %+$.2f per BTC (%+.1f%%)\n",
		comp.Discrepancies.PriceDifference, comp.Discrepancies.PriceDifferencePercent)

	fmt.Printf("\n📋 ANALYSIS:\n")
	fmt.Printf("%s\n", comp.Analysis)
}

func saveComparison(comp ComparisonResult, filePath string) error {
	data, err := json.MarshalIndent(comp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
