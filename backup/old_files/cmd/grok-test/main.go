package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ultrarare-tech/mNAV/pkg/edgar"
)

func main() {
	var (
		ticker    = flag.String("ticker", "MSTR", "Company ticker symbol")
		startDate = flag.String("start", "2024-10-01", "Start date (YYYY-MM-DD)")
		endDate   = flag.String("end", "2024-12-31", "End date (YYYY-MM-DD)")
		useGrok   = flag.Bool("grok", false, "Use Grok API for enhanced extraction")
		verbose   = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Check for Grok API key if requested
	if *useGrok && os.Getenv("GROK_API_KEY") == "" {
		log.Fatal("GROK_API_KEY environment variable must be set to use Grok API")
	}

	fmt.Printf("🚀 Testing Bitcoin Transaction Extraction\n")
	fmt.Printf("Ticker: %s\n", *ticker)
	fmt.Printf("Date Range: %s to %s\n", *startDate, *endDate)
	fmt.Printf("Using Grok API: %v\n", *useGrok)
	fmt.Printf("%s\n", strings.Repeat("=", 60))

	// Create EDGAR client
	client := edgar.NewClient("mNAV Grok Test 1.0 test@example.com")

	// Get company filings
	fmt.Printf("Fetching filings for %s...\n", *ticker)
	filings, err := client.GetCompanyFilings(*ticker, []string{"8-K", "10-Q", "10-K"}, *startDate, *endDate)
	if err != nil {
		log.Fatalf("Error getting filings: %v", err)
	}

	fmt.Printf("Found %d filings\n\n", len(filings))

	// Process each filing
	if *useGrok {
		fmt.Println("🤖 Using Enhanced Parser with Grok API")
		enhancedParser := edgar.NewEnhancedDocumentParser(client)
		processFilings(filings, enhancedParser.ParseHTMLDocumentEnhanced, client, *verbose)
	} else {
		fmt.Println("📝 Using Regex-based Parser")
		regexParser := edgar.NewDocumentParser(client)
		processFilings(filings, regexParser.ParseHTMLDocument, client, *verbose)
	}
}

func processFilings(filings []edgar.Filing, parseFunc func([]byte, edgar.Filing) ([]edgar.BitcoinTransaction, error), client *edgar.Client, verbose bool) {
	fmt.Printf("%s\n", strings.Repeat("-", 60))

	totalTransactions := 0

	// Process each filing
	for i, filing := range filings {
		if i >= 3 { // Limit to first 3 filings for testing
			break
		}

		fmt.Printf("\n📄 Processing Filing %d/%d\n", i+1, min(len(filings), 3))
		fmt.Printf("Type: %s | Date: %s\n", filing.FilingType, filing.FilingDate.Format("2006-01-02"))
		fmt.Printf("Accession: %s\n", filing.AccessionNumber)

		// Fetch document content
		content, err := client.FetchDocumentContent(filing.DocumentURL)
		if err != nil {
			fmt.Printf("❌ Error fetching content: %v\n", err)
			continue
		}

		fmt.Printf("Document size: %d bytes\n", len(content))

		// Parse for Bitcoin transactions
		transactions, err := parseFunc(content, filing)
		if err != nil {
			fmt.Printf("❌ Error parsing document: %v\n", err)
			continue
		}

		fmt.Printf("Found %d Bitcoin transactions:\n", len(transactions))

		for j, tx := range transactions {
			fmt.Printf("\n  Transaction %d:\n", j+1)
			fmt.Printf("    BTC: %.2f\n", tx.BTCPurchased)
			fmt.Printf("    USD: $%.2f\n", tx.USDSpent)
			fmt.Printf("    Price: $%.2f per BTC\n", tx.AvgPriceUSD)
			fmt.Printf("    Confidence: %.2f\n", tx.ConfidenceScore)

			if verbose {
				fmt.Printf("    Source: %s\n", truncateText(tx.ExtractedText, 150))
			}
		}

		totalTransactions += len(transactions)

		if len(transactions) == 0 {
			fmt.Printf("  ℹ️  No Bitcoin transactions found\n")
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("Total filings processed: %d\n", min(len(filings), 3))
	fmt.Printf("Total Bitcoin transactions found: %d\n", totalTransactions)
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
