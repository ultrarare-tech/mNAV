package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	edgarclient "github.com/ultrarare-tech/mNAV/pkg/collection/edgar"
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
		dryRun      = flag.Bool("dry-run", false, "Show what would be collected without actually downloading")
		listLocal   = flag.Bool("list", false, "List already downloaded filings")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n🗂️  DATA COLLECTION - SEC EDGAR Filings\n")
		fmt.Fprintf(os.Stderr, "Collects raw SEC filing documents for future interpretation and analysis.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Collect all 2023 filings for MSTR\n")
		fmt.Fprintf(os.Stderr, "  %s -ticker MSTR -start 2023-01-01\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Collect only 10-Q quarterly reports\n")
		fmt.Fprintf(os.Stderr, "  %s -ticker MSTR -filing-types 10-Q\n", os.Args[0])
	}

	flag.Parse()

	// Validate required arguments
	if *ticker == "" {
		fmt.Fprintf(os.Stderr, "Error: ticker is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("🗂️  DATA COLLECTION - SEC EDGAR Filings\n")
	fmt.Printf("==================================================\n\n")

	// Initialize EDGAR client
	userAgent := "mNAV Application - Jeffrey Kibler (jeffreykibler@protonmail.com)"
	client := edgarclient.NewClient(userAgent)

	// Handle list command
	if *listLocal {
		companyDir := filepath.Join(*dataDir, *ticker)
		filings, err := client.ListDownloadedFilings(*ticker, *dataDir)
		if err != nil {
			log.Fatalf("Error listing downloaded filings: %v", err)
		}

		if len(filings) == 0 {
			fmt.Printf("❌ No downloaded filings found for %s in %s\n", *ticker, companyDir)
			return
		}

		fmt.Printf("📁 Downloaded filings for %s (%d total):\n\n", *ticker, len(filings))
		for i, filing := range filings {
			fmt.Printf("%d. %s - %s (%s)\n", i+1, filing.FilingType, filing.FilingDate.Format("2006-01-02"), filing.AccessionNumber)
		}
		fmt.Printf("\n📁 Files stored in: %s/\n", companyDir)
		return
	}

	// Look up CIK if not provided
	if *cik == "" {
		lookedUpCIK, err := client.GetCIKByTicker(*ticker)
		if err != nil {
			log.Fatalf("Error looking up CIK for ticker %s: %v", *ticker, err)
		}
		*cik = lookedUpCIK
		if *verbose {
			fmt.Printf("✅ Found CIK %s for ticker %s\n", *cik, *ticker)
		}
	}

	// Parse filing types
	types := strings.Split(*filingTypes, ",")
	for i := range types {
		types[i] = strings.TrimSpace(types[i])
	}

	// Set default start date if not provided
	effectiveStartDate := *startDate
	if effectiveStartDate == "" {
		effectiveStartDate = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}

	// Get filings list
	fmt.Printf("📊 Fetching %s filings for %s (%s) from %s to %s...\n",
		*filingTypes, *ticker, *cik, effectiveStartDate, *endDate)

	filings, err := client.GetCompanyFilings(*ticker, types, effectiveStartDate, *endDate)
	if err != nil {
		log.Fatalf("Error fetching filings list: %v", err)
	}

	if len(filings) == 0 {
		fmt.Println("❌ No filings found for the specified criteria.")
		return
	}

	fmt.Printf("✅ Found %d filings to collect\n\n", len(filings))

	if *dryRun {
		fmt.Println("🔍 DRY RUN - Filings that would be collected:")
		for i, filing := range filings {
			fmt.Printf("%d. %s - %s (%s)\n", i+1, filing.FilingType, filing.FilingDate.Format("2006-01-02"), filing.AccessionNumber)
		}
		return
	}

	// Download filings
	fmt.Printf("⬇️  Downloading %d filings...\n", len(filings))

	successCount := 0
	errorCount := 0

	// Create company-specific directory
	companyDir := filepath.Join(*dataDir, *ticker)
	if err := os.MkdirAll(companyDir, 0755); err != nil {
		log.Fatalf("Error creating company directory: %v", err)
	}

	for i, filing := range filings {
		fmt.Printf("[%d/%d] Downloading %s (%s)... ",
			i+1, len(filings), filing.AccessionNumber, filing.FilingType)

		// Use the DownloadFilingContent method to save to disk
		filePath, err := client.DownloadFilingContent(filing, companyDir)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			errorCount++
			continue
		}

		// Get file size for reporting
		fileInfo, err := os.Stat(filePath)
		var sizeKB int64
		if err == nil {
			sizeKB = fileInfo.Size() / 1024
		}

		fmt.Printf("✅ Saved to %s (%d KB)\n", filepath.Base(filePath), sizeKB)
		successCount++

		// Rate limiting - be respectful to SEC servers
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n📈 Collection Summary:\n")
	fmt.Printf("   ✅ Successfully downloaded: %d filings\n", successCount)
	if errorCount > 0 {
		fmt.Printf("   ❌ Errors: %d filings\n", errorCount)
	}
	fmt.Printf("   📁 Data stored in: %s/\n", companyDir)
	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   • Run interpretation commands to extract data from filings\n")
	fmt.Printf("   • Run analysis commands to calculate metrics\n")
}
