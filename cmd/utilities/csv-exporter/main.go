package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/shared/models"
)

// DailyFinancialData represents a single day's comprehensive financial data
type DailyFinancialData struct {
	Date                      time.Time
	StockPrice                float64
	StockVolume               float64
	MarketCap                 float64
	BitcoinPrice              float64
	BitcoinHoldings           float64
	BitcoinValue              float64
	SharesOutstanding         float64
	MNAV                      float64
	Premium                   float64
	BitcoinPerShare           float64
	BookValuePerShare         float64
	PriceToBook               float64
	BitcoinYield              float64
	TransactionDate           bool
	TransactionAmount         float64
	CumulativeBitcoinInvested float64
	AverageBitcoinCost        float64
	MarketClosed              bool
}

// StockDataPoint represents daily stock data
type StockDataPoint struct {
	Date   time.Time `json:"date"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// BitcoinDataPoint represents daily Bitcoin price data
type BitcoinDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
}

// StockDataResponse represents the stock data file format
type StockDataResponse struct {
	Symbol     string           `json:"symbol"`
	DataPoints []StockDataPoint `json:"dataPoints"`
}

// BitcoinDataResponse represents the Bitcoin data file format
type BitcoinDataResponse struct {
	Prices []BitcoinDataPoint `json:"prices"`
}

// BitcoinHistoricalData represents the structure of historical Bitcoin data files
type BitcoinHistoricalData struct {
	Symbol    string `json:"symbol"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Data      []struct {
		Date   string  `json:"date"`
		Open   float64 `json:"open"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Close  float64 `json:"close"`
		Volume float64 `json:"volume"`
	} `json:"data"`
}

func main() {
	var (
		symbol     = flag.String("symbol", "MSTR", "Stock symbol to export")
		outputFile = flag.String("output", "", "Output CSV file (default: {symbol}_financial_data_{date}.csv)")
		startDate  = flag.String("start", "2020-08-11", "Start date (YYYY-MM-DD)")
		endDate    = flag.String("end", "", "End date (YYYY-MM-DD), defaults to today")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	fmt.Printf("📊 CSV FINANCIAL DATA EXPORTER\n")
	fmt.Printf("==============================\n\n")
	fmt.Printf("🏢 Symbol: %s\n", *symbol)
	fmt.Printf("📅 Period: %s to %s\n", *startDate, getEndDate(*endDate))
	fmt.Printf("📁 Collecting all available financial data...\n\n")

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("❌ Error parsing start date: %v", err)
	}

	end := time.Now()
	if *endDate != "" {
		end, err = time.Parse("2006-01-02", *endDate)
		if err != nil {
			log.Fatalf("❌ Error parsing end date: %v", err)
		}
	}

	// Load all data sources
	fmt.Printf("📈 Loading stock data...\n")
	stockData, err := loadStockData(*symbol)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load stock data: %v", err)
	} else if *verbose {
		fmt.Printf("   ✅ Loaded %d stock data points\n", len(stockData.DataPoints))
	}

	fmt.Printf("₿ Loading Bitcoin price data...\n")
	bitcoinData, err := loadBitcoinData()
	if err != nil {
		log.Printf("⚠️  Warning: Could not load Bitcoin data: %v", err)
	} else if *verbose {
		fmt.Printf("   ✅ Loaded %d Bitcoin price points\n", len(bitcoinData.Prices))
	}

	fmt.Printf("🪙 Loading Bitcoin transaction data...\n")
	bitcoinTxData, err := loadBitcoinTransactionData(*symbol)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load Bitcoin transaction data: %v", err)
	} else if *verbose {
		fmt.Printf("   ✅ Loaded %d Bitcoin transactions\n", len(bitcoinTxData.AllTransactions))
	}

	fmt.Printf("📊 Loading shares outstanding data...\n")
	sharesData, err := loadSharesData(*symbol)
	if err != nil {
		log.Printf("⚠️  Warning: Could not load shares data: %v", err)
	} else if *verbose {
		fmt.Printf("   ✅ Loaded shares outstanding data\n")
	}

	// Generate comprehensive daily dataset
	fmt.Printf("\n🔄 Processing daily financial data...\n")
	dailyData := generateDailyDataset(start, end, stockData, bitcoinData, bitcoinTxData, sharesData, *verbose)

	if *verbose {
		fmt.Printf("   ✅ Generated %d daily records\n", len(dailyData))
	}

	// Validate data freshness
	fmt.Printf("\n🔍 Validating data freshness...\n")
	validateDataFreshness(dailyData, *symbol)

	// Export to CSV
	outputPath := *outputFile
	if outputPath == "" {
		timestamp := time.Now().Format("2006-01-02")
		outputPath = fmt.Sprintf("%s_financial_data_%s.csv", *symbol, timestamp)
	}

	fmt.Printf("\n💾 Exporting to CSV: %s\n", outputPath)
	if err := exportToCSV(dailyData, outputPath); err != nil {
		log.Fatalf("❌ Error exporting CSV: %v", err)
	}

	// Print summary
	printSummary(dailyData, *symbol, outputPath)
}

// loadStockData loads historical stock data from all available sources
func loadStockData(symbol string) (*StockDataResponse, error) {
	allStockData := make(map[string]StockDataPoint) // date -> latest stock data for that date

	// Load from all possible stock data sources
	stockSources := []func(string) (*StockDataResponse, error){
		loadFromPrimaryStockFiles,
		loadFromHistoricalStockFiles,
	}

	var totalLoaded int
	var loadedSources []string

	for _, loadFunc := range stockSources {
		if data, err := loadFunc(symbol); err == nil && data != nil {
			merged := mergeIntoStockMap(allStockData, data.DataPoints)
			totalLoaded += merged
			loadedSources = append(loadedSources, "stock source")
		}
	}

	if len(allStockData) == 0 {
		return nil, fmt.Errorf("no stock data files found for %s", symbol)
	}

	// Convert map back to sorted slice
	var dataPoints []StockDataPoint
	for _, point := range allStockData {
		dataPoints = append(dataPoints, point)
	}

	// Sort by date (oldest first)
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Date.Before(dataPoints[j].Date)
	})

	return &StockDataResponse{
		Symbol:     symbol,
		DataPoints: dataPoints,
	}, nil
}

// mergeIntoStockMap merges new stock data into the existing map, keeping the most recent data
func mergeIntoStockMap(existingData map[string]StockDataPoint, newData []StockDataPoint) int {
	merged := 0
	for _, point := range newData {
		dateKey := point.Date.Format("2006-01-02")

		// Always keep the newer data (prefer recently collected data)
		if existing, exists := existingData[dateKey]; !exists || point.Date.After(existing.Date) {
			existingData[dateKey] = point
			merged++
		}
	}
	return merged
}

// loadFromPrimaryStockFiles loads from primary stock data files
func loadFromPrimaryStockFiles(symbol string) (*StockDataResponse, error) {
	// Look for the most recent stock data file
	pattern := fmt.Sprintf("data/stock-data/%s_stock_data_*.json", symbol)
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no primary stock data files found")
	}

	// Use the most recent file
	sort.Strings(files)
	latestFile := files[len(files)-1]

	return loadStockDataFromFile(latestFile, symbol)
}

// loadFromHistoricalStockFiles loads from historical stock data files
func loadFromHistoricalStockFiles(symbol string) (*StockDataResponse, error) {
	// Look for historical stock data files
	pattern := fmt.Sprintf("data/stock-data/historical/%s_*.json", symbol)
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no historical stock files found")
	}

	allStockData := make([]StockDataPoint, 0)
	loadedFiles := 0

	// Load all historical files and merge their data
	for _, file := range files {
		if data, err := loadStockDataFromFile(file, symbol); err == nil {
			allStockData = append(allStockData, data.DataPoints...)
			loadedFiles++
		}
	}

	if loadedFiles == 0 {
		return nil, fmt.Errorf("failed to load any historical stock files")
	}

	return &StockDataResponse{
		Symbol:     symbol,
		DataPoints: allStockData,
	}, nil
}

// loadStockDataFromFile loads stock data from a specific file
func loadStockDataFromFile(filename, symbol string) (*StockDataResponse, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Parse the JSON structure - need to extract the historical data
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	// Extract historical prices
	historicalPrices, ok := rawData["historical_prices"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no historical_prices found in stock data")
	}

	historicalData, ok := historicalPrices["historical"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no historical array found in historical_prices")
	}

	stockData := &StockDataResponse{
		Symbol:     symbol,
		DataPoints: make([]StockDataPoint, 0, len(historicalData)),
	}

	for _, item := range historicalData {
		point, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse date
		dateStr, ok := point["date"].(string)
		if !ok {
			continue
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Parse price data
		close, _ := point["close"].(float64)
		volume, _ := point["volume"].(float64)
		open, _ := point["open"].(float64)
		high, _ := point["high"].(float64)
		low, _ := point["low"].(float64)

		stockPoint := StockDataPoint{
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}

		stockData.DataPoints = append(stockData.DataPoints, stockPoint)
	}

	return stockData, nil
}

// loadBitcoinData loads and merges historical Bitcoin price data from all available sources
// ensuring the most recent data is always used
func loadBitcoinData() (*BitcoinDataResponse, error) {
	allPrices := make(map[string]BitcoinDataPoint) // date -> latest price for that date

	// Load from all possible data sources
	sources := []func() (*BitcoinDataResponse, error){
		loadFromComprehensiveFiles,
		loadFromCoinMarketCapCSV,
		loadFromRecentCoinGeckoFiles,
		loadFromLegacyFiles,
	}

	var totalLoaded int
	var loadedSources []string

	for _, loadFunc := range sources {
		if data, err := loadFunc(); err == nil && data != nil {
			merged := mergeIntoPriceMap(allPrices, data.Prices)
			totalLoaded += merged
			loadedSources = append(loadedSources, "data source")
		}
	}

	if len(allPrices) == 0 {
		return nil, fmt.Errorf("no Bitcoin price data found from any source")
	}

	// Convert map back to sorted slice
	var prices []BitcoinDataPoint
	for _, price := range allPrices {
		prices = append(prices, price)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Timestamp.Before(prices[j].Timestamp)
	})

	return &BitcoinDataResponse{Prices: prices}, nil
}

// mergeIntoPriceMap merges new prices into the existing map, keeping the most recent data
func mergeIntoPriceMap(existingPrices map[string]BitcoinDataPoint, newPrices []BitcoinDataPoint) int {
	merged := 0
	for _, price := range newPrices {
		dateKey := price.Timestamp.Format("2006-01-02")

		// Always keep the newer data (prefer recently collected data)
		if existing, exists := existingPrices[dateKey]; !exists || price.Timestamp.After(existing.Timestamp) {
			existingPrices[dateKey] = price
			merged++
		}
	}
	return merged
}

// loadFromComprehensiveFiles loads from comprehensive historical files
func loadFromComprehensiveFiles() (*BitcoinDataResponse, error) {
	comprehensiveFiles := []string{
		"data/bitcoin-prices/historical/bitcoin_historical_2025-05-31_to_2020-07-23_coinmarketcap.json",
		"data/bitcoin-prices/historical/bitcoin_historical_2020-07-23_to_2025-05-31_coinmarketcap.json",
		"data/bitcoin-prices/historical/bitcoin_historical_2024-04-29_to_2025-05-31_coinmarketcap.json",
	}

	for _, file := range comprehensiveFiles {
		if _, err := os.Stat(file); err == nil {
			return loadFromBitcoinHistoricalJSON(file)
		}
	}

	return nil, fmt.Errorf("no comprehensive files found")
}

// loadFromCoinMarketCapCSV loads from CoinMarketCap CSV if available
func loadFromCoinMarketCapCSV() (*BitcoinDataResponse, error) {
	csvFile := "data/bitcoin-prices/Bitcoin_5_1_2020-6_1_2025_historical_data_coinmarketcap.csv"
	if _, err := os.Stat(csvFile); err == nil {
		return loadFromCoinMarketCapCSVFile(csvFile)
	}
	return nil, fmt.Errorf("no CSV file found")
}

// loadFromCoinMarketCapCSVFile loads Bitcoin price data from the CoinMarketCap CSV file
func loadFromCoinMarketCapCSVFile(filename string) (*BitcoinDataResponse, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create CSV reader with semicolon delimiter
	reader := csv.NewReader(file)
	reader.Comma = ';'

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	bitcoinData := &BitcoinDataResponse{
		Prices: make([]BitcoinDataPoint, 0, len(records)-1),
	}

	// Skip header row (index 0)
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 12 {
			continue // Skip incomplete records
		}

		// Parse the timestamp from timeClose field (index 1)
		timeCloseStr := strings.Trim(record[1], "\"")
		timestamp, err := time.Parse("2006-01-02T15:04:05.999Z", timeCloseStr)
		if err != nil {
			continue
		}

		// Parse the close price (index 8, not 9)
		closeStr := strings.Trim(record[8], "\"")
		closePrice, err := strconv.ParseFloat(closeStr, 64)
		if err != nil {
			continue
		}

		bitcoinPoint := BitcoinDataPoint{
			Timestamp: timestamp,
			Price:     closePrice,
		}

		bitcoinData.Prices = append(bitcoinData.Prices, bitcoinPoint)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(bitcoinData.Prices, func(i, j int) bool {
		return bitcoinData.Prices[i].Timestamp.Before(bitcoinData.Prices[j].Timestamp)
	})

	return bitcoinData, nil
}

// loadFromRecentCoinGeckoFiles loads from recent CoinGecko data files
func loadFromRecentCoinGeckoFiles() (*BitcoinDataResponse, error) {
	// Look for current Bitcoin data first (most recent)
	currentFiles := []string{
		"data/bitcoin-prices/historical/bitcoin_current_2025-06-16_7days.json",
		"data/bitcoin-prices/historical/bitcoin_current_*_*days.json",
	}

	// Then look for historical CoinGecko files
	patterns := []string{
		"data/bitcoin-prices/historical/bitcoin_current_*.json",
		"data/bitcoin-prices/historical/bitcoin_historical_202*-*-*_to_202*-*-*.json",
	}

	allFiles := make([]string, 0)

	// Add current files first (they have priority)
	for _, pattern := range currentFiles {
		if files, err := filepath.Glob(pattern); err == nil {
			allFiles = append(allFiles, files...)
		}
	}

	// Add historical files
	for _, pattern := range patterns {
		if files, err := filepath.Glob(pattern); err == nil {
			allFiles = append(allFiles, files...)
		}
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("no recent CoinGecko files found")
	}

	// Sort files to get the most recent ones first
	sort.Strings(allFiles)

	allPrices := make([]BitcoinDataPoint, 0)
	loadedFiles := 0

	// Load all recent files and merge their data
	for _, file := range allFiles {
		if data, err := loadFromCoinGeckoHistoricalJSON(file); err == nil {
			allPrices = append(allPrices, data.Prices...)
			loadedFiles++
		}
	}

	if loadedFiles == 0 {
		return nil, fmt.Errorf("failed to load any recent files")
	}

	return &BitcoinDataResponse{Prices: allPrices}, nil
}

// loadFromLegacyFiles loads from legacy file formats as final fallback
func loadFromLegacyFiles() (*BitcoinDataResponse, error) {
	pattern := "data/bitcoin-prices/historical/bitcoin_historical_*.json"
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no legacy files found")
	}

	// Use the most recent file
	sort.Strings(files)
	latestFile := files[len(files)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var response BitcoinDataResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// loadFromCoinGeckoHistoricalJSON loads Bitcoin price data from CoinGecko historical JSON files
func loadFromCoinGeckoHistoricalJSON(filename string) (*BitcoinDataResponse, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading CoinGecko file %s: %w", filename, err)
	}

	// Try to parse as CoinGecko format first
	var coinGeckoData struct {
		Symbol    string `json:"symbol"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Data      []struct {
			Date   string  `json:"date"`
			Open   float64 `json:"open"`
			High   float64 `json:"high"`
			Low    float64 `json:"low"`
			Close  float64 `json:"close"`
			Volume float64 `json:"volume"`
		} `json:"data"`
		Source string `json:"source"`
	}

	if err := json.Unmarshal(data, &coinGeckoData); err == nil && len(coinGeckoData.Data) > 0 {
		bitcoinData := &BitcoinDataResponse{
			Prices: make([]BitcoinDataPoint, 0, len(coinGeckoData.Data)),
		}

		for _, dataPoint := range coinGeckoData.Data {
			timestamp, err := time.Parse("2006-01-02", dataPoint.Date)
			if err != nil {
				continue
			}

			bitcoinPoint := BitcoinDataPoint{
				Timestamp: timestamp,
				Price:     dataPoint.Close,
			}

			bitcoinData.Prices = append(bitcoinData.Prices, bitcoinPoint)
		}

		return bitcoinData, nil
	}

	// Fallback to the historical format
	return loadFromBitcoinHistoricalJSON(filename)
}

