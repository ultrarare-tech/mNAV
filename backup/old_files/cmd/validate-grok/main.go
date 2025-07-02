package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/edgar"
)

// ValidationResult represents the results of comparing extraction methods
type ValidationResult struct {
	Filing              edgar.Filing               `json:"filing"`
	RegexTransactions   []edgar.BitcoinTransaction `json:"regex_transactions"`
	GrokTransactions    []edgar.BitcoinTransaction `json:"grok_transactions"`
	RegexCount          int                        `json:"regex_count"`
	GrokCount           int                        `json:"grok_count"`
	Agreement           bool                       `json:"agreement"`
	Differences         []string                   `json:"differences"`
	ProcessingTimeRegex time.Duration              `json:"processing_time_regex_ms"`
	ProcessingTimeGrok  time.Duration              `json:"processing_time_grok_ms"`
	GrokEnhancement     bool                       `json:"grok_enhancement"`
	FilePath            string                     `json:"file_path"`
}

// ValidationSummary provides overall statistics
type ValidationSummary struct {
	TotalFilings       int                `json:"total_filings"`
	FilingsWithBitcoin int                `json:"filings_with_bitcoin"`
	RegexTotal         int                `json:"regex_total_transactions"`
	GrokTotal          int                `json:"grok_total_transactions"`
	AgreementRate      float64            `json:"agreement_rate"`
	GrokEnhancements   int                `json:"grok_enhancements"`
	AvgProcessingRegex time.Duration      `json:"avg_processing_regex_ms"`
	AvgProcessingGrok  time.Duration      `json:"avg_processing_grok_ms"`
	Results            []ValidationResult `json:"results"`
}

