package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/edgar"
)

func main() {
	// Parse command-line flags
	ticker := flag.String("ticker", "MSTR", "Stock ticker symbol")
	startDate := flag.String("start", "", "Start date (YYYY-MM-DD)")
	endDate := flag.String("end", "", "End date (YYYY-MM-DD)")
	filingTypesStr := flag.String("filings", "10-K,10-Q,8-K", "Comma-separated list of filing types")
	userAgent := flag.String("user-agent", "", "User Agent for SEC EDGAR API")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	listOnly := flag.Bool("list", false, "Only list filings without parsing")
	email := flag.String("email", "", "Email address for SEC API user agent")
	openBrowser := flag.Bool("open-browser", false, "Open the SEC search page in default browser")
	downloadFilings := flag.Bool("download", false, "Download all filings to data/filings/TICKER directory")
	analyzeFilings := flag.Bool("analyze", false, "Analyze downloaded filings for Bitcoin transactions")
	analyzeAfterDownload := flag.Bool("analyze-after-download", false, "Analyze filings immediately after downloading")
	manualMode := flag.Bool("manual", false, "Show instructions for manual filing download")

	flag.Parse()

	// Set up proper user agent if email is provided
	if *email != "" && *userAgent == "" {
		*userAgent = fmt.Sprintf("MicroStrategy Bitcoin Tracker 1.0 %s", *email)
	} else if *userAgent == "" {
		// Set a default user agent with proper contact info
		hostname, _ := os.Hostname()
		// Include the contact email in the user agent
		*userAgent = fmt.Sprintf("MNAV Bitcoin Tracker 1.0 (Contact: contact@example.com; Host: %s)", hostname)
	}

	// Convert filing types string to slice
	filingTypes := strings.Split(*filingTypesStr, ",")
	for i, ft := range filingTypes {
		filingTypes[i] = strings.TrimSpace(ft)
	}

	// Set default dates if not provided
	if *startDate == "" {
		// Default to 1 year ago
		*startDate = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}
	if *endDate == "" {
		// Default to today
		*endDate = time.Now().Format("2006-01-02")
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	// Determine the project root directory
	var basePath string
	if filepath.Base(cwd) == "edgar-scraper" && filepath.Base(filepath.Dir(cwd)) == "cmd" {
		basePath = filepath.Join(cwd, "..", "..")
	} else {
		basePath = "."
	}

	// If analyze mode is enabled, analyze downloaded filings
	if *analyzeFilings {
		fmt.Printf("Analyzing downloaded filings for %s\n", *ticker)
		downloadPath := filepath.Join(basePath, "data", "filings")
		analyzeDownloadedFilings(*ticker, basePath, downloadPath)
		return
	}

	// Create the EDGAR client
	client := edgar.NewClient(*userAgent)

	if *verbose {
		log.Printf("Using User-Agent: %s", *userAgent)
		log.Printf("Date range: %s to %s", *startDate, *endDate)
		log.Printf("Filing types: %s", *filingTypesStr)
	}

	// Get CIK for the ticker
	cik, err := client.GetCIKByTicker(*ticker)
	if err != nil {
		log.Fatalf("Error getting CIK: %v", err)
	}
	if *verbose {
		log.Printf("Found CIK %s for ticker %s", cik, *ticker)
	}

	// Create the EDGAR URLs
	cikNum := strings.TrimLeft(cik, "0")
	// Classic browser interface
	classicURL := fmt.Sprintf("https://www.sec.gov/cgi-bin/browse-edgar?CIK=%s&owner=exclude&action=getcompany", cikNum)
	// New search interface
	searchURL := fmt.Sprintf("https://www.sec.gov/edgar/search/#/q=%s", url.QueryEscape(fmt.Sprintf("cik=%s", cikNum)))

	// If manual mode is enabled, provide detailed instructions
	if *manualMode || *openBrowser {
		fmt.Println("=======================================================================")
		fmt.Printf("MANUAL FILING DOWNLOAD INSTRUCTIONS FOR %s (CIK: %s)\n", *ticker, cik)
		fmt.Println("=======================================================================")
		fmt.Println("Due to SEC rate limiting, automated downloads may be restricted.")
		fmt.Println("Follow these steps to manually download filings:")
		fmt.Println()
		fmt.Println("1. Visit one of these URLs:")
		fmt.Printf("   Classic interface: %s\n", classicURL)
		fmt.Printf("   New search interface: %s\n", searchURL)
		fmt.Println()
		fmt.Println("2. For the classic interface:")
		fmt.Println("   - Filter by filing type using the dropdown")
		fmt.Println("   - Click on the filing you want to view")
		fmt.Println("   - On the filing page, find and click on the .htm document")
		fmt.Println("   - Save the HTML page (Ctrl+S or Cmd+S)")
		fmt.Println()
		fmt.Println("3. Save the files to this structure:")
		fmt.Printf("   data/filings/%s/YYYY-MM-DD_TYPE_ACCESSION.htm\n", *ticker)
		fmt.Println("   Example: data/filings/MSTR/2023-05-01_10-Q_0001050446-23-000051.htm")
		fmt.Println()
		fmt.Println("4. After downloading, analyze with:")
		fmt.Printf("   go run cmd/edgar-scraper/main.go -ticker %s -analyze\n", *ticker)
		fmt.Println("=======================================================================")

		// Open browser if requested
		if *openBrowser {
			fmt.Println("Opening SEC EDGAR search in your default browser...")
			openURLInBrowser(classicURL)
		}

		return
	}

	// If download mode is enabled, attempt to download filings
	if *downloadFilings {
		fmt.Printf("Attempting to download filings for %s to data/filings/%s\n", *ticker, *ticker)
		fmt.Println("Note: SEC rate limiting may prevent automated downloads.")
		fmt.Println("If downloads fail, use -manual flag for instructions on manual downloading.")

		downloadPath := filepath.Join(basePath, "data", "filings")

		// Download filings
		downloadedFiles, err := client.DownloadAllFilings(*ticker, filingTypes, *startDate, *endDate, downloadPath)
		if err != nil {
			fmt.Printf("Error downloading filings: %v\n", err)
			fmt.Println("Try using the -manual flag for instructions on manual downloading.")
			return
		}

		fmt.Printf("Downloaded %d filings for %s\n", len(downloadedFiles), *ticker)

		// If verbose, list the downloaded files
		if *verbose && len(downloadedFiles) > 0 {
			fmt.Println("Downloaded files:")
			for _, file := range downloadedFiles {
				fmt.Printf("  %s\n", file)
			}
		} else if len(downloadedFiles) == 0 {
			fmt.Println("No filings were downloaded. Try using the -manual flag for instructions.")
			return
		}

		// If analyze-after-download is enabled, analyze the filings
		if *analyzeAfterDownload {
			fmt.Printf("\nAnalyzing downloaded filings for %s\n", *ticker)
			analyzeDownloadedFilings(*ticker, basePath, downloadPath)
		}

		return
	}

	// Display guidance for using the tool
	fmt.Println("=======================================================================")
	fmt.Printf("SEC EDGAR Search for %s (CIK: %s)\n", *ticker, cik)
	fmt.Println("=======================================================================")
	fmt.Println("Due to SEC rate limiting, automated downloads may be restricted.")
	fmt.Println()
	fmt.Println("RECOMMENDED USAGE:")
	fmt.Println("1. Get manual download instructions:")
	fmt.Printf("   go run cmd/edgar-scraper/main.go -ticker %s -manual\n", *ticker)
	fmt.Println()
	fmt.Println("2. Open SEC EDGAR in browser:")
	fmt.Printf("   go run cmd/edgar-scraper/main.go -ticker %s -open-browser\n", *ticker)
	fmt.Println()
	fmt.Println("3. Analyze manually downloaded filings:")
	fmt.Printf("   go run cmd/edgar-scraper/main.go -ticker %s -analyze\n", *ticker)
	fmt.Println("=======================================================================")

	// Only attempt to list filings if specifically requested
	if *listOnly {
		fmt.Println("Attempting to retrieve filings through the SEC EDGAR browser interface...")
		fmt.Println("Note: This may fail due to SEC rate limiting.")

		filings, err := client.GetFilingsByCIK(cik, filingTypes, *startDate, *endDate)
		if err != nil {
			log.Printf("Error getting filings: %v", err)
			fmt.Println("Try using the -manual flag for instructions on manual searching.")
		} else if len(filings) > 0 {
			log.Printf("Found %d filings", len(filings))

			fmt.Printf("\nFilings for %s (CIK: %s):\n", *ticker, cik)
			fmt.Println("---------------------------------------------------")
			fmt.Printf("%-12s %-8s %-50s\n", "Date", "Type", "URL")
			fmt.Println("---------------------------------------------------")

			// Sort filings by date (newest first)
			sort.Slice(filings, func(i, j int) bool {
				return filings[i].FilingDate.After(filings[j].FilingDate)
			})

			for _, filing := range filings {
				fmt.Printf("%-12s %-8s %s\n",
					filing.FilingDate.Format("2006-01-02"),
					filing.FilingType,
					filing.DocumentURL)
			}
			fmt.Println("---------------------------------------------------")
		} else {
			fmt.Println("No filings found. Try using the -manual flag for instructions.")
		}
	}
}

// analyzeDownloadedFilings analyzes the downloaded filings for a ticker
func analyzeDownloadedFilings(ticker, basePath, downloadPath string) {
	// Analyze filings
	transactions, err := edgar.AnalyzeDownloadedFilings(ticker, downloadPath)
	if err != nil {
		log.Fatalf("Error analyzing filings: %v", err)
	}

	// Save the transactions
	storage := edgar.NewTransactionStorage(basePath)
	if err := storage.SaveTransactions(transactions); err != nil {
		log.Fatalf("Error saving transactions: %v", err)
	}

	// Output summary
	fmt.Printf("Company: %s\n", transactions.Company)
	if transactions.CIK != "" {
		fmt.Printf("CIK: %s\n", transactions.CIK)
	}
	fmt.Printf("Transactions: %d\n", len(transactions.Transactions))
	fmt.Printf("Last Updated: %s\n\n", transactions.LastUpdated.Format("2006-01-02 15:04:05"))

	// Sort transactions by date (newest first)
	sort.Slice(transactions.Transactions, func(i, j int) bool {
		return transactions.Transactions[i].Date.After(transactions.Transactions[j].Date)
	})

	// Output the most recent transactions
	fmt.Println("Recent Transactions:")

	// Display up to 10 recent transactions
	count := len(transactions.Transactions)
	if count > 10 {
		count = 10
	}

	for i := 0; i < count; i++ {
		tx := transactions.Transactions[i]
		fmt.Printf("  [%d] Date: %s", i+1, tx.Date.Format("2006-01-02"))

		if tx.BTCPurchased > 0 {
			fmt.Printf(", BTC: %.2f, USD: $%.2f million, Price: $%.2f",
				tx.BTCPurchased, tx.USDSpent/1000000, tx.AvgPriceUSD)
		} else if tx.TotalBTCAfter > 0 {
			fmt.Printf(", Holdings: %.2f BTC", tx.TotalBTCAfter)
		}
		fmt.Println()
	}

	// Output totals - only include actual purchases, not holdings statements
	var totalBTC float64
	var totalUSD float64
	var purchaseCount int

	for _, tx := range transactions.Transactions {
		if tx.BTCPurchased > 0 {
			totalBTC += tx.BTCPurchased
			totalUSD += tx.USDSpent
			purchaseCount++
		}
	}

	fmt.Printf("\nTotal BTC Purchased: %.2f\n", totalBTC)
	fmt.Printf("Total USD Spent: $%.2f million\n", totalUSD/1000000)
	if totalBTC > 0 {
		fmt.Printf("Average Price per BTC: $%.2f\n", totalUSD/totalBTC)
	}
	fmt.Printf("Number of purchase transactions: %d\n", purchaseCount)
}

// openURLInBrowser attempts to open the URL in the default browser
func openURLInBrowser(url string) {
	var err error

	// Check operating system
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default: // Linux, BSD, etc.
		err = exec.Command("xdg-open", url).Start()
	}

	if err != nil {
		log.Printf("Error opening browser: %v", err)
	}
}
