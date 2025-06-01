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

	"github.com/jeffreykibler/mNAV/pkg/analysis/metrics"
	"github.com/jeffreykibler/mNAV/pkg/collection/alphavantage"
	"github.com/jeffreykibler/mNAV/pkg/collection/fmp"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

// SharesOutstanding represents shares outstanding at a point in time
type SharesOutstanding struct {
	Date        time.Time `json:"date"`
	TotalShares float64   `json:"total_shares"`
	FilingType  string    `json:"filing_type"`
	Source      string    `json:"source"`
}

// HistoricalMNAVPoint represents a single point in the mNAV time series
type HistoricalMNAVPoint struct {
	Date              string  `json:"date"`
	StockPrice        float64 `json:"stock_price"`
	BitcoinPrice      float64 `json:"bitcoin_price"`
	BitcoinHoldings   float64 `json:"bitcoin_holdings"`
	SharesOutstanding float64 `json:"shares_outstanding"`
	MarketCap         float64 `json:"market_cap"`
	BitcoinValue      float64 `json:"bitcoin_value"`
	MNAV              float64 `json:"mnav"`
	MNAVPerShare      float64 `json:"mnav_per_share"`
	Premium           float64 `json:"premium_percentage"`
}

// HistoricalMNAVData represents the complete historical mNAV dataset
type HistoricalMNAVData struct {
	Symbol      string                 `json:"symbol"`
	StartDate   string                 `json:"start_date"`
	EndDate     string                 `json:"end_date"`
	DataPoints  []HistoricalMNAVPoint  `json:"data_points"`
	Metadata    map[string]interface{} `json:"metadata"`
	GeneratedAt time.Time              `json:"generated_at"`
}

func main() {
	var (
		symbol    = flag.String("symbol", "MSTR", "Stock symbol")
		startDate = flag.String("start", "2020-08-11", "Start date (YYYY-MM-DD)")
		endDate   = flag.String("end", "", "End date (YYYY-MM-DD), defaults to today")
		outputDir = flag.String("output", "data/analysis/mnav", "Output directory")
		interval  = flag.String("interval", "daily", "Calculation interval: daily, weekly, monthly")
		fmpAPIKey = flag.String("fmp-api-key", "", "Financial Modeling Prep API key (or set FMP_API_KEY env var)")
		avAPIKey  = flag.String("av-api-key", "", "Alpha Vantage API key (or set ALPHA_VANTAGE_API_KEY env var)")
	)
	flag.Parse()

	fmt.Printf("📊 HISTORICAL mNAV CALCULATOR\n")
	fmt.Printf("============================\n\n")

	// Get API keys from environment if not provided
	if *fmpAPIKey == "" {
		*fmpAPIKey = os.Getenv("FMP_API_KEY")
	}
	if *avAPIKey == "" {
		*avAPIKey = os.Getenv("ALPHA_VANTAGE_API_KEY")
	}

	if *fmpAPIKey == "" {
		log.Fatalf("❌ Financial Modeling Prep API key is required. Set -fmp-api-key flag or FMP_API_KEY env var")
	}
	if *avAPIKey == "" {
		log.Fatalf("❌ Alpha Vantage API key is required. Set -av-api-key flag or ALPHA_VANTAGE_API_KEY env var")
	}

	// Default end date to today
	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}

	fmt.Printf("🏢 Symbol: %s\n", *symbol)
	fmt.Printf("📅 Period: %s to %s\n", *startDate, *endDate)
	fmt.Printf("⏱️  Interval: %s\n\n", *interval)

	// Initialize API clients
	fmpClient := fmp.NewClient(*fmpAPIKey)
	avClient := alphavantage.NewClient(*avAPIKey)

	// Load required data
	fmt.Printf("📂 Loading historical data...\n")

	// 1. Load Bitcoin transactions
	bitcoinTxs, err := loadBitcoinTransactions(*symbol)
	if err != nil {
		log.Fatalf("❌ Error loading Bitcoin transactions: %v", err)
	}
	fmt.Printf("   ✅ Loaded %d Bitcoin transactions\n", len(bitcoinTxs))

	// 2. Load shares outstanding from Alpha Vantage
	sharesData, err := loadSharesFromAlphaVantage(avClient, *symbol)
	if err != nil {
		log.Fatalf("❌ Error loading shares data: %v", err)
	}
	fmt.Printf("   ✅ Loaded current shares outstanding: %.0f\n", sharesData)

	// 3. Load historical stock prices from Financial Modeling Prep
	stockPrices, err := loadHistoricalStockPricesFromFMP(fmpClient, *symbol, *startDate, *endDate)
	if err != nil {
		log.Fatalf("❌ Error loading stock prices: %v", err)
	}
	fmt.Printf("   ✅ Loaded %d stock price points\n", len(stockPrices))

	// 4. Load historical Bitcoin prices
	btcPrices, err := loadHistoricalBitcoinPrices(*startDate, *endDate)
	if err != nil {
		log.Fatalf("❌ Error loading Bitcoin prices: %v", err)
	}
	fmt.Printf("   ✅ Loaded %d Bitcoin price points\n", len(btcPrices))

	// Calculate historical mNAV
	fmt.Printf("\n📈 Calculating historical mNAV...\n")
	mnavData := calculateHistoricalMNAV(*symbol, bitcoinTxs, sharesData, stockPrices, btcPrices, *startDate, *endDate, *interval)

	fmt.Printf("   ✅ Generated %d mNAV data points\n", len(mnavData.DataPoints))

	// Save results
	if err := saveMNAVData(mnavData, *outputDir); err != nil {
		log.Fatalf("❌ Error saving mNAV data: %v", err)
	}

	// Print summary
	printSummary(mnavData)
}

