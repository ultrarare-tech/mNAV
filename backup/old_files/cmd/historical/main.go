package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/yahoo"
)

func main() {
	// Define command-line flags
	symbolFlag := flag.String("symbol", "MSTR", "Stock symbol to fetch historical data for")
	periodFlag := flag.String("period", "5y", "Time period (1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max)")
	outputDirFlag := flag.String("output", "data/historical", "Directory to save the historical data")
	customRangeFlag := flag.String("custom-range", "", "Custom date range in format 'YYYY-MM-DD,YYYY-MM-DD' (overrides period)")

	flag.Parse()

	// Validate inputs
	if *symbolFlag == "" {
		log.Fatalf("Error: Symbol cannot be empty")
	}

	// Determine the range parameter
	rangeParam := *periodFlag
	if *customRangeFlag != "" {
		// Parse the custom range dates
		startEndDates, err := parseCustomDateRange(*customRangeFlag)
		if err != nil {
			log.Fatalf("Error parsing custom date range: %v", err)
		}
		rangeParam = startEndDates
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	// Determine project root and output directory paths
	var outputDir string
	if filepath.IsAbs(*outputDirFlag) {
		outputDir = *outputDirFlag
	} else {
		// Determine if we're running from project root or cmd/historical
		var basePath string
		if filepath.Base(cwd) == "historical" && filepath.Base(filepath.Dir(cwd)) == "cmd" {
			basePath = filepath.Join(cwd, "..", "..")
		} else {
			basePath = cwd
		}
		outputDir = filepath.Join(basePath, *outputDirFlag)
	}

	// Fetch historical data
	fmt.Printf("Fetching historical data for %s over period %s...\n", *symbolFlag, rangeParam)
	histData, err := yahoo.GetHistoricalData(*symbolFlag, rangeParam)
	if err != nil {
		log.Fatalf("Error fetching historical data: %v", err)
	}

	// Save to file
	filePath, err := yahoo.SaveHistoricalDataToFile(histData, outputDir)
	if err != nil {
		log.Fatalf("Error saving historical data: %v", err)
	}

	fmt.Printf("Successfully fetched %d data points for %s\n", len(histData.Data), *symbolFlag)
	fmt.Printf("Data saved to: %s\n", filePath)
}

// parseCustomDateRange converts a date range string in format "YYYY-MM-DD,YYYY-MM-DD" to Unix timestamps
func parseCustomDateRange(dateRangeStr string) (string, error) {
	// Parse date format YYYY-MM-DD,YYYY-MM-DD
	layout := "2006-01-02"

	// Split the range
	dates := strings.Split(dateRangeStr, ",")

	if len(dates) != 2 || dates[0] == "" || dates[1] == "" {
		return "", fmt.Errorf("invalid date range format: %s - expected 'YYYY-MM-DD,YYYY-MM-DD'", dateRangeStr)
	}

	// Parse start date
	startDate, err := time.Parse(layout, dates[0])
	if err != nil {
		return "", fmt.Errorf("invalid start date: %v", err)
	}

	// Parse end date
	endDate, err := time.Parse(layout, dates[1])
	if err != nil {
		return "", fmt.Errorf("invalid end date: %v", err)
	}

	// Convert to Unix timestamps
	startUnix := startDate.Unix()
	endUnix := endDate.Unix()

	// Return formatted range string
	return fmt.Sprintf("%d,%d", startUnix, endUnix), nil
}