func main() {
	var (
		ticker     = flag.String("ticker", "MSTR", "Company ticker symbol")
		maxFilings = flag.Int("max", 10, "Maximum number of filings to test")
		output     = flag.String("output", "validation_results.json", "Output file for results")
		verbose    = flag.Bool("verbose", false, "Verbose output")
		dataDir    = flag.String("data-dir", "data/edgar/companies", "Data directory path")
	)
	flag.Parse()

	// Check for Grok API key
	if os.Getenv("GROK_API_KEY") == "" {
		log.Fatal("GROK_API_KEY environment variable must be set")
	}

	fmt.Printf("🔍 Validating Grok Integration (Using Local Raw Filings)\n")
	fmt.Printf("========================================================\n")
	fmt.Printf("Ticker: %s\n", *ticker)
	fmt.Printf("Max Filings: %d\n", *maxFilings)
	fmt.Printf("Output: %s\n", *output)
	fmt.Printf("Data Directory: %s\n", *dataDir)
	fmt.Printf("%s\n\n", strings.Repeat("=", 50))

	// Create parsers
	client := edgar.NewClient("mNAV Validation 1.0 test@example.com")
	regexParser := edgar.NewDocumentParser(client)
	enhancedParser := edgar.NewEnhancedDocumentParser(client)

	// Get raw filing files
	rawFilingsDir := filepath.Join(*dataDir, *ticker, "raw_filings")
	fmt.Printf("📁 Looking for raw filings in: %s\n", rawFilingsDir)

	filingFiles, err := getRawFilingFiles(rawFilingsDir, *maxFilings)
	if err != nil {
		log.Fatalf("Error getting raw filing files: %v", err)
	}

	if len(filingFiles) == 0 {
		log.Fatal("No raw filing files found. Run edgar-enhanced first to download filings.")
	}

	fmt.Printf("Found %d raw filing files to validate\n\n", len(filingFiles))

	// Validation results
	var results []ValidationResult
	summary := ValidationSummary{
		TotalFilings: len(filingFiles),
		Results:      []ValidationResult{},
	}

	// Process each filing
	for i, filePath := range filingFiles {
		fmt.Printf("📋 Processing Filing %d/%d\n", i+1, len(filingFiles))
		fmt.Printf("   File: %s\n", filepath.Base(filePath))

		// Load raw filing content
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("   ❌ Error reading file: %v\n\n", err)
			continue
		}

		fmt.Printf("   Document size: %d bytes\n", len(content))

		// Create a mock filing object from filename
		filing := createFilingFromFilename(filepath.Base(filePath))

		// Test regex extraction
		fmt.Printf("   🔍 Testing regex extraction...\n")
		startTime := time.Now()
		regexTxns, err := regexParser.ParseHTMLDocument(content, filing)
		regexTime := time.Since(startTime)

		if err != nil {
			fmt.Printf("   ❌ Regex parsing error: %v\n", err)
			regexTxns = []edgar.BitcoinTransaction{}
		}

		// Test Grok extraction
		fmt.Printf("   🤖 Testing Grok extraction...\n")
		startTime = time.Now()
		grokTxns, err := enhancedParser.ParseHTMLDocumentEnhanced(content, filing)
		grokTime := time.Since(startTime)

		if err != nil {
			fmt.Printf("   ❌ Grok parsing error: %v\n", err)
			grokTxns = []edgar.BitcoinTransaction{}
		}

		// Compare results
		result := ValidationResult{
			Filing:              filing,
			RegexTransactions:   regexTxns,
			GrokTransactions:    grokTxns,
			RegexCount:          len(regexTxns),
			GrokCount:           len(grokTxns),
			ProcessingTimeRegex: regexTime,
			ProcessingTimeGrok:  grokTime,
			FilePath:            filePath,
		}

		// Analyze differences
		result.Agreement = len(regexTxns) == len(grokTxns)
		result.GrokEnhancement = len(grokTxns) > len(regexTxns)
		result.Differences = analyzeDifferences(regexTxns, grokTxns)

		// Display results
		fmt.Printf("   📊 Results:\n")
		fmt.Printf("      Regex: %d transactions (%.2fms)\n", len(regexTxns), float64(regexTime.Nanoseconds())/1e6)
		fmt.Printf("      Grok:  %d transactions (%.2fms)\n", len(grokTxns), float64(grokTime.Nanoseconds())/1e6)

		if result.GrokEnhancement {
			fmt.Printf("      ✨ Grok found %d additional transactions!\n", len(grokTxns)-len(regexTxns))
			if *verbose {
				for j := len(regexTxns); j < len(grokTxns); j++ {
					tx := grokTxns[j]
					fmt.Printf("        • %.2f BTC for $%.2f (confidence: %.2f)\n", tx.BTCPurchased, tx.USDSpent, tx.ConfidenceScore)
				}
			}
		} else if len(regexTxns) > len(grokTxns) {
			fmt.Printf("      ⚠️  Regex found %d more transactions\n", len(regexTxns)-len(grokTxns))
		} else if len(regexTxns) == len(grokTxns) && len(regexTxns) > 0 {
			fmt.Printf("      ✅ Both methods agree (%d transactions)\n", len(regexTxns))
		} else {
			fmt.Printf("      ℹ️  No Bitcoin transactions found\n")
		}

		if *verbose && len(result.Differences) > 0 {
			fmt.Printf("      Differences:\n")
			for _, diff := range result.Differences {
				fmt.Printf("        • %s\n", diff)
			}
		}

		fmt.Printf("\n")

		// Update summary
		if len(regexTxns) > 0 || len(grokTxns) > 0 {
			summary.FilingsWithBitcoin++
		}
		summary.RegexTotal += len(regexTxns)
		summary.GrokTotal += len(grokTxns)
		if result.GrokEnhancement {
			summary.GrokEnhancements++
		}
		summary.AvgProcessingRegex += regexTime
		summary.AvgProcessingGrok += grokTime

		results = append(results, result)
	}

	// Calculate final statistics
	summary.Results = results
	if len(results) > 0 {
		agreementCount := 0
		for _, r := range results {
			if r.Agreement {
				agreementCount++
			}
		}
		summary.AgreementRate = float64(agreementCount) / float64(len(results)) * 100
		summary.AvgProcessingRegex = summary.AvgProcessingRegex / time.Duration(len(results))
		summary.AvgProcessingGrok = summary.AvgProcessingGrok / time.Duration(len(results))
	}

	// Display summary
	displaySummary(summary)

	// Save results to file
	if err := saveResults(summary, *output); err != nil {
		log.Printf("Error saving results: %v", err)
	} else {
		fmt.Printf("💾 Results saved to %s\n", *output)
	}
}

// getRawFilingFiles gets a list of raw filing files from the directory
func getRawFilingFiles(dir string, maxFiles int) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only include HTML and XML files (not JSON metadata)
		if !info.IsDir() && (strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".xml")) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files by name (which includes date) and limit
	sort.Strings(files)
	if len(files) > maxFiles {
		files = files[:maxFiles]
	}

	return files, nil
}

// createFilingFromFilename creates a Filing object from the filename
func createFilingFromFilename(filename string) edgar.Filing {
	// Parse filename format: YYYY-MM-DD_TYPE_ACCESSION.ext
	parts := strings.Split(filename, "_")
	if len(parts) < 3 {
		return edgar.Filing{
			FilingType:      "Unknown",
			AccessionNumber: "Unknown",
			FilingDate:      time.Now(),
		}
	}

	dateStr := parts[0]
	filingType := parts[1]
	accessionPart := parts[2]
	accession := strings.Split(accessionPart, ".")[0] // Remove extension

	// Parse date
	filingDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		filingDate = time.Now()
	}

	return edgar.Filing{
		FilingType:      filingType,
		AccessionNumber: accession,
		FilingDate:      filingDate,
		URL:             "local://" + filename,
		DocumentURL:     "local://" + filename,
	}
}

