package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/collection/edgar"
)

func main() {
	// Define command-line flags
	var (
		ticker      = flag.String("ticker", "", "Company ticker symbol (required)")
		cik         = flag.String("cik", "", "Company CIK (optional, will be looked up if not provided)")
		filingTypes = flag.String("filing-types", "8-K,10-Q,10-K", "Comma-separated list of filing types")
		startDate   = flag.String("start", "", "Start date (YYYY-MM-DD, optional)")
		endDate     = flag.String("end", time.Now().Format("2006-01-02"), "End date (YYYY-MM-DD)")
		dataDir     = flag.String("data-dir", "data/edgar/companies", "Data directory")
		dryRun      = flag.Bool("dry-run", false, "Show what would be processed without actually processing")
		verbose     = flag.Bool("verbose", false, "Verbose output")
		incremental = flag.Bool("incremental", false, "Only process new filings since last run")
		useGrok     = flag.Bool("grok", false, "Use Grok AI for enhanced Bitcoin transaction extraction")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nEnhanced SEC EDGAR scraper for Bitcoin transactions and shares outstanding.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Download all filings for MSTR from 2023\n")
		fmt.Fprintf(os.Stderr, "  %s -ticker MSTR -start 2023-01-01\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Incremental update for MSTR\n")
		fmt.Fprintf(os.Stderr, "  %s -ticker MSTR -incremental\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Dry run to see what would be downloaded\n")
		fmt.Fprintf(os.Stderr, "  %s -ticker MSTR -start 2024-01-01 -dry-run\n", os.Args[0])
	}

	flag.Parse()

	// Validate required arguments
	if *ticker == "" {
		fmt.Fprintf(os.Stderr, "Error: ticker is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Set default end date to today
	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}

	// Parse filing types
	types := strings.Split(*filingTypes, ",")
	for i := range types {
		types[i] = strings.TrimSpace(types[i])
	}

	// Create data directory
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Error creating data directory: %v", err)
	}

	// Initialize EDGAR client
	userAgent := "mNAV Application - Jeffrey Kibler (jeffreykibler@protonmail.com)"
	client := edgar.NewClient(userAgent)

	// Initialize enhanced parser if Grok is enabled
	var enhancedParser *edgar.EnhancedDocumentParser
	if *useGrok {
		if os.Getenv("GROK_API_KEY") == "" {
			log.Fatal("GROK_API_KEY environment variable must be set when using -grok flag")
		}
		enhancedParser = edgar.NewEnhancedDocumentParser(client)
		fmt.Println("🤖 Grok AI enhancement enabled")
	}

	// Look up CIK if not provided
	if *cik == "" {
		lookedUpCIK, err := client.GetCIKByTicker(*ticker)
		if err != nil {
			log.Fatalf("Error looking up CIK for ticker %s: %v", *ticker, err)
		}
		*cik = lookedUpCIK
		if *verbose {
			fmt.Printf("Found CIK %s for ticker %s\n", *cik, *ticker)
		}
	}

	// Initialize document parser and storage
	storage := edgar.NewCompanyDataStorage(*dataDir)

	// Determine start date
	effectiveStartDate := *startDate
	if *incremental {
		// Load existing data to find last processed date
		existingData, err := storage.LoadCompanyData(*ticker)
		if err == nil && !existingData.LastFilingDate.IsZero() {
			// Start from day after last filing
			effectiveStartDate = existingData.LastFilingDate.AddDate(0, 0, 1).Format("2006-01-02")
			if *verbose {
				fmt.Printf("Incremental mode: starting from %s\n", effectiveStartDate)
			}
		} else if *startDate == "" {
			// No existing data and no start date specified, default to 1 year ago
			effectiveStartDate = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
			if *verbose {
				fmt.Printf("No existing data found, starting from %s\n", effectiveStartDate)
			}
		}
	} else if effectiveStartDate == "" {
		// Non-incremental mode with no start date, default to 1 year ago
		effectiveStartDate = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}

	// Get filings
	fmt.Printf("Fetching %s filings for %s (%s) from %s to %s...\n",
		*filingTypes, *ticker, *cik, effectiveStartDate, *endDate)

	filings, err := client.GetCompanyFilings(*ticker, types, effectiveStartDate, *endDate)
	if err != nil {
		log.Fatalf("Error fetching filings: %v", err)
	}

	if len(filings) == 0 {
		fmt.Println("No filings found for the specified criteria.")
		return
	}

	fmt.Printf("Found %d filings\n", len(filings))

	if *dryRun {
		fmt.Println("\nDry run mode - showing what would be processed:")
		for i, filing := range filings {
			fmt.Printf("%d. %s - %s (%s)\n", i+1, filing.FilingType, filing.FilingDate.Format("2006-01-02"), filing.AccessionNumber)
			fmt.Printf("   URL: %s\n", filing.URL)
		}
		return
	}

	// Process filings
	fmt.Println("\n🚀 Processing filings...")
	fmt.Printf("📊 Progress: [%s] 0/%d (0.0%%) | ETA: calculating...\n",
		strings.Repeat(" ", 20), len(filings))

	companyData := &edgar.CompanyFinancialData{
		Symbol:      *ticker,
		CompanyName: *ticker, // Will be updated with proper name from filings
		CIK:         *cik,
		LastUpdated: time.Now(),
	}

	processedCount := 0
	errorCount := 0
	btcCount := 0
	sharesCount := 0
	startTime := time.Now()

	// Process each filing
	for i, filing := range filings {
		currentNum := i + 1

		// Calculate progress
		progress := float64(currentNum-1) / float64(len(filings)) * 100
		elapsed := time.Since(startTime)
		var eta time.Duration
		if currentNum > 1 {
			avgTimePerFiling := elapsed / time.Duration(currentNum-1)
			remainingFilings := len(filings) - (currentNum - 1)
			eta = avgTimePerFiling * time.Duration(remainingFilings)
		}

		// Create progress bar
		progressBarWidth := 20
		filledWidth := int(progress / 100 * float64(progressBarWidth))
		progressBar := strings.Repeat("█", filledWidth) + strings.Repeat("░", progressBarWidth-filledWidth)

		// Clear previous line and show progress
		fmt.Printf("\r📊 Progress: [%s] %d/%d (%.1f%%) | Elapsed: %v | ETA: %v",
			progressBar, currentNum-1, len(filings), progress,
			elapsed.Round(time.Second), eta.Round(time.Second))

		fmt.Printf("\n🔍 [%d/%d] Processing: %s (%s) from %s\n",
			currentNum, len(filings), filing.AccessionNumber, filing.FilingType, filing.FilingDate.Format("2006-01-02"))

		if *verbose {
			fmt.Printf("    📄 Document URL: %s\n", filing.DocumentURL)
		}

		if *dryRun {
			fmt.Printf("    [DRY RUN] Would process filing %s\n", filing.AccessionNumber)
			continue
		}

		// Show current step
		fmt.Printf("    ⏳ Fetching document content...\n")

		// Process the filing with raw content storage
		var result *edgar.FilingProcessingResult
		if enhancedParser != nil {
			fmt.Printf("    🤖 Analyzing with Grok AI...\n")
			result, err = client.FetchAndParseDocumentWithParser(filing, storage, *ticker, enhancedParser)
		} else {
			fmt.Printf("    🔍 Parsing with regex patterns...\n")
			result, err = client.FetchAndParseDocument(filing, storage, *ticker)
		}

		if err != nil {
			fmt.Printf("    ❌ Error processing filing: %v\n", err)
			errorCount++
			continue
		}

		// Display processing results
		if result.ExtractedData != nil {
			fmt.Printf("    ✅ Raw filing saved (%d bytes, %s)\n",
				result.Document.ContentLength, result.Document.ContentType)

			if len(result.ExtractedData.BTCTransactions) > 0 {
				fmt.Printf("    🪙 Found %d BTC transactions\n", len(result.ExtractedData.BTCTransactions))
				btcCount += len(result.ExtractedData.BTCTransactions)
				if *verbose {
					for _, tx := range result.ExtractedData.BTCTransactions {
						fmt.Printf("      • %.2f BTC purchased for $%.2f\n", tx.BTCPurchased, tx.USDSpent)
					}
				}
			}

			if result.ExtractedData.SharesOutstanding != nil {
				fmt.Printf("    📈 Found shares outstanding: %.0f (confidence: %.2f)\n",
					result.ExtractedData.SharesOutstanding.TotalShares,
					result.ExtractedData.SharesOutstanding.ConfidenceScore)
				sharesCount++
			}

			if len(result.ExtractedData.ProcessingErrors) > 0 {
				fmt.Printf("    ⚠️  Processing warnings: %d\n", len(result.ExtractedData.ProcessingErrors))
				if *verbose {
					for _, err := range result.ExtractedData.ProcessingErrors {
						fmt.Printf("      - %s\n", err)
					}
				}
			}
		} else {
			fmt.Printf("    ❌ Processing failed: %v\n", result.ProcessingErrors)
			errorCount++
		}

		// Merge the extracted data into company storage
		if result.ExtractedData != nil {
			fmt.Printf("    💾 Saving extracted data...\n")
			if err := storage.MergeExtractedData(*ticker, result.ExtractedData); err != nil {
				fmt.Printf("    ❌ Error merging data: %v\n", err)
				errorCount++
			} else {
				fmt.Printf("    ✅ Data merged successfully\n")
			}
		}

		processedCount++

		// Show running totals
		fmt.Printf("    📊 Running totals: %d BTC transactions, %d shares records\n", btcCount, sharesCount)

		// Rate limiting with countdown
		if currentNum < len(filings) {
			fmt.Printf("    ⏱️  Rate limiting (2s)...")
			for j := 2; j > 0; j-- {
				fmt.Printf(" %d", j)
				time.Sleep(1 * time.Second)
			}
			fmt.Printf(" ✓\n")
		}

		fmt.Println() // Add spacing between filings
	}

	// Final progress update
	fmt.Printf("\r📊 Progress: [%s] %d/%d (100.0%%) | Total time: %v | Complete! ✅\n",
		strings.Repeat("█", 20), len(filings), len(filings), time.Since(startTime).Round(time.Second))

	companyData.LastProcessedDate = time.Now()

	// Save or merge data
	fmt.Println("\nSaving data...")
	if *incremental {
		err = storage.MergeCompanyData(*ticker, companyData)
	} else {
		err = storage.SaveCompanyData(companyData)
	}

	if err != nil {
		log.Fatalf("Error saving data: %v", err)
	}

	// Print summary
	fmt.Println("\n========== Summary ==========")
	fmt.Printf("Ticker: %s (CIK: %s)\n", *ticker, *cik)
	fmt.Printf("Filings processed: %d successful, %d errors\n", processedCount, errorCount)
	fmt.Printf("Bitcoin transactions found: %d\n", btcCount)
	fmt.Printf("Shares outstanding records found: %d\n", sharesCount)

	// Load and display current totals
	finalData, err := storage.LoadCompanyData(*ticker)
	if err == nil {
		totalBTC, _ := storage.GetTotalBTCHoldings(*ticker)
		latestShares, _ := storage.GetLatestSharesOutstanding(*ticker)

		fmt.Printf("\nCurrent totals:\n")
		fmt.Printf("  Total BTC holdings: %.2f\n", totalBTC)
		if latestShares != nil {
			fmt.Printf("  Latest shares outstanding: %.0f (as of %s)\n",
				latestShares.TotalShares, latestShares.Date.Format("2006-01-02"))
		}
		fmt.Printf("  Total transaction records: %d\n", len(finalData.BTCTransactions))
		fmt.Printf("  Total shares history records: %d\n", len(finalData.SharesHistory))
	}

	fmt.Printf("\nData saved to: %s\n", filepath.Join(*dataDir, "companies", *ticker))
}
