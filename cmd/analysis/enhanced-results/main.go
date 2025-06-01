package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

type TransactionSummary struct {
	Date        string  `json:"date"`
	BTCAmount   float64 `json:"btc_amount"`
	USDAmount   float64 `json:"usd_amount"`
	AvgPrice    float64 `json:"avg_price"`
	FilingType  string  `json:"filing_type"`
	Source      string  `json:"source"`
	Method      string  `json:"parsing_method"`
	Confidence  float64 `json:"confidence"`
	ExtractText string  `json:"extract_text"`
}

type ComparisonResult struct {
	GeneratedAt        time.Time            `json:"generatedAt"`
	EnhancedResults    AnalysisResult       `json:"enhancedResults"`
	PreviousResults    AnalysisResult       `json:"previousResults"`
	Improvements       ImprovementAnalysis  `json:"improvements"`
	TransactionDetails []TransactionSummary `json:"transactionDetails"`
}

type AnalysisResult struct {
	TotalTransactions int            `json:"totalTransactions"`
	TotalBTC          float64        `json:"totalBtc"`
	TotalUSD          float64        `json:"totalUsd"`
	AveragePrice      float64        `json:"averagePrice"`
	FirstTransaction  string         `json:"firstTransaction"`
	LastTransaction   string         `json:"lastTransaction"`
	FilingBreakdown   map[string]int `json:"filingBreakdown"`
}

type ImprovementAnalysis struct {
	TransactionReduction int     `json:"transactionReduction"`
	BTCReduction         float64 `json:"btcReduction"`
	USDReduction         float64 `json:"usdReduction"`
	ReductionPercentage  float64 `json:"reductionPercentage"`
	CumulativeFiltered   int     `json:"cumulativeFiltered"`
	Analysis             string  `json:"analysis"`
}

func main() {
	fmt.Printf("🔍 ENHANCED PARSING RESULTS ANALYSIS\n")
	fmt.Printf("====================================\n\n")

	// Load current enhanced results
	enhancedResults, err := loadEnhancedResults("data/parsed")
	if err != nil {
		log.Fatalf("❌ Error loading enhanced results: %v", err)
	}

	// Load previous comprehensive results for comparison
	previousResults, err := loadPreviousResults("data/analysis/MSTR_comprehensive_bitcoin_analysis.json")
	if err != nil {
		log.Printf("⚠️  Warning: Could not load previous results for comparison: %v", err)
		previousResults = AnalysisResult{}
	}

	// Analyze improvements
	improvements := analyzeImprovements(enhancedResults, previousResults)

	// Create comparison
	comparison := ComparisonResult{
		GeneratedAt:        time.Now(),
		EnhancedResults:    enhancedResults,
		PreviousResults:    previousResults,
		Improvements:       improvements,
		TransactionDetails: extractTransactionDetails("data/parsed"),
	}

	// Display results
	displayComparison(comparison)

	// Save analysis
	if err := saveComparison(comparison, "data/analysis/enhanced_parsing_comparison.json"); err != nil {
		log.Printf("⚠️  Warning: Could not save comparison: %v", err)
	} else {
		fmt.Printf("\n💾 Analysis saved to: data/analysis/enhanced_parsing_comparison.json\n")
	}
}

func loadEnhancedResults(parsedDir string) (AnalysisResult, error) {
	var allTransactions []models.BitcoinTransaction
	filingBreakdown := make(map[string]int)

	err := filepath.Walk(parsedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, "_parsed.json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var result models.FilingParseResult
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}

		// Count transactions by filing type
		if len(result.BitcoinTransactions) > 0 {
			filingBreakdown[result.Filing.FilingType] += len(result.BitcoinTransactions)
		}

		allTransactions = append(allTransactions, result.BitcoinTransactions...)
		return nil
	})

	if err != nil {
		return AnalysisResult{}, err
	}

	// Sort transactions by date
	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].Date.Before(allTransactions[j].Date)
	})

	// Calculate totals
	var totalBTC, totalUSD float64
	for _, tx := range allTransactions {
		totalBTC += tx.BTCPurchased
		totalUSD += tx.USDSpent
	}

	var avgPrice float64
	if totalBTC > 0 {
		avgPrice = totalUSD / totalBTC
	}

	var firstTx, lastTx string
	if len(allTransactions) > 0 {
		firstTx = allTransactions[0].Date.Format("2006-01-02")
		lastTx = allTransactions[len(allTransactions)-1].Date.Format("2006-01-02")
	}

	return AnalysisResult{
		TotalTransactions: len(allTransactions),
		TotalBTC:          totalBTC,
		TotalUSD:          totalUSD,
		AveragePrice:      avgPrice,
		FirstTransaction:  firstTx,
		LastTransaction:   lastTx,
		FilingBreakdown:   filingBreakdown,
	}, nil
}