// loadBitcoinTransactionData loads Bitcoin transaction data from the new MSTR JSON file
func loadBitcoinTransactionData(symbol string) (*models.ComprehensiveBitcoinAnalysis, error) {
	// First try to load from the saylor_tracker_mstr.json file
	saylorFile := "data/mstr/saylor_tracker_mstr.json"
	if _, err := os.Stat(saylorFile); err == nil {
		return loadFromSaylorTrackerFormat(saylorFile)
	}

	// Then try the comprehensive MSTR data file
	newDataFile := "data/mstr/mstr_bitcoin_data.json"
	if _, err := os.Stat(newDataFile); err == nil {
		return loadFromNewMSTRFormat(newDataFile)
	}

	// Fallback to old format for backwards compatibility
	filename := fmt.Sprintf("data/analysis/%s_comprehensive_bitcoin_analysis.json", symbol)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var bitcoinAnalysis models.ComprehensiveBitcoinAnalysis
	if err := json.Unmarshal(data, &bitcoinAnalysis); err != nil {
		return nil, err
	}

	return &bitcoinAnalysis, nil
}

// SaylorTrackerResponse represents the new MSTR data format
type SaylorTrackerResponse struct {
	Symbol            string                       `json:"symbol"`
	CompanyName       string                       `json:"company_name"`
	TotalBitcoin      float64                      `json:"total_bitcoin"`
	TotalInvestment   float64                      `json:"total_investment_usd"`
	AveragePrice      float64                      `json:"average_price_usd"`
	LastUpdated       string                       `json:"last_updated"`
	DataSources       []string                     `json:"data_sources"`
	Transactions      []SaylorTrackerTx            `json:"transactions"`
	QuarterlyData     []QuarterlyBitcoinData       `json:"quarterly_data"`
	SharesOutstanding []NewSharesOutstandingFormat `json:"shares_outstanding"`
}

