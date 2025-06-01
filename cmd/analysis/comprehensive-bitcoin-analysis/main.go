package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

type ComprehensiveAnalysis struct {
	GeneratedAt           time.Time                    `json:"generatedAt"`
	TotalFilingsProcessed int                          `json:"totalFilingsProcessed"`
	FilingsWithBitcoin    int                          `json:"filingsWithBitcoin"`
	TotalTransactions     int                          `json:"totalTransactions"`
	AllTransactions       []models.BitcoinTransaction  `json:"allTransactions"`
	Summary               TransactionSummary           `json:"summary"`
	ByFilingType          map[string]FilingTypeSummary `json:"byFilingType"`
	Timeline              []TimelineEntry              `json:"timeline"`
}

type TransactionSummary struct {
	TotalBTC         float64 `json:"totalBtc"`
	TotalUSD         float64 `json:"totalUsd"`
	AveragePrice     float64 `json:"averagePrice"`
	FirstTransaction string  `json:"firstTransaction"`
	LastTransaction  string  `json:"lastTransaction"`
	LargestBTCAmount float64 `json:"largestBtcAmount"`
	LargestUSDAmount float64 `json:"largestUsdAmount"`
}

type FilingTypeSummary struct {
	Count        int     `json:"count"`
	TotalBTC     float64 `json:"totalBtc"`
	TotalUSD     float64 `json:"totalUsd"`
	AveragePrice float64 `json:"averagePrice"`
}

type TimelineEntry struct {
	Date          string  `json:"date"`
	BTCAmount     float64 `json:"btcAmount"`
	USDAmount     float64 `json:"usdAmount"`
	AvgPrice      float64 `json:"avgPrice"`
	FilingType    string  `json:"filingType"`
	CumulativeBTC float64 `json:"cumulativeBtc"`
}

func main() {
	var (
		ticker    = flag.String("ticker", "MSTR", "Company ticker symbol")
		dataDir   = flag.String("data-dir", "data/edgar/companies", "Data directory containing downloaded filings")
		outputDir = flag.String("output-dir", "data/analysis", "Output directory for analysis results")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
		useGrok   = flag.Bool("grok", true, "Enable Grok AI enhancement for parsing")
		reparse   = flag.Bool("reparse", false, "Force reparsing of all filings")
	)

	flag.Parse()

	fmt.Printf("🔍 COMPREHENSIVE BITCOIN ANALYSIS\n")
	fmt.Printf("==================================\n\n")

	// Step 1: Run the bitcoin parser if needed
	if *reparse {
		fmt.Printf("🔄 Step 1: Running Bitcoin Transaction Parser...\n")
		if err := runBitcoinParser(*ticker, *dataDir, *verbose, *useGrok); err != nil {
			log.Fatalf("❌ Error running bitcoin parser: %v", err)
		}
		fmt.Printf("✅ Bitcoin parsing complete\n\n")
	}

	// Step 2: Collect all parsed results
	fmt.Printf("📊 Step 2: Collecting parsed results...\n")
	allResults, err := collectParsedResults()
	if err != nil {
		log.Fatalf("❌ Error collecting results: %v", err)
	}

	fmt.Printf("📁 Found %d parsed filing results\n", len(allResults))

	// Step 3: Analyze and aggregate
	fmt.Printf("🔬 Step 3: Analyzing transactions...\n")
	analysis := analyzeTransactions(allResults)

	// Step 4: Save comprehensive analysis
	outputFile := filepath.Join(*outputDir, fmt.Sprintf("%s_comprehensive_bitcoin_analysis.json", *ticker))
	if err := saveAnalysis(analysis, outputFile); err != nil {
		log.Fatalf("❌ Error saving analysis: %v", err)
	}

	// Step 5: Display summary
	displaySummary(analysis)

	fmt.Printf("\n💾 Comprehensive analysis saved to: %s\n", outputFile)
	fmt.Printf("✅ Analysis complete!\n")
}

func runBitcoinParser(ticker, dataDir string, verbose, useGrok bool) error {
	args := []string{
		"-ticker=" + ticker,
		"-data-dir=" + dataDir,
	}

	if verbose {
		args = append(args, "-verbose")
	}
	if useGrok {
		args = append(args, "-grok")
	}

	// Set environment variable for Grok API key
	cmd := exec.Command("./bin/interpretation/bitcoin-parser", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GROK_API_KEY=%s", os.Getenv("GROK_API_KEY")))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func collectParsedResults() ([]models.FilingParseResult, error) {
	var results []models.FilingParseResult

	parsedDir := "data/parsed"
	files, err := filepath.Glob(filepath.Join(parsedDir, "*_parsed.json"))
	if err != nil {
		return nil, fmt.Errorf("error finding parsed files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Warning: Could not read %s: %v", file, err)
			continue
		}

		var result models.FilingParseResult
		if err := json.Unmarshal(data, &result); err != nil {
			log.Printf("Warning: Could not parse %s: %v", file, err)
			continue
		}

		// Only include results with Bitcoin transactions
		if len(result.BitcoinTransactions) > 0 {
			results = append(results, result)
		}
	}

	return results, nil
}