func loadPreviousResults(filePath string) (AnalysisResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return AnalysisResult{}, err
	}

	var analysis struct {
		Summary struct {
			TotalBTC         float64 `json:"totalBtc"`
			TotalUSD         float64 `json:"totalUsd"`
			AveragePrice     float64 `json:"averagePrice"`
			FirstTransaction string  `json:"firstTransaction"`
			LastTransaction  string  `json:"lastTransaction"`
		} `json:"summary"`
		TotalTransactions int            `json:"totalTransactions"`
		FilingBreakdown   map[string]int `json:"filingBreakdown"`
	}

	if err := json.Unmarshal(data, &analysis); err != nil {
		return AnalysisResult{}, err
	}

	return AnalysisResult{
		TotalTransactions: analysis.TotalTransactions,
		TotalBTC:          analysis.Summary.TotalBTC,
		TotalUSD:          analysis.Summary.TotalUSD,
		AveragePrice:      analysis.Summary.AveragePrice,
		FirstTransaction:  analysis.Summary.FirstTransaction,
		LastTransaction:   analysis.Summary.LastTransaction,
		FilingBreakdown:   analysis.FilingBreakdown,
	}, nil
}

func analyzeImprovements(enhanced, previous AnalysisResult) ImprovementAnalysis {
	txReduction := previous.TotalTransactions - enhanced.TotalTransactions
	btcReduction := previous.TotalBTC - enhanced.TotalBTC
	usdReduction := previous.TotalUSD - enhanced.TotalUSD

	var reductionPercentage float64
	if previous.TotalTransactions > 0 {
		reductionPercentage = (float64(txReduction) / float64(previous.TotalTransactions)) * 100
	}

	analysis := generateImprovementAnalysis(txReduction, btcReduction, reductionPercentage)

	return ImprovementAnalysis{
		TransactionReduction: txReduction,
		BTCReduction:         btcReduction,
		USDReduction:         usdReduction,
		ReductionPercentage:  reductionPercentage,
		Analysis:             analysis,
	}
}

func generateImprovementAnalysis(txReduction int, btcReduction float64, reductionPercentage float64) string {
	analysis := "CUMULATIVE DETECTION IMPACT:\n\n"

	if txReduction > 0 {
		analysis += fmt.Sprintf("✅ SUCCESSFUL FILTERING: Removed %d transactions (%.1f%% reduction)\n",
			txReduction, reductionPercentage)
		analysis += fmt.Sprintf("💰 BTC REDUCTION: %.0f BTC removed from over-counting\n", btcReduction)
		analysis += "🎯 ACCURACY IMPROVEMENT: Enhanced parser successfully identified and excluded cumulative totals\n\n"

		if reductionPercentage > 30 {
			analysis += "🚨 SIGNIFICANT IMPROVEMENT: Large reduction suggests previous over-counting was substantial\n"
		} else if reductionPercentage > 10 {
			analysis += "⚠️  MODERATE IMPROVEMENT: Good reduction in over-counting\n"
		} else {
			analysis += "✅ MINOR IMPROVEMENT: Small but meaningful reduction\n"
		}
	} else if txReduction == 0 {
		analysis += "🔄 NO CHANGE: Same number of transactions detected\n"
		analysis += "📊 This could indicate either perfect accuracy or need for further refinement\n"
	} else {
		analysis += "⚠️  UNEXPECTED: More transactions found than before\n"
		analysis += "🔍 This requires investigation - may indicate improved detection of valid transactions\n"
	}

	return analysis
}