type SaylorTrackerTx struct {
	Date            string  `json:"date"`
	Quarter         string  `json:"quarter"`
	EventType       string  `json:"event_type"`
	BitcoinAmount   float64 `json:"bitcoin_amount"`
	USDAmount       float64 `json:"usd_amount"`
	PricePerBitcoin float64 `json:"price_per_bitcoin"`
	CumulativeBTC   float64 `json:"cumulative_btc"`
	FilingType      string  `json:"filing_type"`
	FilingURL       string  `json:"filing_url"`
	DataSource      string  `json:"data_source"`
	Confidence      float64 `json:"confidence"`
	Notes           string  `json:"notes"`
}

type QuarterlyBitcoinData struct {
	Quarter           string  `json:"quarter"`
	Year              int     `json:"year"`
	BitcoinHeld       float64 `json:"bitcoin_held"`
	CarryingValue     float64 `json:"carrying_value_usd"`
	FairValue         float64 `json:"fair_value_usd"`
	Impairments       float64 `json:"impairments_usd"`
	SharesOutstanding float64 `json:"shares_outstanding"`
	Source            string  `json:"source"`
}

type NewSharesOutstandingFormat struct {
	Date              string  `json:"date"`
	SharesOutstanding float64 `json:"shares_outstanding"`
	SharesFloat       float64 `json:"shares_float"`
	Source            string  `json:"source"`
}