func analyzeDifferences(regexTxns, grokTxns []edgar.BitcoinTransaction) []string {
	var differences []string

	if len(regexTxns) != len(grokTxns) {
		differences = append(differences, fmt.Sprintf("Count mismatch: Regex=%d, Grok=%d", len(regexTxns), len(grokTxns)))
	}

	// Compare transaction amounts if both found transactions
	if len(regexTxns) > 0 && len(grokTxns) > 0 {
		regexTotal := 0.0
		grokTotal := 0.0

		for _, tx := range regexTxns {
			regexTotal += tx.BTCPurchased
		}
		for _, tx := range grokTxns {
			grokTotal += tx.BTCPurchased
		}

		if regexTotal != grokTotal {
			differences = append(differences, fmt.Sprintf("BTC amount mismatch: Regex=%.2f, Grok=%.2f", regexTotal, grokTotal))
		}
	}

	// Check for Grok-specific extractions
	if len(grokTxns) > len(regexTxns) {
		for i := len(regexTxns); i < len(grokTxns); i++ {
			tx := grokTxns[i]
			if strings.HasPrefix(tx.ExtractedText, "[GROK]") {
				differences = append(differences, fmt.Sprintf("Grok-only extraction: %.2f BTC for $%.2f", tx.BTCPurchased, tx.USDSpent))
			}
		}
	}

	return differences
}

func displaySummary(summary ValidationSummary) {
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("📊 VALIDATION SUMMARY\n")
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("Total Filings Processed:     %d\n", summary.TotalFilings)
	fmt.Printf("Filings with Bitcoin Data:   %d (%.1f%%)\n", summary.FilingsWithBitcoin, float64(summary.FilingsWithBitcoin)/float64(summary.TotalFilings)*100)
	fmt.Printf("Agreement Rate:              %.1f%%\n", summary.AgreementRate)
	fmt.Printf("\n")

	fmt.Printf("📈 EXTRACTION RESULTS\n")
	fmt.Printf("Regex Total Transactions:    %d\n", summary.RegexTotal)
	fmt.Printf("Grok Total Transactions:     %d\n", summary.GrokTotal)
	fmt.Printf("Grok Enhancements:           %d filings\n", summary.GrokEnhancements)

	if summary.GrokTotal > summary.RegexTotal {
		improvement := float64(summary.GrokTotal-summary.RegexTotal) / float64(summary.RegexTotal) * 100
		fmt.Printf("Grok Improvement:            +%.1f%% more transactions\n", improvement)
	}
	fmt.Printf("\n")

	fmt.Printf("⏱️  PERFORMANCE\n")
	fmt.Printf("Avg Regex Processing:        %.2fms\n", float64(summary.AvgProcessingRegex.Nanoseconds())/1e6)
	fmt.Printf("Avg Grok Processing:         %.2fms\n", float64(summary.AvgProcessingGrok.Nanoseconds())/1e6)

	if summary.AvgProcessingGrok > summary.AvgProcessingRegex {
		slowdown := float64(summary.AvgProcessingGrok) / float64(summary.AvgProcessingRegex)
		fmt.Printf("Grok Overhead:               %.1fx slower\n", slowdown)
	}
	fmt.Printf("\n")

	// Show top enhancements
	enhancements := []ValidationResult{}
	for _, result := range summary.Results {
		if result.GrokEnhancement {
			enhancements = append(enhancements, result)
		}
	}

	if len(enhancements) > 0 {
		fmt.Printf("🚀 TOP GROK ENHANCEMENTS\n")
		sort.Slice(enhancements, func(i, j int) bool {
			return (enhancements[i].GrokCount - enhancements[i].RegexCount) > (enhancements[j].GrokCount - enhancements[j].RegexCount)
		})

		for i, result := range enhancements {
			if i >= 3 { // Show top 3
				break
			}
			enhancement := result.GrokCount - result.RegexCount
			fmt.Printf("  %d. %s (%s): +%d transactions\n",
				i+1, result.Filing.FilingType, result.Filing.FilingDate.Format("2006-01-02"), enhancement)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("💡 RECOMMENDATIONS\n")
	if summary.GrokEnhancements > 0 {
		fmt.Printf("✅ Grok integration is providing value with %d enhanced extractions\n", summary.GrokEnhancements)
	}
	if summary.AgreementRate > 80 {
		fmt.Printf("✅ High agreement rate (%.1f%%) indicates consistent results\n", summary.AgreementRate)
	} else {
		fmt.Printf("⚠️  Lower agreement rate (%.1f%%) - review differences for accuracy\n", summary.AgreementRate)
	}
	if summary.GrokTotal > summary.RegexTotal {
		fmt.Printf("🎯 Grok found %d additional transactions - consider using for production\n", summary.GrokTotal-summary.RegexTotal)
	}
	fmt.Printf("%s\n", strings.Repeat("=", 60))
}

func saveResults(summary ValidationSummary, filename string) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
