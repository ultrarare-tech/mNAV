package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jeffreykibler/mNAV/pkg/portfolio/analyzer"
	"github.com/jeffreykibler/mNAV/pkg/portfolio/tracker"
)

func main() {
	var (
		csvFile = flag.String("csv", "", "Path to portfolio CSV file")
		dataDir = flag.String("data", "data/portfolio/processed", "Directory to store processed portfolio data")
		verbose = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	if *csvFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -csv <portfolio.csv>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check if CSV file exists
	if _, err := os.Stat(*csvFile); os.IsNotExist(err) {
		log.Fatalf("CSV file does not exist: %s", *csvFile)
	}

	// Create analyzer and tracker
	analyzer := analyzer.NewAnalyzer()
	tracker := tracker.NewTracker(*dataDir)

	if *verbose {
		log.Printf("Parsing CSV file: %s", *csvFile)
	}

	// Parse the CSV file
	portfolio, err := analyzer.ParseCSV(*csvFile)
	if err != nil {
		log.Fatalf("Failed to parse CSV: %v", err)
	}

	if *verbose {
		log.Printf("Successfully parsed portfolio with %d positions", len(portfolio.Positions))
		log.Printf("Total portfolio value: $%.2f", portfolio.TotalValue)
		log.Printf("Portfolio date: %s", portfolio.Date.Format("2006-01-02"))
	}

	// Store the portfolio data
	if err := tracker.Store(portfolio); err != nil {
		log.Fatalf("Failed to store portfolio: %v", err)
	}

	// Copy raw CSV to raw data directory
	rawDir := filepath.Join(filepath.Dir(*dataDir), "raw")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		log.Printf("Warning: Failed to create raw directory: %v", err)
	} else {
		rawFileName := fmt.Sprintf("portfolio_%s.csv", portfolio.Date.Format("2006-01-02"))
		rawPath := filepath.Join(rawDir, rawFileName)

		if err := copyFile(*csvFile, rawPath); err != nil {
			log.Printf("Warning: Failed to copy CSV to raw directory: %v", err)
		} else if *verbose {
			log.Printf("Copied CSV to: %s", rawPath)
		}
	}

	fmt.Printf("✅ Successfully imported portfolio data for %s\n", portfolio.Date.Format("2006-01-02"))
	fmt.Printf("📊 Portfolio Summary:\n")
	fmt.Printf("   Total Value: $%.2f\n", portfolio.TotalValue)
	fmt.Printf("   Total Gain/Loss: $%.2f (%.2f%%)\n", portfolio.TotalGainLoss, portfolio.TotalGainLossPct)
	fmt.Printf("   Bitcoin Exposure: $%.2f (%.1f%%)\n", portfolio.AssetAllocation.BitcoinExposure, portfolio.AssetAllocation.BitcoinPercent)
	fmt.Printf("   FBTC/MSTR Ratio: %.2f:1\n", portfolio.AssetAllocation.FBTCMSTRRatio)

	fmt.Printf("\n🏦 Account Breakdown:\n")
	for name, account := range portfolio.Accounts {
		fmt.Printf("   %s: $%.2f\n", name, account.TotalValue)
	}

	fmt.Printf("\n💰 Asset Allocation:\n")
	fmt.Printf("   FBTC: $%.2f (%.1f%%)\n", portfolio.AssetAllocation.FBTCValue, portfolio.AssetAllocation.FBTCPercent)
	fmt.Printf("   MSTR: $%.2f (%.1f%%)\n", portfolio.AssetAllocation.MSTRValue, portfolio.AssetAllocation.MSTRPercent)
	fmt.Printf("   GLD:  $%.2f (%.1f%%)\n", portfolio.AssetAllocation.GLDValue, portfolio.AssetAllocation.GLDPercent)
	fmt.Printf("   Other: $%.2f (%.1f%%)\n", portfolio.AssetAllocation.OtherValue, portfolio.AssetAllocation.OtherPercent)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