// loadFromNewMSTRFormat converts the new MSTR JSON format to the expected format
func loadFromNewMSTRFormat(filename string) (*models.ComprehensiveBitcoinAnalysis, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var saylorData SaylorTrackerResponse
	if err := json.Unmarshal(data, &saylorData); err != nil {
		return nil, err
	}

	// Convert to the expected format
	analysis := &models.ComprehensiveBitcoinAnalysis{
		Symbol:             saylorData.Symbol,
		Source:             "SaylorTracker-style comprehensive aggregation (Updated)",
		LastUpdated:        saylorData.LastUpdated,
		TotalBTC:           saylorData.TotalBitcoin,
		TotalInvestmentUSD: saylorData.TotalInvestment,
		AveragePrice:       saylorData.AveragePrice,
		AllTransactions:    make([]models.BitcoinTransaction, 0, len(saylorData.Transactions)),
		DataSources:        saylorData.DataSources,
	}

	// Convert transactions
	for _, tx := range saylorData.Transactions {
		// Parse the date string to time.Time
		txDate, err := time.Parse("2006-01-02T15:04:05Z", tx.Date)
		if err != nil {
			return nil, fmt.Errorf("error parsing transaction date %s: %w", tx.Date, err)
		}

		stdTx := models.BitcoinTransaction{
			Date:            txDate,
			FilingType:      tx.FilingType,
			FilingURL:       tx.FilingURL,
			BTCPurchased:    tx.BitcoinAmount,
			USDSpent:        tx.USDAmount,
			AvgPriceUSD:     tx.PricePerBitcoin,
			TotalBTCAfter:   tx.CumulativeBTC,
			ExtractedText:   tx.Notes,
			ConfidenceScore: tx.Confidence,
		}
		analysis.AllTransactions = append(analysis.AllTransactions, stdTx)
	}

	return analysis, nil
}

// SaylorTrackerSimple represents the simple saylor tracker format
type SaylorTrackerSimple struct {
	Date      string  `json:"date"`
	BTCAmount float64 `json:"btc_amount"`
	BTCOwned  float64 `json:"btc_owned"`
}

// loadFromSaylorTrackerFormat loads from the saylor_tracker_mstr.json format
func loadFromSaylorTrackerFormat(filename string) (*models.ComprehensiveBitcoinAnalysis, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var saylorData []SaylorTrackerSimple
	if err := json.Unmarshal(data, &saylorData); err != nil {
		return nil, err
	}

	// Calculate totals
	var totalBTC, totalInvestment float64
	for _, entry := range saylorData {
		if entry.BTCAmount > 0 { // Only count purchases, not sales
			totalInvestment += entry.BTCAmount * 50000 // Rough estimate of $50k average
		}
	}

	if len(saylorData) > 0 {
		totalBTC = saylorData[len(saylorData)-1].BTCOwned
	}

	avgPrice := totalInvestment / totalBTC
	if totalBTC == 0 {
		avgPrice = 0
	}

	// Convert to the expected format
	analysis := &models.ComprehensiveBitcoinAnalysis{
		Symbol:             "MSTR",
		Source:             "SaylorTracker simple format",
		LastUpdated:        time.Now().Format(time.RFC3339),
		TotalBTC:           totalBTC,
		TotalInvestmentUSD: totalInvestment,
		AveragePrice:       avgPrice,
		AllTransactions:    make([]models.BitcoinTransaction, 0, len(saylorData)),
		DataSources:        []string{"SaylorTracker data"},
	}

	// Convert transactions
	for _, entry := range saylorData {
		// Parse the date string to time.Time (format: "MMM DD, YYYY")
		txDate, err := time.Parse("Jan 02, 2006", entry.Date)
		if err != nil {
			return nil, fmt.Errorf("error parsing transaction date %s: %w", entry.Date, err)
		}

		// Estimate USD amount based on typical purchase prices
		var estimatedUSD float64
		if entry.BTCAmount != 0 {
			// Use rough price estimates based on date
			year := txDate.Year()
			var estimatedPrice float64
			switch {
			case year <= 2020:
				estimatedPrice = 15000
			case year == 2021:
				estimatedPrice = 45000
			case year == 2022:
				estimatedPrice = 25000
			case year == 2023:
				estimatedPrice = 30000
			case year == 2024:
				estimatedPrice = 75000
			case year >= 2025:
				estimatedPrice = 100000
			}
			estimatedUSD = entry.BTCAmount * estimatedPrice
		}

		var estimatedPricePerBTC float64
		if entry.BTCAmount != 0 {
			estimatedPricePerBTC = estimatedUSD / entry.BTCAmount
		}

		stdTx := models.BitcoinTransaction{
			Date:            txDate,
			FilingType:      "8-K",
			FilingURL:       "",
			BTCPurchased:    entry.BTCAmount,
			USDSpent:        estimatedUSD,
			AvgPriceUSD:     estimatedPricePerBTC,
			TotalBTCAfter:   entry.BTCOwned,
			ExtractedText:   fmt.Sprintf("Bitcoin transaction: %.0f BTC", entry.BTCAmount),
			ConfidenceScore: 0.9,
		}
		analysis.AllTransactions = append(analysis.AllTransactions, stdTx)
	}

	return analysis, nil
}

