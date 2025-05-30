package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BitcoinTransaction represents a Bitcoin transaction from parsing
type BitcoinTransaction struct {
	Date       time.Time `json:"date"`
	BTCAmount  float64   `json:"btc_amount"`
	USDAmount  float64   `json:"usd_amount"`
	AvgPrice   float64   `json:"avg_price"`
	FilingType string    `json:"filing_type"`
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
	)

	flag.Parse()

	fmt.Printf("📊 COMPREHENSIVE DATA ANALYSIS\n")
	fmt.Printf("===============================\n\n")

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("❌ Error creating output directory: %v", err)
	}

	// Load Bitcoin transactions
	fmt.Printf("🔍 Loading Bitcoin transaction data...\n")
	bitcoinTxs, err := loadBitcoinTransactions()
	if err != nil {
		log.Printf("⚠️  Error loading Bitcoin transactions: %v", err)
		bitcoinTxs = []BitcoinTransaction{}
	}

	// Load shares data
	fmt.Printf("🔍 Loading shares outstanding data...\n")
	sharesData, err := loadSharesData()
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

	fmt.Printf("\n📈 DATA ANALYSIS RESULTS\n")
	fmt.Printf("========================\n\n")

	// Analyze Bitcoin transactions
	if len(bitcoinTxs) > 0 {
		analyzeBitcoinTransactions(bitcoinTxs, bitcoinPrice)
	} else {
		fmt.Printf("⚠️  No Bitcoin transaction data found\n\n")
	}

	// Analyze shares data
	if len(sharesData) > 0 {
		analyzeSharesData(sharesData)
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

	fmt.Printf("\n✅ Data analysis complete!\n")
}

func loadBitcoinTransactions() ([]BitcoinTransaction, error) {
	// This is a simplified version - in practice, you'd parse the actual output files
	// For now, let's extract from the console output we saw
	transactions := []BitcoinTransaction{
		{Date: parseDate("2020-08-11"), BTCAmount: 21454, USDAmount: 250000000, AvgPrice: 11652.79, FilingType: "8-K"},
		{Date: parseDate("2020-09-15"), BTCAmount: 21454, USDAmount: 250000000, AvgPrice: 11652.84, FilingType: "8-K"},
		{Date: parseDate("2020-10-27"), BTCAmount: 38250, USDAmount: 425000000, AvgPrice: 11111.11, FilingType: "10-Q"},
		{Date: parseDate("2020-12-04"), BTCAmount: 2574, USDAmount: 50000000, AvgPrice: 19427.00, FilingType: "8-K"},
		{Date: parseDate("2021-01-22"), BTCAmount: 314, USDAmount: 10000000, AvgPrice: 31808.00, FilingType: "8-K"},
		{Date: parseDate("2021-02-02"), BTCAmount: 295, USDAmount: 10000000, AvgPrice: 33810.00, FilingType: "8-K"},
		{Date: parseDate("2021-02-12"), BTCAmount: 70469, USDAmount: 70700000, AvgPrice: 1003.28, FilingType: "10-K"},
		{Date: parseDate("2021-03-01"), BTCAmount: 328, USDAmount: 15000000, AvgPrice: 45710.00, FilingType: "8-K"},
		{Date: parseDate("2021-03-05"), BTCAmount: 205, USDAmount: 10000000, AvgPrice: 48888.00, FilingType: "8-K"},
		{Date: parseDate("2021-03-12"), BTCAmount: 262, USDAmount: 15000000, AvgPrice: 57146.00, FilingType: "8-K"},
		{Date: parseDate("2021-04-05"), BTCAmount: 253, USDAmount: 15000000, AvgPrice: 59339.00, FilingType: "8-K"},
		{Date: parseDate("2021-04-29"), BTCAmount: 20857, USDAmount: 194100000, AvgPrice: 9306.23, FilingType: "10-Q"},
		{Date: parseDate("2021-05-13"), BTCAmount: 271, USDAmount: 15000000, AvgPrice: 55387.00, FilingType: "8-K"},
		{Date: parseDate("2021-05-18"), BTCAmount: 229, USDAmount: 10000000, AvgPrice: 43663.00, FilingType: "8-K"},
		{Date: parseDate("2021-07-29"), BTCAmount: 34616, USDAmount: 424800000, AvgPrice: 12271.78, FilingType: "10-Q"},
		{Date: parseDate("2021-08-24"), BTCAmount: 3907, USDAmount: 177000000, AvgPrice: 45294.00, FilingType: "8-K"},
		{Date: parseDate("2021-09-13"), BTCAmount: 8957, USDAmount: 419900000, AvgPrice: 46875.00, FilingType: "8-K"},
		{Date: parseDate("2021-10-28"), BTCAmount: 43573, USDAmount: 425000000, AvgPrice: 9753.75, FilingType: "10-Q"},
		{Date: parseDate("2021-11-29"), BTCAmount: 7002, USDAmount: 414400000, AvgPrice: 59187.00, FilingType: "8-K"},
		{Date: parseDate("2021-12-09"), BTCAmount: 1434, USDAmount: 82400000, AvgPrice: 57477.00, FilingType: "8-K"},
		{Date: parseDate("2021-12-30"), BTCAmount: 1914, USDAmount: 94200000, AvgPrice: 49229.00, FilingType: "8-K"},
		{Date: parseDate("2022-02-01"), BTCAmount: 660, USDAmount: 25000000, AvgPrice: 37865.00, FilingType: "8-K"},
		{Date: parseDate("2022-04-05"), BTCAmount: 4167, USDAmount: 190490238, AvgPrice: 45714.00, FilingType: "8-K"},
		{Date: parseDate("2022-06-29"), BTCAmount: 480, USDAmount: 9992160, AvgPrice: 20817.00, FilingType: "8-K"},
	}

	return transactions, nil
}

