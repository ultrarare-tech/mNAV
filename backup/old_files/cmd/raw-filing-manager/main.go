package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ultrarare-tech/mNAV/pkg/edgar"
)

func main() {
	// Command line flags
	ticker := flag.String("ticker", "", "Company ticker symbol (required)")
	dataDir := flag.String("data-dir", "data/edgar", "Data directory path")
	command := flag.String("command", "list", "Command: list, stats, show, search")
	accessionNumber := flag.String("accession", "", "Accession number for show command")
	searchTerm := flag.String("search", "", "Search term for content search")
	outputFormat := flag.String("format", "table", "Output format: table, json")
	verbose := flag.Bool("verbose", false, "Verbose output")

	flag.Parse()

	if *ticker == "" {
		fmt.Println("Error: ticker is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize storage
	storage := edgar.NewCompanyDataStorage(*dataDir)

	switch *command {
	case "list":
		listRawFilings(storage, *ticker, *outputFormat, *verbose)
	case "stats":
		showStats(storage, *ticker, *outputFormat)
	case "show":
		if *accessionNumber == "" {
			fmt.Println("Error: accession number is required for show command")
			os.Exit(1)
		}
		showRawFiling(storage, *ticker, *accessionNumber, *verbose)
	case "search":
		if *searchTerm == "" {
			fmt.Println("Error: search term is required for search command")
			os.Exit(1)
		}
		searchRawFilings(storage, *ticker, *searchTerm, *verbose)
	default:
		fmt.Printf("Error: unknown command '%s'\n", *command)
		fmt.Println("Available commands: list, stats, show, search")
		os.Exit(1)
	}
}

func listRawFilings(storage *edgar.CompanyDataStorage, ticker, format string, verbose bool) {
	rawFilings, err := storage.ListRawFilings(ticker)
	if err != nil {
		log.Fatalf("Error listing raw filings: %v", err)
	}

	if len(rawFilings) == 0 {
		fmt.Printf("No raw filings found for %s\n", ticker)
		return
	}

	if format == "json" {
		jsonData, err := json.MarshalIndent(rawFilings, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return
	}

	// Table format
	fmt.Printf("Raw Filings for %s (%d total)\n", ticker, len(rawFilings))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("%-12s %-8s %-12s %-10s %-15s %s\n",
		"Date", "Type", "Accession", "Size", "Content Type", "Status")
	fmt.Println(strings.Repeat("-", 80))

	for _, filing := range rawFilings {
		status := "Downloaded"
		if !filing.ProcessedAt.IsZero() {
			status = "Processed"
		}

		sizeStr := formatBytes(filing.ContentLength)

		fmt.Printf("%-12s %-8s %-12s %-10s %-15s %s\n",
			filing.FilingDate.Format("2006-01-02"),
			filing.FilingType,
			filing.AccessionNumber[:12], // Truncate for display
			sizeStr,
			filing.ContentType,
			status)

		if verbose && filing.ProcessingNotes != "" {
			fmt.Printf("  Notes: %s\n", filing.ProcessingNotes)
		}
	}
}

func showStats(storage *edgar.CompanyDataStorage, ticker, format string) {
	stats, err := storage.GetRawFilingStats(ticker)
	if err != nil {
		log.Fatalf("Error getting stats: %v", err)
	}

	if format == "json" {
		jsonData, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		}
		fmt.Println(string(jsonData))
		return
	}

	// Table format
	fmt.Printf("Raw Filing Statistics for %s\n", ticker)
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("Total Filings: %v\n", stats["total_filings"])
	fmt.Printf("Total Size: %s\n", formatBytes(stats["total_size_bytes"].(int64)))

	if dateRange, ok := stats["date_range"].(map[string]interface{}); ok {
		if earliest, ok := dateRange["earliest"].(string); ok {
			fmt.Printf("Date Range: %s", earliest)
		}
		if latest, ok := dateRange["latest"].(string); ok {
			fmt.Printf(" to %s\n", latest)
		}
	}

	fmt.Println("\nFiling Types:")
	if filingTypes, ok := stats["filing_types"].(map[string]int); ok {
		for filingType, count := range filingTypes {
			fmt.Printf("  %s: %d\n", filingType, count)
		}
	}
}

func showRawFiling(storage *edgar.CompanyDataStorage, ticker, accessionNumber string, verbose bool) {
	rawDoc, content, err := storage.LoadRawFiling(ticker, accessionNumber)
	if err != nil {
		log.Fatalf("Error loading raw filing: %v", err)
	}

	fmt.Printf("Raw Filing Details\n")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Company: %s\n", rawDoc.CompanySymbol)
	fmt.Printf("Filing Type: %s\n", rawDoc.FilingType)
	fmt.Printf("Filing Date: %s\n", rawDoc.FilingDate.Format("2006-01-02"))
	fmt.Printf("Accession Number: %s\n", rawDoc.AccessionNumber)
	fmt.Printf("Document URL: %s\n", rawDoc.DocumentURL)
	fmt.Printf("Content Type: %s\n", rawDoc.ContentType)
	fmt.Printf("Content Length: %s\n", formatBytes(rawDoc.ContentLength))
	fmt.Printf("Downloaded: %s\n", rawDoc.DownloadedAt.Format("2006-01-02 15:04:05"))

	if !rawDoc.ProcessedAt.IsZero() {
		fmt.Printf("Processed: %s\n", rawDoc.ProcessedAt.Format("2006-01-02 15:04:05"))
	}

	if rawDoc.ProcessingNotes != "" {
		fmt.Printf("Processing Notes: %s\n", rawDoc.ProcessingNotes)
	}

	fmt.Printf("Checksum: %s\n", rawDoc.Checksum)

	if verbose && len(content) > 0 {
		fmt.Printf("\nContent Preview (first 1000 characters):\n")
		fmt.Println(strings.Repeat("-", 50))

		preview := string(content)
		if len(preview) > 1000 {
			preview = preview[:1000] + "..."
		}
		fmt.Println(preview)
	}
}

func searchRawFilings(storage *edgar.CompanyDataStorage, ticker, searchTerm string, verbose bool) {
	rawFilings, err := storage.ListRawFilings(ticker)
	if err != nil {
		log.Fatalf("Error listing raw filings: %v", err)
	}

	fmt.Printf("Searching for '%s' in %d raw filings for %s\n", searchTerm, len(rawFilings), ticker)
	fmt.Println(strings.Repeat("=", 80))

	matchCount := 0
	searchTermLower := strings.ToLower(searchTerm)

	for _, filing := range rawFilings {
		_, content, err := storage.LoadRawFiling(ticker, filing.AccessionNumber)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Could not load content for %s: %v\n", filing.AccessionNumber, err)
			}
			continue
		}

		contentLower := strings.ToLower(string(content))
		if strings.Contains(contentLower, searchTermLower) {
			matchCount++
			fmt.Printf("Match found in %s (%s) from %s\n",
				filing.FilingType,
				filing.AccessionNumber,
				filing.FilingDate.Format("2006-01-02"))

			if verbose {
				// Show context around matches
				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					if strings.Contains(strings.ToLower(line), searchTermLower) {
						fmt.Printf("  Line %d: %s\n", i+1, strings.TrimSpace(line))
						// Show a bit of context
						if i > 0 {
							fmt.Printf("    Previous: %s\n", strings.TrimSpace(lines[i-1]))
						}
						if i < len(lines)-1 {
							fmt.Printf("    Next: %s\n", strings.TrimSpace(lines[i+1]))
						}
						fmt.Println()
						break // Only show first match per filing
					}
				}
			}
		}
	}

	fmt.Printf("\nSearch complete: %d matches found in %d filings\n", matchCount, len(rawFilings))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