// SharesOutstandingData represents shares data structure
type SharesOutstandingData struct {
	Symbol                   string  `json:"symbol"`
	CurrentSharesOutstanding float64 `json:"current_shares_outstanding"`
	HistoricalData           []struct {
		Date              string  `json:"date"`
		SharesOutstanding float64 `json:"shares_outstanding"`
	} `json:"historical_data"`
}

// loadFreshHoldingsData loads the most recent Bitcoin holdings from fetch-mstr-holdings output
func loadFreshHoldingsData() (float64, error) {
	// Try to load from the most recent comprehensive analysis file
	comprehensiveFile := "data/analysis/MSTR_comprehensive_bitcoin_analysis.json"
	if _, err := os.Stat(comprehensiveFile); err == nil {
		data, err := os.ReadFile(comprehensiveFile)
		if err != nil {
			return 0, err
		}

		var analysis models.ComprehensiveBitcoinAnalysis
		if err := json.Unmarshal(data, &analysis); err != nil {
			return 0, err
		}

		if analysis.TotalBTC > 0 {
			return analysis.TotalBTC, nil
		}
	}

	// Try to load from raw holdings data
	rawHoldingsFile := "data/analysis/MSTR_bitcoin_holdings_raw.json"
	if _, err := os.Stat(rawHoldingsFile); err == nil {
		data, err := os.ReadFile(rawHoldingsFile)
		if err != nil {
			return 0, err
		}

		var rawData map[string]interface{}
		if err := json.Unmarshal(data, &rawData); err != nil {
			return 0, err
		}

		if totalBTC, ok := rawData["TotalBTC"].(float64); ok && totalBTC > 0 {
			return totalBTC, nil
		}
	}

	return 0, fmt.Errorf("no fresh holdings data found")
}

// loadSharesData loads shares outstanding data
func loadSharesData(symbol string) (*SharesOutstandingData, error) {
	// Look for the most recent shares data file
	pattern := fmt.Sprintf("data/analysis/%s_shares_outstanding_*.json", symbol)
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no shares data files found")
	}

	// Use the most recent file
	sort.Strings(files)
	latestFile := files[len(files)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var sharesData SharesOutstandingData
	if err := json.Unmarshal(data, &sharesData); err != nil {
		return nil, err
	}

	return &sharesData, nil
}

// generateDailyDataset creates a comprehensive daily dataset
func generateDailyDataset(start, end time.Time, stockData *StockDataResponse, bitcoinData *BitcoinDataResponse,
	bitcoinTxData *models.ComprehensiveBitcoinAnalysis, sharesData *SharesOutstandingData, verbose bool) []DailyFinancialData {

	dailyData := make(map[string]*DailyFinancialData)

	// Create daily records for the date range
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dailyData[dateStr] = &DailyFinancialData{
			Date: d,
		}
	}

	// Populate stock data
	if stockData != nil {
		for _, point := range stockData.DataPoints {
			dateStr := point.Date.Format("2006-01-02")
			if record, exists := dailyData[dateStr]; exists {
				record.StockPrice = point.Close
				record.StockVolume = point.Volume
			}
		}
	}

	// Populate Bitcoin price data
	if bitcoinData != nil {
		for _, point := range bitcoinData.Prices {
			dateStr := point.Timestamp.Format("2006-01-02")
			if record, exists := dailyData[dateStr]; exists {
				record.BitcoinPrice = point.Price
			}
		}
	}

	// Forward fill missing data
	fillMissingData(dailyData, start, end)

	// Calculate Bitcoin holdings over time
	if bitcoinTxData != nil {
		calculateBitcoinHoldings(dailyData, bitcoinTxData, verbose)
	}

	// Add shares outstanding data
	if sharesData != nil {
		populateSharesData(dailyData, sharesData)
	}

	// Calculate derived metrics
	calculateDerivedMetrics(dailyData)

	// Convert to sorted slice
	var result []DailyFinancialData
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if record, exists := dailyData[dateStr]; exists {
			result = append(result, *record)
		}
	}

	return result
}

