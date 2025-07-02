package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ultrarare-tech/mNAV/pkg/edgar"
)

func main() {
	var (
		companiesJSON   = flag.String("companies", "data/companies.json", "Path to companies.json file")
		transactionsDir = flag.String("transactions", "data/transactions", "Path to transactions directory")
		dataDir         = flag.String("data", "data/edgar", "Directory for new data format")
		compatFile      = flag.String("compat", "", "Path to create compatibility file (optional)")
		skipCompanies   = flag.Bool("skip-companies", false, "Skip migrating companies.json")
		skipTx          = flag.Bool("skip-transactions", false, "Skip migrating transactions")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nMigrates data from old format to new SEC EDGAR-based format.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Migrate all data\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Only migrate companies.json\n")
		fmt.Fprintf(os.Stderr, "  %s -skip-transactions\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Create compatibility file for old code\n")
		fmt.Fprintf(os.Stderr, "  %s -compat data/companies_compat.json\n", os.Args[0])
	}

	flag.Parse()

	// Create storage
	storage := edgar.NewCompanyDataStorage(*dataDir)

	// Create migration service
	migrator := edgar.NewMigrationService(storage)

	// Migrate companies.json
	if !*skipCompanies {
		if _, err := os.Stat(*companiesJSON); err == nil {
			fmt.Printf("Migrating companies from %s...\n", *companiesJSON)
			if err := migrator.MigrateCompaniesJSON(*companiesJSON); err != nil {
				log.Printf("Error migrating companies.json: %v", err)
			} else {
				fmt.Println("Companies migration completed.")
			}
		} else {
			fmt.Printf("No companies.json found at %s, skipping.\n", *companiesJSON)
		}
	}

	// Migrate transaction files
	if !*skipTx {
		if info, err := os.Stat(*transactionsDir); err == nil && info.IsDir() {
			fmt.Printf("Migrating transactions from %s...\n", *transactionsDir)
			if err := migrator.MigrateTransactionFiles(*transactionsDir); err != nil {
				log.Printf("Error migrating transactions: %v", err)
			} else {
				fmt.Println("Transactions migration completed.")
			}
		} else {
			fmt.Printf("No transactions directory found at %s, skipping.\n", *transactionsDir)
		}
	}

	// Create compatibility file if requested
	if *compatFile != "" {
		fmt.Printf("Creating compatibility file at %s...\n", *compatFile)
		if err := migrator.CreateCompatibilityAdapter(*compatFile); err != nil {
			log.Fatalf("Error creating compatibility file: %v", err)
		}
		fmt.Println("Compatibility file created.")
	}

	// Summary
	companies, err := storage.ListCompanies()
	if err == nil {
		fmt.Printf("\nMigration complete. Total companies in new format: %d\n", len(companies))

		// Show summary for each company
		for _, symbol := range companies {
			data, err := storage.LoadCompanyData(symbol)
			if err == nil {
				totalBTC := 0.0
				for _, tx := range data.BTCTransactions {
					totalBTC += tx.BTCPurchased
				}

				fmt.Printf("  %s: %d shares records, %d BTC transactions (%.2f BTC total)\n",
					symbol, len(data.SharesHistory), len(data.BTCTransactions), totalBTC)
			}
		}
	}
}
