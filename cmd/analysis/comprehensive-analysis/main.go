package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BitcoinTransaction represents a Bitcoin transaction from parsing
type BitcoinTransaction struct {
	Date       time.Time `json:"date"`
	BTCAmount  float64   `json:"btc_amount"`
	USDAmount  float64   `json:"usd_amount"`
	AvgPrice   float64   `json:"avg_price"`
	FilingType string    `json:"filing_type"`
	Source     string    `json:"source"`
	Confidence float64   `json:"confidence"`
}

// SharesRecord represents shares outstanding data
type SharesRecord struct {
	Date         time.Time `json:"date"`
	CommonShares float64   `json:"common_shares"`
	FilingType   string    `json:"filing_type"`
	Source       string    `json:"source"`
}

// StockPrice represents current stock price data
type StockPrice struct {
	Ticker            string    `json:"ticker"`
	Price             float64   `json:"price"`
	MarketCap         float64   `json:"market_cap"`
	OutstandingShares float64   `json:"outstanding_shares"`
	Timestamp         time.Time `json:"timestamp"`
}

// BitcoinPrice represents current Bitcoin price data
type BitcoinPrice struct {
	Symbol           string    `json:"symbol"`
	Price            float64   `json:"price"`
	PercentChange24h float64   `json:"percent_change_24h"`
	Timestamp        time.Time `json:"timestamp"`
}

func main() {
	var (
		ticker    = flag.String("ticker", "MSTR", "Company ticker symbol")
		outputDir = flag.String("output-dir", "data/analysis", "Output directory for analysis results")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)

	flag.Parse()

	fmt.Printf("📊 COMPREHENSIVE MSTR ANALYSIS\n")
	fmt.Printf("===============================\n\n")

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	// Load Bitcoin transactions from parsed files
	fmt.Printf("🔍 Loading Bitcoin transaction data from parsed files...\n")
	bitcoinTxs, err := loadBitcoinTransactionsFromFiles(*verbose)
	if err != nil {
		log.Printf("⚠️  Error loading Bitcoin transactions: %v", err)
		bitcoinTxs = []BitcoinTransaction{}
	}

	// Load shares data from parsed files
	fmt.Printf("🔍 Loading shares outstanding data from parsed files...\n")
	sharesData, err := loadSharesDataFromFiles(*verbose)
	if err != nil {
		log.Printf("⚠️  Error loading shares data: %v", err)
		sharesData = []SharesRecord{}
	}

	// Load current stock price
	fmt.Printf("🔍 Loading current stock price...\n")
	stockPrice, err := loadLatestStockPrice(*ticker)
	if err != nil {
		log.Printf("⚠️  Error loading stock price: %v", err)
	}

	// Load current Bitcoin price
	fmt.Printf("🔍 Loading current Bitcoin price...\n")
	bitcoinPrice, err := loadLatestBitcoinPrice()
	if err != nil {
		log.Printf("⚠️  Error loading Bitcoin price: %v", err)
	}

	fmt.Printf("\n📈 COMPREHENSIVE ANALYSIS RESULTS\n")
	fmt.Printf("==================================\n\n")

	// Analyze Bitcoin transactions
	if len(bitcoinTxs) > 0 {
		analyzeBitcoinTransactions(bitcoinTxs, bitcoinPrice, *verbose)
	} else {
		fmt.Printf("⚠️  No Bitcoin transaction data found\n\n")
	}

	// Analyze shares data
	if len(sharesData) > 0 {
		analyzeSharesData(sharesData, *verbose)
	} else {
		fmt.Printf("⚠️  No shares outstanding data found\n\n")
	}

	// Current market data
	if stockPrice != nil {
		analyzeCurrentMarket(stockPrice, bitcoinPrice)
	}

	// Calculate mNAV if we have all data
	if len(bitcoinTxs) > 0 && stockPrice != nil && bitcoinPrice != nil {
		calculateMNAV(bitcoinTxs, stockPrice, bitcoinPrice)
	}

	// Save comprehensive report
	saveComprehensiveReport(*outputDir, bitcoinTxs, sharesData, stockPrice, bitcoinPrice)

	fmt.Printf("\n✅ Comprehensive analysis complete!\n")
}