// fillMissingData forward fills missing price data with constraints for data freshness
func fillMissingData(dailyData map[string]*DailyFinancialData, start, end time.Time) {
	var lastStockPrice, lastBitcoinPrice float64
	var lastStockDate, lastBitcoinDate time.Time
	const maxForwardFillDays = 5 // Don't forward-fill beyond 5 trading days

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if record, exists := dailyData[dateStr]; exists {
			// Check if this is a NYSE holiday/weekend
			isHoliday := isNYSEHoliday(d)

			// Handle stock price data
			if record.StockPrice > 0 {
				lastStockPrice = record.StockPrice
				lastStockDate = d
				record.MarketClosed = false
			} else if lastStockPrice > 0 {
				// Only forward-fill if within the acceptable time range
				daysSinceLastStock := int(d.Sub(lastStockDate).Hours() / 24)
				if daysSinceLastStock <= maxForwardFillDays {
					record.StockPrice = lastStockPrice
					if isHoliday {
						record.MarketClosed = true // True market closure
					} else {
						record.MarketClosed = false // Missing data on trading day
					}
				} else {
					record.MarketClosed = isHoliday // Only mark as closed if it's actually a holiday
				}
			} else {
				record.MarketClosed = isHoliday // Only mark as closed if it's actually a holiday
			}

			// Handle Bitcoin price data
			if record.BitcoinPrice > 0 {
				lastBitcoinPrice = record.BitcoinPrice
				lastBitcoinDate = d
			} else if lastBitcoinPrice > 0 {
				daysSinceLastBitcoin := int(d.Sub(lastBitcoinDate).Hours() / 24)
				if daysSinceLastBitcoin <= 2 { // Only 2 days for Bitcoin
					record.BitcoinPrice = lastBitcoinPrice
				}
			}
		}
	}
}

// validateDataFreshness checks if the data is current enough for recent dates
func validateDataFreshness(data []DailyFinancialData, symbol string) {
	if len(data) == 0 {
		return
	}

	today := time.Now()
	lastRecord := data[len(data)-1]

	// Check stock data freshness
	if lastRecord.StockPrice > 0 {
		daysSinceLastStock := int(today.Sub(lastRecord.Date).Hours() / 24)
		if daysSinceLastStock > 7 {
			fmt.Printf("⚠️  Warning: Stock data for %s is %d days old (last: %s)\n",
				symbol, daysSinceLastStock, lastRecord.Date.Format("2006-01-02"))
		}
	} else {
		fmt.Printf("⚠️  Warning: No current stock price data available for %s\n", symbol)
	}

	// Check Bitcoin data freshness
	if lastRecord.BitcoinPrice > 0 {
		daysSinceLastBitcoin := int(today.Sub(lastRecord.Date).Hours() / 24)
		if daysSinceLastBitcoin > 3 {
			fmt.Printf("⚠️  Warning: Bitcoin data is %d days old (last: %s)\n",
				daysSinceLastBitcoin, lastRecord.Date.Format("2006-01-02"))
		}
	} else {
		fmt.Printf("⚠️  Warning: No current Bitcoin price data available\n")
	}

	// Find the most recent data for each type
	var latestStockDate, latestBitcoinDate time.Time
	for _, record := range data {
		if record.StockPrice > 0 && record.Date.After(latestStockDate) {
			latestStockDate = record.Date
		}
		if record.BitcoinPrice > 0 && record.Date.After(latestBitcoinDate) {
			latestBitcoinDate = record.Date
		}
	}

	fmt.Printf("📊 Data Freshness Report:\n")
	fmt.Printf("   📈 Latest Stock Data: %s (%.1f days ago)\n",
		latestStockDate.Format("2006-01-02"), today.Sub(latestStockDate).Hours()/24)
	fmt.Printf("   ₿ Latest Bitcoin Data: %s (%.1f days ago)\n",
		latestBitcoinDate.Format("2006-01-02"), today.Sub(latestBitcoinDate).Hours()/24)
}

// calculateBitcoinHoldings calculates Bitcoin holdings over time
func calculateBitcoinHoldings(dailyData map[string]*DailyFinancialData, bitcoinTxData *models.ComprehensiveBitcoinAnalysis, verbose bool) {
	// Check for fresh holdings data from recent fetch-mstr-holdings run
	var latestHoldings float64
	if freshHoldings, err := loadFreshHoldingsData(); err == nil && freshHoldings > 0 {
		latestHoldings = freshHoldings
		if verbose {
			fmt.Printf("   🆕 Using fresh holdings data: %.0f BTC\n", latestHoldings)
		}
	}

	// Sort transactions by date
	sort.Slice(bitcoinTxData.AllTransactions, func(i, j int) bool {
		return bitcoinTxData.AllTransactions[i].Date.Before(bitcoinTxData.AllTransactions[j].Date)
	})

	// Create a map of transactions by date for quick lookup
	transactionsByDate := make(map[string]models.BitcoinTransaction)
	for _, tx := range bitcoinTxData.AllTransactions {
		dateStr := tx.Date.Format("2006-01-02")
		transactionsByDate[dateStr] = tx
	}

	// Process in chronological order by sorting the dates
	var dates []string
	for dateStr := range dailyData {
		dates = append(dates, dateStr)
	}
	sort.Strings(dates)

	var currentHoldings, totalInvested float64

	// Process each date in chronological order
	for _, dateStr := range dates {
		record := dailyData[dateStr]

		// Check if there's a transaction on this date
		if tx, exists := transactionsByDate[dateStr]; exists {
			currentHoldings = tx.TotalBTCAfter
			totalInvested += tx.USDSpent
			record.TransactionDate = true
			record.TransactionAmount = tx.BTCPurchased
			if verbose {
				fmt.Printf("   📈 %s: +%.0f BTC (Total: %.0f BTC)\n", dateStr, tx.BTCPurchased, currentHoldings)
			}
		}

		// For the most recent dates, use fresh holdings data if available
		if latestHoldings > 0 {
			recordDate, _ := time.Parse("2006-01-02", dateStr)
			today := time.Now()
			daysDiff := int(today.Sub(recordDate).Hours() / 24)

			// Use fresh holdings for recent dates (within last 30 days)
			if daysDiff <= 30 && latestHoldings > currentHoldings {
				currentHoldings = latestHoldings
				if verbose && daysDiff <= 1 {
					fmt.Printf("   🆕 %s: Updated to fresh holdings: %.0f BTC\n", dateStr, currentHoldings)
				}
			}
		}

		// Set the holdings for this date (carry forward from previous)
		record.BitcoinHoldings = currentHoldings
		record.CumulativeBitcoinInvested = totalInvested
		if currentHoldings > 0 {
			record.AverageBitcoinCost = totalInvested / currentHoldings
		}

		// Calculate Bitcoin value
		if record.BitcoinHoldings > 0 && record.BitcoinPrice > 0 {
			record.BitcoinValue = record.BitcoinHoldings * record.BitcoinPrice
		}
	}
}