func analyzeTransactions(results []models.FilingParseResult) *ComprehensiveAnalysis {
	analysis := &ComprehensiveAnalysis{
		GeneratedAt:           time.Now(),
		TotalFilingsProcessed: len(results),
		ByFilingType:          make(map[string]FilingTypeSummary),
	}

	var allTransactions []models.BitcoinTransaction
	filingTypeStats := make(map[string]*FilingTypeSummary)

	// Collect all transactions
	for _, result := range results {
		if len(result.BitcoinTransactions) > 0 {
			analysis.FilingsWithBitcoin++
			allTransactions = append(allTransactions, result.BitcoinTransactions...)

			// Track by filing type
			filingType := result.Filing.FilingType
			if filingType == "" {
				filingType = "Unknown"
			}

			if _, exists := filingTypeStats[filingType]; !exists {
				filingTypeStats[filingType] = &FilingTypeSummary{}
			}

			for _, tx := range result.BitcoinTransactions {
				filingTypeStats[filingType].Count++
				filingTypeStats[filingType].TotalBTC += tx.BTCPurchased
				filingTypeStats[filingType].TotalUSD += tx.USDSpent
			}
		}
	}

	analysis.TotalTransactions = len(allTransactions)
	analysis.AllTransactions = allTransactions

	// Calculate summary statistics
	if len(allTransactions) > 0 {
		summary := TransactionSummary{}

		var totalBTC, totalUSD float64
		var firstDate, lastDate time.Time
		var largestBTC, largestUSD float64

		for i, tx := range allTransactions {
			totalBTC += tx.BTCPurchased
			totalUSD += tx.USDSpent

			if tx.BTCPurchased > largestBTC {
				largestBTC = tx.BTCPurchased
			}
			if tx.USDSpent > largestUSD {
				largestUSD = tx.USDSpent
			}

			if i == 0 || tx.Date.Before(firstDate) {
				firstDate = tx.Date
			}
			if i == 0 || tx.Date.After(lastDate) {
				lastDate = tx.Date
			}
		}

		summary.TotalBTC = totalBTC
		summary.TotalUSD = totalUSD
		if totalBTC > 0 {
			summary.AveragePrice = totalUSD / totalBTC
		}
		summary.FirstTransaction = firstDate.Format("2006-01-02")
		summary.LastTransaction = lastDate.Format("2006-01-02")
		summary.LargestBTCAmount = largestBTC
		summary.LargestUSDAmount = largestUSD

		analysis.Summary = summary
	}

	// Calculate filing type averages
	for filingType, stats := range filingTypeStats {
		if stats.TotalBTC > 0 {
			stats.AveragePrice = stats.TotalUSD / stats.TotalBTC
		}
		analysis.ByFilingType[filingType] = *stats
	}

	// Create timeline
	analysis.Timeline = createTimeline(allTransactions)

	return analysis
}

func createTimeline(transactions []models.BitcoinTransaction) []TimelineEntry {
	// Sort transactions by date
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date.Before(transactions[j].Date)
	})

	var timeline []TimelineEntry
	var cumulativeBTC float64

	for _, tx := range transactions {
		cumulativeBTC += tx.BTCPurchased

		entry := TimelineEntry{
			Date:          tx.Date.Format("2006-01-02"),
			BTCAmount:     tx.BTCPurchased,
			USDAmount:     tx.USDSpent,
			AvgPrice:      tx.AvgPriceUSD,
			FilingType:    tx.FilingType,
			CumulativeBTC: cumulativeBTC,
		}
		timeline = append(timeline, entry)
	}

	return timeline
}

func saveAnalysis(analysis *ComprehensiveAnalysis, outputFile string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling analysis: %w", err)
	}

	return os.WriteFile(outputFile, data, 0644)
}

func displaySummary(analysis *ComprehensiveAnalysis) {
	fmt.Printf("\n📊 COMPREHENSIVE BITCOIN ANALYSIS SUMMARY\n")
	fmt.Printf("==========================================\n")
	fmt.Printf("Total Filings Processed: %d\n", analysis.TotalFilingsProcessed)
	fmt.Printf("Filings with Bitcoin: %d\n", analysis.FilingsWithBitcoin)
	fmt.Printf("Total Transactions: %d\n", analysis.TotalTransactions)
	fmt.Printf("\n💰 TRANSACTION SUMMARY\n")
	fmt.Printf("Total BTC Acquired: %.2f BTC\n", analysis.Summary.TotalBTC)
	fmt.Printf("Total USD Invested: $%.2f\n", analysis.Summary.TotalUSD)
	fmt.Printf("Average Price: $%.2f per BTC\n", analysis.Summary.AveragePrice)
	fmt.Printf("First Transaction: %s\n", analysis.Summary.FirstTransaction)
	fmt.Printf("Last Transaction: %s\n", analysis.Summary.LastTransaction)
	fmt.Printf("Largest BTC Purchase: %.2f BTC\n", analysis.Summary.LargestBTCAmount)
	fmt.Printf("Largest USD Purchase: $%.2f\n", analysis.Summary.LargestUSDAmount)

	fmt.Printf("\n📋 BY FILING TYPE\n")
	for filingType, stats := range analysis.ByFilingType {
		fmt.Printf("%s: %d transactions, %.2f BTC, $%.2f (avg: $%.2f)\n",
			filingType, stats.Count, stats.TotalBTC, stats.TotalUSD, stats.AveragePrice)
	}

	fmt.Printf("\n📈 RECENT TRANSACTIONS\n")
	recentCount := 5
	if len(analysis.Timeline) < recentCount {
		recentCount = len(analysis.Timeline)
	}

	for i := len(analysis.Timeline) - recentCount; i < len(analysis.Timeline); i++ {
		entry := analysis.Timeline[i]
		fmt.Printf("%s: %.2f BTC for $%.2f (cumulative: %.2f BTC)\n",
			entry.Date, entry.BTCAmount, entry.USDAmount, entry.CumulativeBTC)
	}
}