// Load functions
func loadBitcoinTransactions(symbol string) ([]models.BitcoinTransaction, error) {
	// Try to load from the comprehensive analysis file first
	analysisFile := fmt.Sprintf("data/analysis/%s_comprehensive_bitcoin_analysis.json", symbol)

	data, err := os.ReadFile(analysisFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read analysis file: %w", err)
	}

	var analysis struct {
		AllTransactions []models.BitcoinTransaction `json:"allTransactions"`
	}

	if err := json.Unmarshal(data, &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis file: %w", err)
	}

	return analysis.AllTransactions, nil
}

func loadSharesFromAlphaVantage(client *alphavantage.Client, symbol string) (float64, error) {
	// Get current shares outstanding from Alpha Vantage
	overview, err := client.GetCompanyOverview(symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get company overview: %w", err)
	}

	if overview.SharesOutstanding == 0 {
		return 0, fmt.Errorf("no shares outstanding data available for %s", symbol)
	}

	// Save the overview data for future reference
	if err := saveCompanyOverview(overview, symbol); err != nil {
		fmt.Printf("⚠️  Warning: failed to save company overview: %v\n", err)
	}

	return overview.SharesOutstanding, nil
}

func loadHistoricalStockPricesFromFMP(client *fmp.Client, symbol string, startDate, endDate string) (map[string]float64, error) {
	// Get historical data from Financial Modeling Prep
	histData, err := client.GetHistoricalData(symbol, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	// Convert to map for easy lookup
	priceMap := make(map[string]float64)
	for _, dp := range histData.Historical {
		priceMap[dp.Date] = dp.Close
	}

	// Save the historical data for future reference
	if err := saveHistoricalStockData(histData, symbol); err != nil {
		fmt.Printf("⚠️  Warning: failed to save historical stock data: %v\n", err)
	}

	return priceMap, nil
}

func loadHistoricalBitcoinPrices(startDate, endDate string) (map[string]float64, error) {
	// Load from the historical Bitcoin price file we created
	pattern := fmt.Sprintf("data/bitcoin-prices/historical/bitcoin_historical_*_to_*.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no historical Bitcoin price files found")
	}

	// Use the most recent file
	sort.Strings(files)
	latestFile := files[len(files)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var histData struct {
		Data []struct {
			Date  string  `json:"date"`
			Close float64 `json:"close"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &histData); err != nil {
		return nil, err
	}

	// Convert to map
	priceMap := make(map[string]float64)
	for _, dp := range histData.Data {
		priceMap[dp.Date] = dp.Close
	}

	return priceMap, nil
}

// Calculate historical mNAV
func calculateHistoricalMNAV(
	symbol string,
	bitcoinTxs []models.BitcoinTransaction,
	currentShares float64,
	stockPrices map[string]float64,
	btcPrices map[string]float64,
	startDate, endDate, interval string,
) *HistoricalMNAVData {
	// Parse dates
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	// Sort Bitcoin transactions by date
	sort.Slice(bitcoinTxs, func(i, j int) bool {
		return bitcoinTxs[i].Date.Before(bitcoinTxs[j].Date)
	})

	// Create result structure
	result := &HistoricalMNAVData{
		Symbol:      symbol,
		StartDate:   startDate,
		EndDate:     endDate,
		DataPoints:  []HistoricalMNAVPoint{},
		GeneratedAt: time.Now(),
		Metadata: map[string]interface{}{
			"interval":                   interval,
			"source":                     "SEC filings + FMP + Alpha Vantage",
			"current_shares_outstanding": currentShares,
			"bitcoin_transactions_count": len(bitcoinTxs),
		},
	}

	// Iterate through dates
	for current := start; !current.After(end); current = getNextDate(current, interval) {
		dateStr := current.Format("2006-01-02")

		// Skip if we don't have both stock and Bitcoin prices for this date
		stockPrice, hasStock := stockPrices[dateStr]
		btcPrice, hasBTC := btcPrices[dateStr]
		if !hasStock || !hasBTC {
			continue
		}

		// Calculate Bitcoin holdings at this date
		btcHoldings := calculateBTCHoldingsAtDate(bitcoinTxs, current)
		if btcHoldings == 0 {
			continue // No Bitcoin holdings yet
		}

		// Use current shares outstanding (simplified approach)
		// In a more sophisticated implementation, you might track historical changes
		shares := currentShares

		// Calculate metrics
		marketCap := stockPrice * shares
		btcValue := btcHoldings * btcPrice

		// Calculate mNAV
		mnav, _ := metrics.CalculateMNAV(marketCap, btcHoldings, btcPrice)
		mnavPerShare := btcValue / shares
		premium := ((stockPrice - mnavPerShare) / mnavPerShare) * 100

		// Add data point
		result.DataPoints = append(result.DataPoints, HistoricalMNAVPoint{
			Date:              dateStr,
			StockPrice:        stockPrice,
			BitcoinPrice:      btcPrice,
			BitcoinHoldings:   btcHoldings,
			SharesOutstanding: shares,
			MarketCap:         marketCap,
			BitcoinValue:      btcValue,
			MNAV:              mnav,
			MNAVPerShare:      mnavPerShare,
			Premium:           premium,
		})
	}

	return result
}

func calculateBTCHoldingsAtDate(txs []models.BitcoinTransaction, date time.Time) float64 {
	holdings := 0.0
	for _, tx := range txs {
		if tx.Date.After(date) {
			break
		}
		holdings += tx.BTCPurchased
	}
	return holdings
}

func getNextDate(current time.Time, interval string) time.Time {
	switch interval {
	case "weekly":
		return current.AddDate(0, 0, 7)
	case "monthly":
		return current.AddDate(0, 1, 0)
	default: // daily
		return current.AddDate(0, 0, 1)
	}
}

// Save helper functions
func saveCompanyOverview(overview *alphavantage.ParsedCompanyOverview, symbol string) error {
	dir := "data/company-overview"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_company_overview_%s.json", symbol, time.Now().Format("2006-01-02"))
	filepath := filepath.Join(dir, filename)

	jsonData, err := json.MarshalIndent(overview, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, jsonData, 0644)
}

func saveHistoricalStockData(data *fmp.HistoricalData, symbol string) error {
	dir := "data/stock-prices/historical"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_historical_prices_%s.json", symbol, time.Now().Format("2006-01-02"))
	filepath := filepath.Join(dir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, jsonData, 0644)
}

// Save and display functions
func saveMNAVData(data *HistoricalMNAVData, outputDir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create filename
	filename := fmt.Sprintf("%s_mnav_historical_%s_to_%s.json",
		data.Symbol, data.StartDate, data.EndDate)
	filepath := filepath.Join(outputDir, filename)

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Printf("\n💾 Data saved to: %s\n", filepath)
	return nil
}

func printSummary(data *HistoricalMNAVData) {
	fmt.Printf("\n📊 HISTORICAL mNAV SUMMARY\n")
	fmt.Printf("=========================\n")

	if len(data.DataPoints) == 0 {
		fmt.Printf("No data points generated\n")
		return
	}

	// Find min/max values
	var minMNAV, maxMNAV = data.DataPoints[0].MNAV, data.DataPoints[0].MNAV
	var minPremium, maxPremium = data.DataPoints[0].Premium, data.DataPoints[0].Premium
	var minDate, maxDate string

	for _, dp := range data.DataPoints {
		if dp.MNAV < minMNAV {
			minMNAV = dp.MNAV
			minDate = dp.Date
		}
		if dp.MNAV > maxMNAV {
			maxMNAV = dp.MNAV
			maxDate = dp.Date
		}
		if dp.Premium < minPremium {
			minPremium = dp.Premium
		}
		if dp.Premium > maxPremium {
			maxPremium = dp.Premium
		}
	}

	// Current values (last data point)
	current := data.DataPoints[len(data.DataPoints)-1]

	fmt.Printf("\n📈 Current Values:\n")
	fmt.Printf("   • Date: %s\n", current.Date)
	fmt.Printf("   • Stock Price: $%.2f\n", current.StockPrice)
	fmt.Printf("   • Bitcoin Holdings: %.0f BTC\n", current.BitcoinHoldings)
	fmt.Printf("   • Bitcoin Value: $%.2fB\n", current.BitcoinValue/1e9)
	fmt.Printf("   • mNAV: %.2f\n", current.MNAV)
	fmt.Printf("   • Premium: %.1f%%\n", current.Premium)

	fmt.Printf("\n📊 Historical Range:\n")
	fmt.Printf("   • mNAV Range: %.2f (on %s) to %.2f (on %s)\n", minMNAV, minDate, maxMNAV, maxDate)
	fmt.Printf("   • Premium Range: %.1f%% to %.1f%%\n", minPremium, maxPremium)

	fmt.Printf("\n✅ Analysis complete!\n")
}