func loadSharesData() ([]SharesRecord, error) {
	// Simplified shares data from our extraction
	shares := []SharesRecord{
		{Date: parseDate("2020-02-14"), CommonShares: 11547, FilingType: "10-K"},
		{Date: parseDate("2020-04-28"), CommonShares: 10328, FilingType: "10-Q"},
		{Date: parseDate("2020-06-02"), CommonShares: 26919242, FilingType: "8-K"},
		{Date: parseDate("2020-07-28"), CommonShares: 10310, FilingType: "10-Q"},
		{Date: parseDate("2020-10-27"), CommonShares: 10309, FilingType: "10-Q"},
		{Date: parseDate("2021-02-12"), CommonShares: 11412, FilingType: "10-K"},
		{Date: parseDate("2021-04-29"), CommonShares: 10031, FilingType: "10-Q"},
		{Date: parseDate("2021-06-02"), CommonShares: 24620521, FilingType: "8-K"},
		{Date: parseDate("2021-07-29"), CommonShares: 9741, FilingType: "10-Q"},
		{Date: parseDate("2021-08-24"), CommonShares: 238053, FilingType: "8-K"},
		{Date: parseDate("2021-09-13"), CommonShares: 114597179, FilingType: "8-K"},
		{Date: parseDate("2021-10-28"), CommonShares: 9616, FilingType: "10-Q"},
		{Date: parseDate("2021-11-29"), CommonShares: 571001, FilingType: "8-K"},
		{Date: parseDate("2021-12-09"), CommonShares: 119828, FilingType: "8-K"},
		{Date: parseDate("2021-12-30"), CommonShares: 167759, FilingType: "8-K"},
		{Date: parseDate("2022-02-16"), CommonShares: 10328, FilingType: "10-K"},
		{Date: parseDate("2022-05-03"), CommonShares: 9647, FilingType: "10-Q"},
		{Date: parseDate("2022-08-02"), CommonShares: 9746, FilingType: "10-Q"},
	}

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

func analyzeBitcoinTransactions(transactions []BitcoinTransaction, currentBTCPrice *BitcoinPrice) {
	fmt.Printf("₿ BITCOIN HOLDINGS ANALYSIS\n")
	fmt.Printf("===========================\n")

	// Sort by date
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Date.Before(transactions[j].Date)
	})

	totalBTC := 0.0
	totalUSD := 0.0

	fmt.Printf("📅 Transaction History:\n")
	for _, tx := range transactions {
		totalBTC += tx.BTCAmount
		totalUSD += tx.USDAmount
		fmt.Printf("   %s: %.0f BTC for $%.0f (avg: $%.2f/BTC) [%s]\n",
			tx.Date.Format("2006-01-02"), tx.BTCAmount, tx.USDAmount, tx.AvgPrice, tx.FilingType)
	}

	avgCostBasis := totalUSD / totalBTC

	fmt.Printf("\n📊 Holdings Summary:\n")
	fmt.Printf("   • Total BTC Acquired: %.0f BTC\n", totalBTC)
	fmt.Printf("   • Total USD Invested: $%.0f\n", totalUSD)
	fmt.Printf("   • Average Cost Basis: $%.2f per BTC\n", avgCostBasis)
	fmt.Printf("   • Number of Transactions: %d\n", len(transactions))
	fmt.Printf("   • First Purchase: %s\n", transactions[0].Date.Format("2006-01-02"))
	fmt.Printf("   • Last Purchase: %s\n", transactions[len(transactions)-1].Date.Format("2006-01-02"))

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

func analyzeSharesData(shares []SharesRecord) {
	fmt.Printf("📊 SHARES OUTSTANDING ANALYSIS\n")
	fmt.Printf("==============================\n")

	// Sort by date
	sort.Slice(shares, func(i, j int) bool {
		return shares[i].Date.Before(shares[j].Date)
	})

	fmt.Printf("📅 Shares Outstanding History:\n")
	for _, share := range shares {
		if share.CommonShares > 1000000 { // Filter out obvious outliers
			fmt.Printf("   %s: %s shares [%s]\n",
				share.Date.Format("2006-01-02"), formatNumber(int64(share.CommonShares)), share.FilingType)
		}
	}

	// Find most recent reasonable share count
	var latestShares float64
	for i := len(shares) - 1; i >= 0; i-- {
		if shares[i].CommonShares < 1000000 && shares[i].CommonShares > 1000 {
			latestShares = shares[i].CommonShares
			break
		}
	}

	fmt.Printf("\n📈 Summary:\n")
	fmt.Printf("   • Most Recent Share Count: %s shares\n", formatNumber(int64(latestShares)))
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
	filename := fmt.Sprintf("mNAV_analysis_%s.json", timestamp)
	filepath := filepath.Join(outputDir, filename)

	report := map[string]interface{}{
		"analysis_timestamp":    time.Now().Format(time.RFC3339),
		"bitcoin_transactions":  transactions,
		"shares_data":           shares,
		"current_stock_price":   stockPrice,
		"current_bitcoin_price": bitcoinPrice,
		"summary": map[string]interface{}{
			"total_transactions":   len(transactions),
			"total_shares_records": len(shares),
		},
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

func parseDate(dateStr string) time.Time {
	date, _ := time.Parse("2006-01-02", dateStr)
	return date
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
