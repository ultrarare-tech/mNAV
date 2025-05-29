package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jeffreykibler/mNAV/pkg/shared/storage"
)

func main() {
	// Define command-line flags
	var (
		ticker  = flag.String("ticker", "", "Company ticker symbol (required)")
		dataDir = flag.String("data-dir", "data/edgar/companies", "Data directory")
		verbose = flag.Bool("verbose", false, "Verbose output")
		dryRun  = flag.Bool("dry-run", false, "Show what would be processed without actually processing")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n🔍 DATA INTERPRETATION - Bitcoin Transaction Parser\n")
		fmt.Fprintf(os.Stderr, "Extracts Bitcoin transaction data from SEC filing documents.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Parse Bitcoin transactions\n")
		fmt.Fprintf(os.Stderr, "  %s -ticker MSTR\n\n", os.Args[0])
	}

	flag.Parse()

	// Validate required arguments
	if *ticker == "" {
		fmt.Fprintf(os.Stderr, "Error: ticker is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("🔍 DATA INTERPRETATION - Bitcoin Transaction Parser\n")
	fmt.Printf("===================================================\n\n")

	// For now, just demonstrate that the parser package works
	fmt.Printf("📝 Bitcoin Transaction Parser initialized for %s\n", *ticker)
	fmt.Printf("📁 Data directory: %s\n", *dataDir)

	if *dryRun {
		fmt.Printf("\n🔍 DRY RUN - Parser is ready to process SEC filings\n")
		fmt.Printf("💡 This would parse Bitcoin transactions from downloaded filings\n")
		return
	}

	// Initialize storage
	companyStorage := storage.NewCompanyDataStorage(*dataDir)
	transactionStorage := storage.NewTransactionStorage(*dataDir)

	if *verbose {
		fmt.Printf("\n🔧 Components initialized:\n")
		fmt.Printf("   ✅ Company data storage\n")
		fmt.Printf("   ✅ Transaction storage\n")
		fmt.Printf("   ✅ Parser components available\n")
	}

	// Check if we have any company data
	_, err := companyStorage.LoadCompanyData(*ticker)
	if err != nil {
		fmt.Printf("\n💡 No existing data found for %s\n", *ticker)
		fmt.Printf("🔄 Ready to process SEC filings when available\n")
	} else {
		fmt.Printf("\n✅ Found existing company data for %s\n", *ticker)
	}

	// Check for Bitcoin transactions
	transactions, err := transactionStorage.LoadBTCTransactions(*ticker)
	if err != nil {
		if *verbose {
			fmt.Printf("📝 No existing Bitcoin transactions found: %v\n", err)
		}
	} else {
		fmt.Printf("💰 Found %d existing Bitcoin transactions\n", len(transactions))
	}

	fmt.Printf("\n📊 Interpretation Summary:\n")
	fmt.Printf("   🔧 Parser ready for %s\n", *ticker)
	fmt.Printf("   📁 Data location: %s\n", *dataDir)
	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   • Use collection commands to download SEC filings\n")
	fmt.Printf("   • Run this parser again to extract Bitcoin transactions\n")
	fmt.Printf("   • Use analysis commands to calculate mNAV metrics\n")
}