func loadBitcoinTransactionsFromFiles(verbose bool) ([]BitcoinTransaction, error) {
	var transactions []BitcoinTransaction

	// Find all parsed files
	files, err := filepath.Glob("data/parsed/*.json")
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		// Fall back to reading from text files
		return loadBitcoinTransactionsFromTextFiles(verbose)
	}

	if verbose {
		fmt.Printf("   Found %d parsed files to analyze\n", len(files))
	}

	// Parse each file for Bitcoin transactions
	for _, file := range files {
		// Extract date and filing type from filename
		filename := filepath.Base(file)
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			continue
		}

		dateStr := parts[0]
		filingType := parts[1]

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			if verbose {
				fmt.Printf("   ⚠️  Could not parse date from %s: %v\n", filename, err)
			}
			continue
		}

		// Read and parse the file content
		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Printf("   ⚠️  Could not read %s: %v\n", file, err)
			}
			continue
		}

		// Extract Bitcoin transactions from the content
		btcTxs := extractBitcoinTransactionsFromContent(string(content), date, filingType, verbose)
		transactions = append(transactions, btcTxs...)
	}

	// Sort by date
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date.Before(transactions[j].Date)
	})

	if verbose {
		fmt.Printf("   Extracted %d Bitcoin transactions\n", len(transactions))
	}

	return transactions, nil
}

func loadBitcoinTransactionsFromTextFiles(verbose bool) ([]BitcoinTransaction, error) {
	var transactions []BitcoinTransaction

	// Find all parsed text files
	files, err := filepath.Glob("data/parsed/*.json")
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("   Reading from text-based parsed files\n")
	}

	// Regex patterns to extract Bitcoin transaction data
	btcRegex := regexp.MustCompile(`💰 BTC: ([\d,]+\.?\d*) BTC for \$([,\d]+\.?\d*) \(avg: \$([,\d]+\.?\d*)\)`)

	for _, file := range files {
		// Extract date and filing type from filename
		filename := filepath.Base(file)
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			continue
		}

		dateStr := parts[0]
		filingType := parts[1]

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Read the file content
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Find Bitcoin transactions in the content
		matches := btcRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) == 4 {
				btcAmount, _ := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", ""), 64)
				usdAmount, _ := strconv.ParseFloat(strings.ReplaceAll(match[2], ",", ""), 64)
				avgPrice, _ := strconv.ParseFloat(strings.ReplaceAll(match[3], ",", ""), 64)

				if btcAmount > 0 && usdAmount > 0 {
					transactions = append(transactions, BitcoinTransaction{
						Date:       date,
						BTCAmount:  btcAmount,
						USDAmount:  usdAmount,
						AvgPrice:   avgPrice,
						FilingType: filingType,
						Source:     "Parsed File",
					})
				}
			}
		}
	}

	// Sort by date
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date.Before(transactions[j].Date)
	})

	return transactions, nil
}