// populateSharesData adds shares outstanding data
func populateSharesData(dailyData map[string]*DailyFinancialData, sharesData *SharesOutstandingData) {
	// Create a map of shares by date
	sharesLookup := make(map[string]float64)
	for _, point := range sharesData.HistoricalData {
		sharesLookup[point.Date] = point.SharesOutstanding
	}

	var currentShares float64 = sharesData.CurrentSharesOutstanding

	for dateStr, record := range dailyData {
		// Find the most recent shares data for this date
		recordDate, _ := time.Parse("2006-01-02", dateStr)

		var bestShares float64
		var bestDiff time.Duration = time.Hour * 24 * 365 * 10 // 10 years

		for shareDate, shares := range sharesLookup {
			shareTime, err := time.Parse("2006-01-02", shareDate)
			if err != nil {
				continue
			}

			diff := recordDate.Sub(shareTime)
			if diff >= 0 && diff < bestDiff {
				bestDiff = diff
				bestShares = shares
			}
		}

		if bestShares > 0 {
			record.SharesOutstanding = bestShares
		} else {
			record.SharesOutstanding = currentShares
		}
	}
}

// calculateDerivedMetrics calculates financial metrics
func calculateDerivedMetrics(dailyData map[string]*DailyFinancialData) {
	for _, record := range dailyData {
		// Market cap
		if record.StockPrice > 0 && record.SharesOutstanding > 0 {
			record.MarketCap = record.StockPrice * record.SharesOutstanding
		}

		// Bitcoin per share
		if record.BitcoinHoldings > 0 && record.SharesOutstanding > 0 {
			record.BitcoinPerShare = record.BitcoinHoldings / record.SharesOutstanding
		}

		// mNAV (Bitcoin value per share)
		if record.BitcoinValue > 0 && record.SharesOutstanding > 0 {
			bitcoinValuePerShare := record.BitcoinValue / record.SharesOutstanding
			if record.StockPrice > 0 {
				record.MNAV = record.StockPrice / bitcoinValuePerShare
				record.Premium = (record.MNAV - 1.0) * 100.0
			}
			record.BookValuePerShare = bitcoinValuePerShare
		}

		// Price to book
		if record.StockPrice > 0 && record.BookValuePerShare > 0 {
			record.PriceToBook = record.StockPrice / record.BookValuePerShare
		}

		// Bitcoin yield (Bitcoin value as % of market cap)
		if record.BitcoinValue > 0 && record.MarketCap > 0 {
			record.BitcoinYield = (record.BitcoinValue / record.MarketCap) * 100.0
		}
	}
}