func extractTransactionDetails(parsedDir string) []TransactionSummary {
	var details []TransactionSummary

	filepath.Walk(parsedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, "_parsed.json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var result models.FilingParseResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil
		}

		for _, tx := range result.BitcoinTransactions {
			details = append(details, TransactionSummary{
				Date:        tx.Date.Format("2006-01-02"),
				BTCAmount:   tx.BTCPurchased,
				USDAmount:   tx.USDSpent,
				AvgPrice:    tx.AvgPriceUSD,
				FilingType:  tx.FilingType,
				Source:      filepath.Base(path),
				Method:      result.ParsingMethod,
				Confidence:  tx.ConfidenceScore,
				ExtractText: truncateText(tx.ExtractedText, 100),
			})
		}

		return nil
	})

	// Sort by date
	sort.Slice(details, func(i, j int) bool {
		return details[i].Date < details[j].Date
	})

	return details
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func displayComparison(comp ComparisonResult) {
	fmt.Printf("📊 ENHANCED PARSING COMPARISON\n")
	fmt.Printf("==============================\n\n")

	fmt.Printf("🔍 ENHANCED RESULTS (with cumulative detection):\n")
	fmt.Printf("   Total Transactions: %d\n", comp.EnhancedResults.TotalTransactions)
	fmt.Printf("   Total BTC: %.0f BTC\n", comp.EnhancedResults.TotalBTC)
	fmt.Printf("   Total USD: $%.2f billion\n", comp.EnhancedResults.TotalUSD/1e9)
	fmt.Printf("   Average Price: $%.2f per BTC\n", comp.EnhancedResults.AveragePrice)
	fmt.Printf("   Date Range: %s to %s\n", comp.EnhancedResults.FirstTransaction, comp.EnhancedResults.LastTransaction)

	if comp.PreviousResults.TotalTransactions > 0 {
		fmt.Printf("\n📋 PREVIOUS RESULTS (before cumulative detection):\n")
		fmt.Printf("   Total Transactions: %d\n", comp.PreviousResults.TotalTransactions)
		fmt.Printf("   Total BTC: %.0f BTC\n", comp.PreviousResults.TotalBTC)
		fmt.Printf("   Total USD: $%.2f billion\n", comp.PreviousResults.TotalUSD/1e9)
		fmt.Printf("   Average Price: $%.2f per BTC\n", comp.PreviousResults.AveragePrice)

		fmt.Printf("\n⚖️  IMPROVEMENTS:\n")
		fmt.Printf("   Transaction Reduction: %d (%.1f%%)\n",
			comp.Improvements.TransactionReduction, comp.Improvements.ReductionPercentage)
		fmt.Printf("   BTC Reduction: %.0f BTC\n", comp.Improvements.BTCReduction)
		fmt.Printf("   USD Reduction: $%.2f billion\n", comp.Improvements.USDReduction/1e9)
	}

	fmt.Printf("\n📋 FILING BREAKDOWN:\n")
	for filingType, count := range comp.EnhancedResults.FilingBreakdown {
		fmt.Printf("   %s: %d transactions\n", filingType, count)
	}

	fmt.Printf("\n📋 ANALYSIS:\n")
	fmt.Printf("%s\n", comp.Improvements.Analysis)

	if len(comp.TransactionDetails) > 0 {
		fmt.Printf("\n🔍 RECENT TRANSACTIONS (last 5):\n")
		start := len(comp.TransactionDetails) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(comp.TransactionDetails); i++ {
			tx := comp.TransactionDetails[i]
			fmt.Printf("   %s: %.0f BTC for $%.0f (avg: $%.0f) [%s]\n",
				tx.Date, tx.BTCAmount, tx.USDAmount, tx.AvgPrice, tx.FilingType)
		}
	}
}

func saveComparison(comp ComparisonResult, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(comp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
