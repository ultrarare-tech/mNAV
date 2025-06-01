package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/interpretation/grok"
	"github.com/jeffreykibler/mNAV/pkg/interpretation/parser"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

func main() {
	var (
		ticker     = flag.String("ticker", "", "Company ticker symbol (required)")
		dataDir    = flag.String("data-dir", "data/edgar/companies", "Data directory containing downloaded filings")
		outputDir  = flag.String("output-dir", "data/parsed", "Output directory for parsed results")
		dryRun     = flag.Bool("dry-run", false, "Show what would be processed without actually parsing")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
		useGrok    = flag.Bool("grok", false, "Enable Grok AI enhancement for parsing")
		maxFiles   = flag.Int("max-files", 0, "Maximum number of files to process (0 = all)")
		filingType = flag.String("filing-type", "", "Filter by filing type (e.g., 10-K, 10-Q, 8-K)")
	)

	flag.Parse()

	fmt.Printf("🔍 DATA INTERPRETATION - Bitcoin Transaction Parser\n")
	fmt.Printf("==================================================\n\n")

	if *ticker == "" {
		fmt.Println("❌ Error: ticker is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize Grok client if requested
	var grokClient *grok.Client
	if *useGrok {
		grokClient = grok.NewClient()
		if !grokClient.IsConfigured() {
			fmt.Println("⚠️  Warning: Grok API key not configured, falling back to regex-only mode")
			grokClient = nil
		}
	}

	// Initialize enhanced parser
	enhancedParser := parser.NewEnhancedParser(grokClient, *verbose)

	// Show parser configuration
	fmt.Printf("📊 Parser Configuration:\n")
	fmt.Printf("   • Grok Enabled: %v\n", *useGrok)
	fmt.Printf("   • Grok Configured: %v\n", grokClient != nil && grokClient.IsConfigured())
	fmt.Printf("   • Verbose Mode: %v\n", *verbose)
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

	// Limit files if maxFiles is specified
	if *maxFiles > 0 && len(files) > *maxFiles {
		files = files[:*maxFiles]
		fmt.Printf("📊 Limited to %d files (max-files setting)\n", len(files))
	}

	fmt.Printf("📁 Found %d filing files to process\n\n", len(files))

	if *dryRun {
		fmt.Printf("🔍 DRY RUN - Files that would be processed:\n")
		for i, file := range files {
			fmt.Printf("[%d] %s\n", i+1, filepath.Base(file))
		}
		fmt.Printf("\n✅ Dry run complete. Use without -dry-run to actually process files.\n")
		return
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	// Process files
	var totalTransactions int
	var totalSharesRecords int
	var processedFiles int
	var errorFiles int

	startTime := time.Now()

	for i, filePath := range files {
		fileName := filepath.Base(filePath)
		fmt.Printf("[%d/%d] Processing %s... ", i+1, len(files), fileName)

		// Parse filename to create filing metadata
		filing := parseFilingFromFilename(fileName, *ticker)

		// Read file content
		content, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("❌ Error reading file: %v\n", err)
			errorFiles++
			continue
		}

		// Read content as string
		contentBytes, err := io.ReadAll(content)
		content.Close()
		if err != nil {
			fmt.Printf("❌ Error reading file content: %v\n", err)
			errorFiles++
			continue
		}

		// Parse the filing
		result, err := enhancedParser.ParseFiling(string(contentBytes), filing.FilingType, filePath)

		if err != nil {
			fmt.Printf("❌ Error parsing: %v\n", err)
			errorFiles++
			continue
		}

		// Count results
		btcCount := len(result.BitcoinTransactions)
		sharesCount := 0
		if result.SharesOutstanding != nil {
			sharesCount = 1
		}

		totalTransactions += btcCount
		totalSharesRecords += sharesCount
		processedFiles++

		// Show results
		if btcCount > 0 || sharesCount > 0 {
			fmt.Printf("✅ Found %d BTC transactions, %d shares record (%dms)\n",
				btcCount, sharesCount, result.ProcessingTimeMs)

			if *verbose {
				for _, tx := range result.BitcoinTransactions {
					fmt.Printf("   💰 BTC: %.2f BTC for $%.2f (avg: $%.2f)\n",
						tx.BTCPurchased, tx.USDSpent, tx.AvgPriceUSD)
				}
				if result.SharesOutstanding != nil {
					fmt.Printf("   📊 Shares: %.0f common shares\n", result.SharesOutstanding.CommonShares)
				}
			}
		} else {
			fmt.Printf("⚪ No data found (%dms)\n", result.ProcessingTimeMs)
		}

		// Save results if any data was found
		if btcCount > 0 || sharesCount > 0 {
			outputFile := filepath.Join(*outputDir, strings.Replace(fileName, ".htm", "_parsed.json", 1))
			if err := saveParseResult(result, outputFile); err != nil {
				fmt.Printf("   ⚠️  Warning: Could not save results: %v\n", err)
			} else if *verbose {
				fmt.Printf("   💾 Saved to: %s\n", outputFile)
			}
		}
	}

	totalTime := time.Since(startTime)

	// Summary
	fmt.Printf("\n📊 PARSING SUMMARY\n")
	fmt.Printf("==================\n")
	fmt.Printf("Files Processed: %d/%d\n", processedFiles, len(files))
	fmt.Printf("Files with Errors: %d\n", errorFiles)
	fmt.Printf("Total BTC Transactions: %d\n", totalTransactions)
	fmt.Printf("Total Shares Records: %d\n", totalSharesRecords)
	fmt.Printf("Processing Time: %v\n", totalTime)
	fmt.Printf("Average Time per File: %v\n", totalTime/time.Duration(len(files)))

	if *useGrok && grokClient != nil && grokClient.IsConfigured() {
		fmt.Printf("\n🤖 Grok AI was available for enhanced parsing\n")
	} else if *useGrok {
		fmt.Printf("\n⚠️  Grok AI was requested but not configured (missing GROK_API_KEY)\n")
	}

	fmt.Printf("\n✅ Bitcoin transaction parsing complete!\n")
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

// saveParseResult saves the parsing result to a JSON file
func saveParseResult(result *models.FilingParseResult, outputFile string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Marshal the full result to JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling result to JSON: %w", err)
	}

	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error writing JSON to file: %w", err)
	}

	return nil
}