// exportToCSV exports the data to CSV format
func exportToCSV(data []DailyFinancialData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Date",
		"Stock_Price",
		"Stock_Volume",
		"Market_Cap",
		"Bitcoin_Price",
		"Bitcoin_Holdings_BTC",
		"Bitcoin_Value_USD",
		"Shares_Outstanding",
		"mNAV_Ratio",
		"Premium_Percent",
		"Bitcoin_Per_Share",
		"Book_Value_Per_Share",
		"Price_To_Book",
		"Bitcoin_Yield_Percent",
		"Transaction_Date",
		"Transaction_Amount_BTC",
		"Cumulative_Investment_USD",
		"Average_Bitcoin_Cost",
		"Market_Closed",
	}

	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, record := range data {
		// Check if this is a NYSE holiday/weekend
		isHoliday := isNYSEHoliday(record.Date)

		// Determine market status
		var marketStatus string
		if isHoliday {
			// It's a holiday/weekend - market was closed
			marketStatus = "MARKET_CLOSED"
		} else if record.StockPrice > 0 {
			// Has stock price data on a trading day - market was open
			marketStatus = ""
		} else {
			// No price on a trading day - missing data
			marketStatus = "MISSING_DATA"
		}

		row := []string{
			record.Date.Format("2006-01-02"),
			formatFloat(record.StockPrice),
			formatFloat(record.StockVolume),
			formatFloat(record.MarketCap),
			formatFloat(record.BitcoinPrice),
			formatFloat(record.BitcoinHoldings),
			formatFloat(record.BitcoinValue),
			formatFloat(record.SharesOutstanding),
			formatFloat(record.MNAV),
			formatFloat(record.Premium),
			formatFloat(record.BitcoinPerShare),
			formatFloat(record.BookValuePerShare),
			formatFloat(record.PriceToBook),
			formatFloat(record.BitcoinYield),
			formatBool(record.TransactionDate),
			formatFloat(record.TransactionAmount),
			formatFloat(record.CumulativeBitcoinInvested),
			formatFloat(record.AverageBitcoinCost),
			marketStatus,
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// formatFloat formats a float64 for CSV
func formatFloat(f float64) string {
	if f == 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return ""
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// formatBool formats a boolean for CSV
func formatBool(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

// getEndDate returns the end date or today
func getEndDate(endDate string) string {
	if endDate == "" {
		return time.Now().Format("2006-01-02")
	}
	return endDate
}

// printSummary prints a summary of the exported data
func printSummary(data []DailyFinancialData, symbol, outputPath string) {
	if len(data) == 0 {
		fmt.Printf("\n❌ No data to export\n")
		return
	}

	// Calculate summary statistics
	var validDays, transactionDays int
	var firstDate, lastDate time.Time
	var maxBTC, maxPrice, maxMarketCap float64

	firstDate = data[0].Date
	lastDate = data[len(data)-1].Date

	for _, record := range data {
		if record.StockPrice > 0 {
			validDays++
		}
		if record.TransactionDate {
			transactionDays++
		}
		if record.BitcoinHoldings > maxBTC {
			maxBTC = record.BitcoinHoldings
		}
		if record.StockPrice > maxPrice {
			maxPrice = record.StockPrice
		}
		if record.MarketCap > maxMarketCap {
			maxMarketCap = record.MarketCap
		}
	}

	fmt.Printf("\n🎉 CSV Export Complete!\n")
	fmt.Printf("=======================\n\n")
	fmt.Printf("📁 File: %s\n", outputPath)
	fmt.Printf("📊 Symbol: %s\n", symbol)
	fmt.Printf("📅 Date Range: %s to %s\n", firstDate.Format("2006-01-02"), lastDate.Format("2006-01-02"))
	fmt.Printf("📈 Total Days: %d\n", len(data))
	fmt.Printf("📊 Valid Trading Days: %d\n", validDays)
	fmt.Printf("🪙 Bitcoin Transaction Days: %d\n", transactionDays)
	fmt.Printf("📈 Max Stock Price: $%.2f\n", maxPrice)
	fmt.Printf("🪙 Max Bitcoin Holdings: %.0f BTC\n", maxBTC)
	fmt.Printf("💰 Max Market Cap: $%.2f billion\n", maxMarketCap/1_000_000_000)

	fmt.Printf("\n📊 Excel Analysis Ready!\n")
	fmt.Printf("💡 Suggested Excel analyses:\n")
	fmt.Printf("   • Chart mNAV_Ratio vs Date\n")
	fmt.Printf("   • Correlation between Bitcoin_Price and Stock_Price\n")
	fmt.Printf("   • Premium_Percent trends over time\n")
	fmt.Printf("   • Bitcoin_Per_Share accumulation\n")
	fmt.Printf("   • Price_To_Book valuation analysis\n")
	fmt.Printf("   • Transaction impact on stock price\n")
}

// loadFromBitcoinHistoricalJSON loads Bitcoin price data from the historical JSON files
func loadFromBitcoinHistoricalJSON(filename string) (*BitcoinDataResponse, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file %s: %w", filename, err)
	}

	var historicalData BitcoinHistoricalData
	if err := json.Unmarshal(data, &historicalData); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	bitcoinData := &BitcoinDataResponse{
		Prices: make([]BitcoinDataPoint, 0, len(historicalData.Data)),
	}

	for _, dataPoint := range historicalData.Data {
		// Parse the date
		timestamp, err := time.Parse("2006-01-02", dataPoint.Date)
		if err != nil {
			continue // Skip invalid dates
		}

		bitcoinPoint := BitcoinDataPoint{
			Timestamp: timestamp,
			Price:     dataPoint.Close, // Use close price
		}

		bitcoinData.Prices = append(bitcoinData.Prices, bitcoinPoint)
	}

	return bitcoinData, nil
}

// isNYSEHoliday checks if a given date is a NYSE holiday
func isNYSEHoliday(date time.Time) bool {
	year := date.Year()
	month := date.Month()
	day := date.Day()

	// Weekend check
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return true
	}

	// NYSE holidays for 2020-2025
	holidays := map[string]bool{
		// 2020
		"2020-01-01": true, // New Year's Day
		"2020-01-20": true, // Martin Luther King Jr. Day
		"2020-02-17": true, // Presidents' Day
		"2020-04-10": true, // Good Friday
		"2020-05-25": true, // Memorial Day
		"2020-07-03": true, // Independence Day (observed)
		"2020-09-07": true, // Labor Day
		"2020-11-26": true, // Thanksgiving
		"2020-12-25": true, // Christmas

		// 2021
		"2021-01-01": true, // New Year's Day
		"2021-01-18": true, // Martin Luther King Jr. Day
		"2021-02-15": true, // Presidents' Day
		"2021-04-02": true, // Good Friday
		"2021-05-31": true, // Memorial Day
		"2021-07-05": true, // Independence Day (observed)
		"2021-09-06": true, // Labor Day
		"2021-11-25": true, // Thanksgiving
		"2021-12-24": true, // Christmas (observed)

		// 2022
		"2022-01-17": true, // Martin Luther King Jr. Day
		"2022-02-21": true, // Presidents' Day
		"2022-04-15": true, // Good Friday
		"2022-05-30": true, // Memorial Day
		"2022-06-20": true, // Juneteenth (observed)
		"2022-07-04": true, // Independence Day
		"2022-09-05": true, // Labor Day
		"2022-11-24": true, // Thanksgiving
		"2022-12-26": true, // Christmas (observed)

		// 2023
		"2023-01-02": true, // New Year's Day (observed)
		"2023-01-16": true, // Martin Luther King Jr. Day
		"2023-02-20": true, // Presidents' Day
		"2023-04-07": true, // Good Friday
		"2023-05-29": true, // Memorial Day
		"2023-06-19": true, // Juneteenth
		"2023-07-04": true, // Independence Day
		"2023-09-04": true, // Labor Day
		"2023-11-23": true, // Thanksgiving
		"2023-12-25": true, // Christmas

		// 2024
		"2024-01-01": true, // New Year's Day
		"2024-01-15": true, // Martin Luther King Jr. Day
		"2024-02-19": true, // Presidents' Day
		"2024-03-29": true, // Good Friday
		"2024-05-27": true, // Memorial Day
		"2024-06-19": true, // Juneteenth
		"2024-07-04": true, // Independence Day
		"2024-09-02": true, // Labor Day
		"2024-11-28": true, // Thanksgiving
		"2024-12-25": true, // Christmas

		// 2025
		"2025-01-01": true, // New Year's Day
		"2025-01-20": true, // Martin Luther King Jr. Day
		"2025-02-17": true, // Presidents' Day
		"2025-04-18": true, // Good Friday
		"2025-05-26": true, // Memorial Day
		"2025-06-19": true, // Juneteenth
		"2025-07-04": true, // Independence Day
		"2025-09-01": true, // Labor Day
		"2025-11-27": true, // Thanksgiving
		"2025-12-25": true, // Christmas
	}

	dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	return holidays[dateStr]
}
