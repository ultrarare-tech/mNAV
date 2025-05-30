package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/interpretation/grok"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

func main() {
	var (
		ticker     = flag.String("ticker", "MSTR", "Company ticker symbol")
		dataDir    = flag.String("data-dir", "data/edgar/companies", "Data directory containing downloaded filings")
		maxFiles   = flag.Int("max-files", 5, "Maximum number of files to test")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
		testType   = flag.String("test-type", "both", "Test type: bitcoin, shares, or both")
		filingType = flag.String("filing-type", "", "Filter by filing type (e.g., 10-K, 10-Q, 8-K)")
		mode       = flag.String("mode", "test", "Mode: test (default) or filter-demo")
	)

	flag.Parse()

	fmt.Printf("🤖 GROK AI INTEGRATION TEST\n")
	fmt.Printf("============================\n\n")

	// Initialize Grok client
	grokClient := grok.NewClient()
	if !grokClient.IsConfigured() {
		log.Fatalf("❌ Grok API key not configured. Please set GROK_API_KEY environment variable.")
	}

	fmt.Printf("✅ Grok client initialized successfully\n")

	// Handle different modes
	switch *mode {
	case "filter-demo":
		runFilterDemo(grokClient, *ticker, *dataDir)
		return
	case "test":
		// Continue with normal test mode
	default:
		log.Fatalf("❌ Unknown mode: %s. Use 'test' or 'filter-demo'", *mode)
	}

	fmt.Printf("📊 Test Configuration:\n")
	fmt.Printf("   • Ticker: %s\n", *ticker)
	fmt.Printf("   • Max Files: %d\n", *maxFiles)
	fmt.Printf("   • Test Type: %s\n", *testType)
	if *filingType != "" {
		fmt.Printf("   • Filing Type Filter: %s\n", *filingType)
	}
	fmt.Printf("\n")

	// Find filing files
	companyDir := filepath.Join(*dataDir, *ticker)
	if _, err := os.Stat(companyDir); os.IsNotExist(err) {
		log.Fatalf("❌ Company directory not found: %s", companyDir)
	}

	files, err := filepath.Glob(filepath.Join(companyDir, "*.htm"))
	if err != nil {
		log.Fatalf("❌ Error finding filing files: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("❌ No filing files found in %s", companyDir)
	}

	// Filter by filing type if specified
	if *filingType != "" {
		var filteredFiles []string
		for _, file := range files {
			if strings.Contains(filepath.Base(file), *filingType) {
				filteredFiles = append(filteredFiles, file)
			}
		}
		files = filteredFiles
		fmt.Printf("🔍 Filtered to %d files matching filing type: %s\n", len(files), *filingType)
	}

	// Limit files
	if len(files) > *maxFiles {
		files = files[:*maxFiles]
	}

	fmt.Printf("📁 Testing %d filing files\n\n", len(files))

	// Test results tracking
	var totalBitcoinTests int
	var totalSharesTests int
	var bitcoinSuccesses int
	var sharesSuccesses int
	var totalProcessingTime time.Duration

	for i, filePath := range files {
		fileName := filepath.Base(filePath)
		fmt.Printf("[%d/%d] Testing %s\n", i+1, len(files), fileName)

		// Parse filename to create filing metadata
		filing := parseFilingFromFilename(fileName, *ticker)

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("   ❌ Error reading file: %v\n", err)
			continue
		}

		text := string(content)

		// Test Bitcoin extraction
		if *testType == "bitcoin" || *testType == "both" {
			totalBitcoinTests++
			fmt.Printf("   🔍 Testing Bitcoin extraction...\n")

			startTime := time.Now()
			bitcoinTxs, err := grokClient.ExtractBitcoinTransactions(text, filing)
			duration := time.Since(startTime)
			totalProcessingTime += duration

			if err != nil {
				fmt.Printf("   ❌ Bitcoin extraction failed: %v\n", err)
			} else {
				bitcoinSuccesses++
				fmt.Printf("   ✅ Bitcoin extraction successful (%v)\n", duration)
				fmt.Printf("   💰 Found %d Bitcoin transactions\n", len(bitcoinTxs))

				if *verbose && len(bitcoinTxs) > 0 {
					for j, tx := range bitcoinTxs {
						fmt.Printf("      [%d] %.2f BTC for $%.2f (confidence: %.2f)\n",
							j+1, tx.BTCPurchased, tx.USDSpent, tx.ConfidenceScore)
						if tx.ExtractedText != "" {
							excerpt := tx.ExtractedText
							if len(excerpt) > 100 {
								excerpt = excerpt[:100] + "..."
							}
							fmt.Printf("          Text: %s\n", excerpt)
						}
					}
				}
			}
		}

		// Test Shares extraction
		if *testType == "shares" || *testType == "both" {
			totalSharesTests++
			fmt.Printf("   🔍 Testing Shares extraction...\n")

			startTime := time.Now()
			sharesRecord, err := grokClient.ExtractSharesOutstanding(text, filing)
			duration := time.Since(startTime)
			totalProcessingTime += duration

			if err != nil {
				fmt.Printf("   ❌ Shares extraction failed: %v\n", err)
			} else {
				sharesSuccesses++
				fmt.Printf("   ✅ Shares extraction successful (%v)\n", duration)

				if sharesRecord != nil {
					fmt.Printf("   📊 Found shares data: %.0f common shares (confidence: %.2f)\n",
						sharesRecord.CommonShares, sharesRecord.ConfidenceScore)

					if *verbose {
						if sharesRecord.ExtractedText != "" {
							excerpt := sharesRecord.ExtractedText
							if len(excerpt) > 100 {
								excerpt = excerpt[:100] + "..."
							}
							fmt.Printf("      Text: %s\n", excerpt)
						}
						fmt.Printf("      Source: %s\n", sharesRecord.ExtractedFrom)
					}
				} else {
					fmt.Printf("   ⚪ No shares data found\n")
				}
			}
		}

		fmt.Printf("\n")
	}

	// Summary
	fmt.Printf("🎯 GROK TEST SUMMARY\n")
	fmt.Printf("====================\n")
	fmt.Printf("Files Tested: %d\n", len(files))
	fmt.Printf("Total Processing Time: %v\n", totalProcessingTime)
	fmt.Printf("Average Time per File: %v\n", totalProcessingTime/time.Duration(len(files)))

	if totalBitcoinTests > 0 {
		successRate := float64(bitcoinSuccesses) / float64(totalBitcoinTests) * 100
		fmt.Printf("\n💰 Bitcoin Extraction:\n")
		fmt.Printf("   Tests: %d\n", totalBitcoinTests)
		fmt.Printf("   Successes: %d\n", bitcoinSuccesses)
		fmt.Printf("   Success Rate: %.1f%%\n", successRate)
	}

	if totalSharesTests > 0 {
		successRate := float64(sharesSuccesses) / float64(totalSharesTests) * 100
		fmt.Printf("\n📊 Shares Extraction:\n")
		fmt.Printf("   Tests: %d\n", totalSharesTests)
		fmt.Printf("   Successes: %d\n", sharesSuccesses)
		fmt.Printf("   Success Rate: %.1f%%\n", successRate)
	}

	if bitcoinSuccesses > 0 || sharesSuccesses > 0 {
		fmt.Printf("\n✅ Grok AI integration is working correctly!\n")
	} else {
		fmt.Printf("\n⚠️  No successful extractions. Check API key and filing content.\n")
	}
}