func extractBitcoinTransactionsFromContent(content string, date time.Time, filingType string, verbose bool) []BitcoinTransaction {
	var transactions []BitcoinTransaction

	// Try to parse as JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
		// Handle JSON format (future enhancement)
		return transactions
	}

	// Parse text format
	scanner := bufio.NewScanner(strings.NewReader(content))
	btcRegex := regexp.MustCompile(`💰 BTC: ([\d,]+\.?\d*) BTC for \$([,\d]+\.?\d*) \(avg: \$([,\d]+\.?\d*)\)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := btcRegex.FindStringSubmatch(line)
		if len(matches) == 4 {
			btcAmount, _ := strconv.ParseFloat(strings.ReplaceAll(matches[1], ",", ""), 64)
			usdAmount, _ := strconv.ParseFloat(strings.ReplaceAll(matches[2], ",", ""), 64)
			avgPrice, _ := strconv.ParseFloat(strings.ReplaceAll(matches[3], ",", ""), 64)

			if btcAmount > 0 && usdAmount > 0 {
				transactions = append(transactions, BitcoinTransaction{
					Date:       date,
					BTCAmount:  btcAmount,
					USDAmount:  usdAmount,
					AvgPrice:   avgPrice,
					FilingType: filingType,
					Source:     "Enhanced Parser",
				})
			}
		}
	}

	return transactions
}

func loadSharesDataFromFiles(verbose bool) ([]SharesRecord, error) {
	var shares []SharesRecord

	// Find all parsed files
	files, err := filepath.Glob("data/parsed/*.json")
	if err != nil {
		return nil, err
	}

	if verbose {
		fmt.Printf("   Found %d parsed files to analyze for shares data\n", len(files))
	}

	// Regex pattern to extract shares data
	sharesRegex := regexp.MustCompile(`📊 Shares: ([,\d]+) common shares`)

	for _, file := range files {
		// Extract date and filing type from filename
		filename := filepath.Base(file)
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			continue
		}

		dateStr := parts[0]
		filingType := parts[1]

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Read the file content
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Find shares data in the content
		matches := sharesRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) == 2 {
				sharesCount, _ := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", ""), 64)

				if sharesCount > 0 {
					shares = append(shares, SharesRecord{
						Date:         date,
						CommonShares: sharesCount,
						FilingType:   filingType,
						Source:       "Enhanced Parser",
					})
				}
			}
		}
	}

	// Sort by date
	sort.Slice(shares, func(i, j int) bool {
		return shares[i].Date.Before(shares[j].Date)
	})

	return shares, nil
}

func loadLatestStockPrice(ticker string) (*StockPrice, error) {
	// Load from the most recent stock price file
	files, err := filepath.Glob("data/stock-prices/*.json")
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no stock price files found")
	}

	// Get the most recent file
	sort.Strings(files)
	latestFile := files[len(files)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var stockData map[string]interface{}
	if err := json.Unmarshal(data, &stockData); err != nil {
		return nil, err
	}

	timestamp, _ := time.Parse(time.RFC3339, stockData["timestamp"].(string))

	return &StockPrice{
		Ticker:            stockData["ticker"].(string),
		Price:             stockData["price"].(float64),
		MarketCap:         stockData["market_cap"].(float64),
		OutstandingShares: stockData["outstanding_shares"].(float64),
		Timestamp:         timestamp,
	}, nil
}

func loadLatestBitcoinPrice() (*BitcoinPrice, error) {
	// Load from the most recent Bitcoin price file
	files, err := filepath.Glob("data/bitcoin-prices/*.json")
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no Bitcoin price files found")
	}

	// Get the most recent file
	sort.Strings(files)
	latestFile := files[len(files)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var btcData map[string]interface{}
	if err := json.Unmarshal(data, &btcData); err != nil {
		return nil, err
	}

	timestamp, _ := time.Parse(time.RFC3339, btcData["timestamp"].(string))

	return &BitcoinPrice{
		Symbol:           btcData["symbol"].(string),
		Price:            btcData["price"].(float64),
		PercentChange24h: btcData["percent_change_24h"].(float64),
		Timestamp:        timestamp,
	}, nil
}

func analyzeBitcoinTransactions(transactions []BitcoinTransaction, currentBTCPrice *BitcoinPrice, verbose bool) {
	fmt.Printf("₿ BITCOIN HOLDINGS ANALYSIS\n")
	fmt.Printf("===========================\n")

	totalBTC := 0.0
	totalUSD := 0.0

	if verbose {
		fmt.Printf("📅 Complete Transaction History:\n")
		for _, tx := range transactions {
			totalBTC += tx.BTCAmount
			totalUSD += tx.USDAmount
			fmt.Printf("   %s: %.0f BTC for $%.0f (avg: $%.2f/BTC) [%s]\n",
				tx.Date.Format("2006-01-02"), tx.BTCAmount, tx.USDAmount, tx.AvgPrice, tx.FilingType)
		}
	} else {
		// Just calculate totals
		for _, tx := range transactions {
			totalBTC += tx.BTCAmount
			totalUSD += tx.USDAmount
		}
	}

	avgCostBasis := totalUSD / totalBTC

	fmt.Printf("\n📊 Holdings Summary:\n")
	fmt.Printf("   • Total BTC Acquired: %.0f BTC\n", totalBTC)
	fmt.Printf("   • Total USD Invested: $%.0f\n", totalUSD)
	fmt.Printf("   • Average Cost Basis: $%.2f per BTC\n", avgCostBasis)
	fmt.Printf("   • Number of Transactions: %d\n", len(transactions))
	if len(transactions) > 0 {
		fmt.Printf("   • First Purchase: %s\n", transactions[0].Date.Format("2006-01-02"))
		fmt.Printf("   • Last Purchase: %s\n", transactions[len(transactions)-1].Date.Format("2006-01-02"))
	}

	if currentBTCPrice != nil {
		currentValue := totalBTC * currentBTCPrice.Price
		unrealizedGain := currentValue - totalUSD
		gainPercent := (unrealizedGain / totalUSD) * 100

		fmt.Printf("\n💰 Current Valuation (as of %s):\n", currentBTCPrice.Timestamp.Format("2006-01-02"))
		fmt.Printf("   • Current BTC Price: $%.2f\n", currentBTCPrice.Price)
		fmt.Printf("   • Current Portfolio Value: $%.0f\n", currentValue)
		fmt.Printf("   • Unrealized Gain/Loss: $%.0f (%.1f%%)\n", unrealizedGain, gainPercent)
	}

	fmt.Printf("\n")
}

func analyzeSharesData(shares []SharesRecord, verbose bool) {
	fmt.Printf("📊 SHARES OUTSTANDING ANALYSIS\n")
	fmt.Printf("==============================\n")

	if verbose {
		fmt.Printf("📅 Complete Shares Outstanding History:\n")
		for _, share := range shares {
			// Filter out obvious outliers for display
			if share.CommonShares < 100000000 { // Less than 100M shares
				fmt.Printf("   %s: %s shares [%s]\n",
					share.Date.Format("2006-01-02"), formatNumber(int64(share.CommonShares)), share.FilingType)
			}
		}
	}

	// Find most recent reasonable share count
	var latestShares float64
	var latestDate time.Time
	for i := len(shares) - 1; i >= 0; i-- {
		if shares[i].CommonShares < 100000000 && shares[i].CommonShares > 1000 {
			latestShares = shares[i].CommonShares
			latestDate = shares[i].Date
			break
		}
	}

	fmt.Printf("\n📈 Summary:\n")
	fmt.Printf("   • Most Recent Share Count: %s shares (%s)\n",
		formatNumber(int64(latestShares)), latestDate.Format("2006-01-02"))
	fmt.Printf("   • Data Points: %d\n", len(shares))
	fmt.Printf("\n")
}

func analyzeCurrentMarket(stockPrice *StockPrice, bitcoinPrice *BitcoinPrice) {
	fmt.Printf("📈 CURRENT MARKET DATA\n")
	fmt.Printf("======================\n")

	fmt.Printf("🏢 %s Stock:\n", stockPrice.Ticker)
	fmt.Printf("   • Current Price: $%.2f\n", stockPrice.Price)
	fmt.Printf("   • Market Cap: $%.2fB\n", stockPrice.MarketCap/1e9)
	fmt.Printf("   • Outstanding Shares: %s\n", formatNumber(int64(stockPrice.OutstandingShares)))
	fmt.Printf("   • Last Updated: %s\n", stockPrice.Timestamp.Format("2006-01-02 15:04:05"))

	if bitcoinPrice != nil {
		fmt.Printf("\n₿ Bitcoin:\n")
		fmt.Printf("   • Current Price: $%.2f\n", bitcoinPrice.Price)
		fmt.Printf("   • 24h Change: %.2f%%\n", bitcoinPrice.PercentChange24h)
		fmt.Printf("   • Last Updated: %s\n", bitcoinPrice.Timestamp.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n")
}

func calculateMNAV(transactions []BitcoinTransaction, stockPrice *StockPrice, bitcoinPrice *BitcoinPrice) {
	fmt.Printf("🧮 mNAV CALCULATION\n")
	fmt.Printf("===================\n")

	// Calculate total BTC holdings
	totalBTC := 0.0
	for _, tx := range transactions {
		totalBTC += tx.BTCAmount
	}

	// Calculate Bitcoin portfolio value
	btcPortfolioValue := totalBTC * bitcoinPrice.Price

	// Calculate mNAV per share
	mnavPerShare := btcPortfolioValue / stockPrice.OutstandingShares

	// Calculate premium/discount
	premium := ((stockPrice.Price - mnavPerShare) / mnavPerShare) * 100

	fmt.Printf("📊 mNAV Analysis:\n")
	fmt.Printf("   • Total BTC Holdings: %.0f BTC\n", totalBTC)
	fmt.Printf("   • BTC Portfolio Value: $%.0f\n", btcPortfolioValue)
	fmt.Printf("   • Outstanding Shares: %s\n", formatNumber(int64(stockPrice.OutstandingShares)))
	fmt.Printf("   • mNAV per Share: $%.2f\n", mnavPerShare)
	fmt.Printf("   • Current Stock Price: $%.2f\n", stockPrice.Price)

	if premium > 0 {
		fmt.Printf("   • Premium to mNAV: %.1f%%\n", premium)
	} else {
		fmt.Printf("   • Discount to mNAV: %.1f%%\n", -premium)
	}

	fmt.Printf("\n💡 Interpretation:\n")
	if premium > 0 {
		fmt.Printf("   MSTR is trading at a %.1f%% PREMIUM to its Bitcoin NAV\n", premium)
		fmt.Printf("   This suggests the market values MSTR's Bitcoin strategy and/or\n")
		fmt.Printf("   expects additional value beyond just Bitcoin holdings.\n")
	} else {
		fmt.Printf("   MSTR is trading at a %.1f%% DISCOUNT to its Bitcoin NAV\n", -premium)
		fmt.Printf("   This could represent a potential value opportunity.\n")
	}

	fmt.Printf("\n")
}

func saveComprehensiveReport(outputDir string, transactions []BitcoinTransaction, shares []SharesRecord, stockPrice *StockPrice, bitcoinPrice *BitcoinPrice) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("comprehensive_mNAV_analysis_%s.json", timestamp)
	filepath := filepath.Join(outputDir, filename)

	// Calculate summary statistics
	totalBTC := 0.0
	totalUSD := 0.0
	for _, tx := range transactions {
		totalBTC += tx.BTCAmount
		totalUSD += tx.USDAmount
	}

	var mnavPerShare, premium float64
	if stockPrice != nil && bitcoinPrice != nil && totalBTC > 0 {
		btcPortfolioValue := totalBTC * bitcoinPrice.Price
		mnavPerShare = btcPortfolioValue / stockPrice.OutstandingShares
		premium = ((stockPrice.Price - mnavPerShare) / mnavPerShare) * 100
	}

	report := map[string]interface{}{
		"analysis_timestamp": time.Now().Format(time.RFC3339),
		"summary": map[string]interface{}{
			"total_btc_holdings":   totalBTC,
			"total_usd_invested":   totalUSD,
			"average_cost_basis":   totalUSD / totalBTC,
			"total_transactions":   len(transactions),
			"total_shares_records": len(shares),
			"mnav_per_share":       mnavPerShare,
			"premium_to_mnav":      premium,
		},
		"bitcoin_transactions":  transactions,
		"shares_data":           shares,
		"current_stock_price":   stockPrice,
		"current_bitcoin_price": bitcoinPrice,
	}

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("❌ Error marshaling report: %v", err)
		return
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		log.Printf("❌ Error writing report: %v", err)
		return
	}

	fmt.Printf("💾 Comprehensive report saved to: %s\n", filepath)
}

func formatNumber(n int64) string {
	if n == 0 {
		return "0"
	}

	str := fmt.Sprintf("%d", n)
	result := ""

	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(char)
	}

	return result
}