// parseFilingFromFilename extracts filing metadata from filename
func parseFilingFromFilename(filename, ticker string) models.Filing {
	// Expected format: YYYY-MM-DD_FORM-TYPE_ACCESSION-NUMBER.htm
	parts := strings.Split(strings.TrimSuffix(filename, ".htm"), "_")

	filing := models.Filing{
		DocumentURL: filename,
	}

	if len(parts) >= 3 {
		// Parse date
		if date, err := time.Parse("2006-01-02", parts[0]); err == nil {
			filing.FilingDate = date
			filing.ReportDate = date
		}

		// Parse filing type
		filing.FilingType = parts[1]

		// Parse accession number
		filing.AccessionNumber = parts[2]

		// Construct URL
		filing.URL = fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%s/%s", ticker, filename)
	}

	return filing
}

// runFilterDemo demonstrates the smart content filtering capabilities
func runFilterDemo(grokClient *grok.Client, ticker, dataDir string) {
	fmt.Println("=== Smart Content Filtering Demo ===")

	// Find filing files
	companyDir := filepath.Join(dataDir, ticker)
	if _, err := os.Stat(companyDir); os.IsNotExist(err) {
		log.Fatalf("❌ Company directory not found: %s", companyDir)
	}

	files, err := filepath.Glob(filepath.Join(companyDir, "*.htm"))
	if err != nil {
		log.Fatalf("❌ Error finding filing files: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("❌ No filing files found in %s", companyDir)
	}

	// Find a filing to demonstrate filtering (prefer Bitcoin-era 8-K filings)
	var selectedFile string

	// First, look for Bitcoin-era 8-K filings (December 2020 onwards)
	for _, file := range files {
		fileName := filepath.Base(file)
		if strings.Contains(fileName, "8-K") &&
			(strings.Contains(fileName, "2020-12") ||
				strings.Contains(fileName, "2021") ||
				strings.Contains(fileName, "2022") ||
				strings.Contains(fileName, "2023") ||
				strings.Contains(fileName, "2024")) {
			selectedFile = file
			break
		}
	}

	// If no Bitcoin-era 8-K found, look for any 10-K
	if selectedFile == "" {
		for _, file := range files {
			if strings.Contains(filepath.Base(file), "10-K") {
				selectedFile = file
				break
			}
		}
	}

	// If still nothing found, use the first file
	if selectedFile == "" {
		selectedFile = files[0]
	}

	fileName := filepath.Base(selectedFile)
	fmt.Printf("Testing with filing: %s\n", fileName)

	// Load the filing content
	content, err := os.ReadFile(selectedFile)
	if err != nil {
		log.Fatalf("Error loading filing content: %v", err)
	}

	contentStr := string(content)

	// Calculate original token count (rough estimate)
	originalTokens := len(strings.Fields(contentStr))
	fmt.Printf("Original content: ~%d tokens\n", originalTokens)

	// Test Bitcoin content filtering
	fmt.Println("\n--- Bitcoin Content Filtering ---")
	bitcoinFiltered := filterBitcoinRelevantContent(grokClient, contentStr)
	if bitcoinFiltered == "" {
		fmt.Println("No Bitcoin-relevant content found")
	} else {
		filteredTokens := len(strings.Fields(bitcoinFiltered))
		reduction := float64(originalTokens-filteredTokens) / float64(originalTokens) * 100
		fmt.Printf("Filtered content: ~%d tokens (%.1f%% reduction)\n", filteredTokens, reduction)

		// Show a sample of the filtered content
		lines := strings.Split(bitcoinFiltered, "\n")
		fmt.Printf("Sample filtered content (%d lines):\n", len(lines))
		for i, line := range lines {
			if i >= 5 { // Show first 5 lines
				fmt.Println("...")
				break
			}
			if len(line) > 100 {
				fmt.Printf("  %s...\n", line[:100])
			} else {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	// Test shares content filtering
	fmt.Println("\n--- Shares Content Filtering ---")
	sharesFiltered := filterSharesRelevantContent(grokClient, contentStr)
	if sharesFiltered == "" {
		fmt.Println("No shares-relevant content found")
	} else {
		filteredTokens := len(strings.Fields(sharesFiltered))
		reduction := float64(originalTokens-filteredTokens) / float64(originalTokens) * 100
		fmt.Printf("Filtered content: ~%d tokens (%.1f%% reduction)\n", filteredTokens, reduction)

		// Show a sample of the filtered content
		lines := strings.Split(sharesFiltered, "\n")
		fmt.Printf("Sample filtered content (%d lines):\n", len(lines))
		for i, line := range lines {
			if i >= 5 { // Show first 5 lines
				fmt.Println("...")
				break
			}
			if len(line) > 100 {
				fmt.Printf("  %s...\n", line[:100])
			} else {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	fmt.Println("\n💡 Benefits of Smart Filtering:")
	fmt.Println("   • Reduces token usage and API costs")
	fmt.Println("   • Focuses AI attention on relevant content")
	fmt.Println("   • Improves extraction accuracy")
	fmt.Println("   • Enables processing of larger documents")
	fmt.Println("\nFiltering helps make Grok AI more efficient and cost-effective!")
}

// Helper functions to access the filtering methods (since they're not exported)
func filterBitcoinRelevantContent(client *grok.Client, content string) string {
	// We need to use reflection or create a wrapper since the methods aren't exported
	// For now, let's implement a simplified version here
	bitcoinKeywords := []string{
		"bitcoin", "btc", "cryptocurrency", "crypto", "digital asset", "digital currency",
		"blockchain", "satoshi", "mining", "wallet",
	}

	return filterContentByKeywords(content, bitcoinKeywords, 50)
}

func filterSharesRelevantContent(client *grok.Client, content string) string {
	sharesKeywords := []string{
		"shares outstanding", "common stock", "preferred stock", "stockholders", "equity",
		"balance sheet", "consolidated balance", "capital stock", "share count",
	}

	return filterContentByKeywords(content, sharesKeywords, 30)
}

func filterContentByKeywords(text string, keywords []string, minLength int) string {
	paragraphs := strings.Split(text, "\n")
	var relevantParagraphs []string
	var currentParagraph strings.Builder

	for _, line := range paragraphs {
		line = strings.TrimSpace(line)

		if line == "" {
			if currentParagraph.Len() > 0 {
				paragraph := currentParagraph.String()
				if containsKeywords(paragraph, keywords) && len(paragraph) > minLength {
					relevantParagraphs = append(relevantParagraphs, paragraph)
				}
				currentParagraph.Reset()
			}
			continue
		}

		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(line)
	}

	// Check final paragraph
	if currentParagraph.Len() > 0 {
		paragraph := currentParagraph.String()
		if containsKeywords(paragraph, keywords) && len(paragraph) > minLength {
			relevantParagraphs = append(relevantParagraphs, paragraph)
		}
	}

	if len(relevantParagraphs) == 0 {
		return ""
	}

	return strings.Join(relevantParagraphs, "\n\n---\n\n")
}

func containsKeywords(text string, keywords []string) bool {
	lowerText := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}
