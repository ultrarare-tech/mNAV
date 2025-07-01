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

	"github.com/jeffreykibler/mNAV/pkg/collection/yahoo"
)

// StockDataCollection represents the stock data file format
type StockDataCollection struct {
	Symbol           string                `json:"symbol"`
	CollectedAt      time.Time             `json:"collected_at"`
	HistoricalPrices *HistoricalPricesData `json:"historical_prices,omitempty"`
	CompanyProfile   interface{}           `json:"company_profile,omitempty"`
	CompanyOverview  interface{}           `json:"company_overview,omitempty"`
	CurrentPrice     float64               `json:"current_price,omitempty"`
	Sources          map[string]string     `json:"sources"`
}

type HistoricalPricesData struct {
	Symbol     string           `json:"symbol"`
	Historical []HistoricalData `json:"historical"`
}

type HistoricalData struct {
	Date             string  `json:"date"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	AdjClose         float64 `json:"adjClose"`
	Volume           float64 `json:"volume"`
	UnadjustedVolume float64 `json:"unadjustedVolume"`
	Change           float64 `json:"change"`
	ChangePercent    float64 `json:"changePercent"`
	Vwap             float64 `json:"vwap"`
	Label            string  `json:"label"`
	ChangeOverTime   float64 `json:"changeOverTime"`
}

func main() {
	var (
		symbol    = flag.String("symbol", "MSTR", "Stock symbol")
		startDate = flag.String("start", "", "Start date for new data (YYYY-MM-DD), defaults to 7 days ago")
		endDate   = flag.String("end", "", "End date for new data (YYYY-MM-DD), defaults to today")
		outputDir = flag.String("output", "data/stock-data", "Output directory")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *verbose {
		fmt.Printf("📈 STOCK DATA UPDATER\n")
		fmt.Printf("=====================\n\n")
		fmt.Printf("🏢 Symbol: %s\n", *symbol)
		fmt.Printf("💾 Output Directory: %s\n\n", *outputDir)
	}

	// Set default dates
	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}
	if *startDate == "" {
		*startDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}

	// Load existing stock data file
	latestFile, err := findLatestStockDataFile(*symbol, *outputDir)
	if err != nil {
		log.Fatalf("❌ Error finding latest stock data file: %v", err)
	}

	if *verbose {
		fmt.Printf("📂 Loading existing data from: %s\n", latestFile)
	}

	existingData, err := loadStockDataFile(latestFile)
	if err != nil {
		log.Fatalf("❌ Error loading existing data: %v", err)
	}

	// Get the latest date in existing data
	latestExistingDate := getLatestDateInData(existingData)
	if *verbose {
		fmt.Printf("📅 Latest existing data: %s\n", latestExistingDate)
	}

	// Calculate the date range for new data
	startTime, _ := time.Parse("2006-01-02", latestExistingDate)
	startTime = startTime.AddDate(0, 0, 1) // Start from the day after latest existing data

	endTime, _ := time.Parse("2006-01-02", *endDate)

	if !startTime.Before(endTime) && !startTime.Equal(endTime) {
		if *verbose {
			fmt.Printf("✅ Data is already up to date (latest: %s)\n", latestExistingDate)
		}
		return
	}

	fetchStart := startTime.Format("2006-01-02")
	fetchEnd := endTime.Format("2006-01-02")

	if *verbose {
		fmt.Printf("🔄 Fetching new data from %s to %s\n", fetchStart, fetchEnd)
	}

	// Fetch new historical data from Yahoo Finance
	// Calculate range string for Yahoo Finance API
	days := int(endTime.Sub(startTime).Hours() / 24)
	rangeStr := fmt.Sprintf("%dd", days)

	newData, err := yahoo.GetHistoricalData(*symbol, rangeStr)
	if err != nil {
		log.Fatalf("❌ Error fetching new data from Yahoo Finance: %v", err)
	}

	if len(newData.Data) == 0 {
		if *verbose {
			fmt.Printf("✅ No new data available\n")
		}
		return
	}

	if *verbose {
		fmt.Printf("✅ Fetched %d new data points\n", len(newData.Data))
	}

	// Merge new data with existing data
	mergedData := mergeHistoricalData(existingData, newData)

	// Get current price for the current_price field
	currentPrice, err := getCurrentPrice(*symbol)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not fetch current price: %v\n", err)
	} else {
		mergedData.CurrentPrice = currentPrice
		if *verbose {
			fmt.Printf("💰 Updated current price: $%.2f\n", currentPrice)
		}
	}

	// Update sources
	if mergedData.Sources == nil {
		mergedData.Sources = make(map[string]string)
	}
	mergedData.Sources["historical_prices"] = "Yahoo Finance (updated)"
	mergedData.CollectedAt = time.Now()

	// Save the updated data
	outputFile := filepath.Join(*outputDir, fmt.Sprintf("%s_stock_data_UPDATED_%s.json",
		*symbol, time.Now().Format("2006-01-02_15-04-05")))

	if err := saveStockDataFile(mergedData, outputFile); err != nil {
		log.Fatalf("❌ Error saving updated data: %v", err)
	}

	if *verbose {
		fmt.Printf("💾 Updated data saved to: %s\n", outputFile)
		fmt.Printf("✅ Stock data update complete!\n")
	} else {
		fmt.Printf("Updated stock data saved to: %s\n", outputFile)
	}
}

func findLatestStockDataFile(symbol, outputDir string) (string, error) {
	pattern := filepath.Join(outputDir, fmt.Sprintf("%s_stock_data_*.json", symbol))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no stock data files found for %s", symbol)
	}

	// Sort files and return the latest
	sort.Strings(files)
	return files[len(files)-1], nil
}

func loadStockDataFile(filename string) (*StockDataCollection, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var stockData StockDataCollection
	if err := json.Unmarshal(data, &stockData); err != nil {
		return nil, err
	}

	return &stockData, nil
}

func saveStockDataFile(data *StockDataCollection, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

func getLatestDateInData(data *StockDataCollection) string {
	if data.HistoricalPrices == nil || len(data.HistoricalPrices.Historical) == 0 {
		return "2020-01-01" // Default to a very old date
	}

	// Find the latest date (data should be sorted with latest first)
	latestDate := data.HistoricalPrices.Historical[0].Date
	for _, point := range data.HistoricalPrices.Historical {
		if point.Date > latestDate {
			latestDate = point.Date
		}
	}

	return latestDate
}

func mergeHistoricalData(existingData *StockDataCollection, newData *yahoo.HistoricalData) *StockDataCollection {
	if existingData.HistoricalPrices == nil {
		existingData.HistoricalPrices = &HistoricalPricesData{
			Symbol:     existingData.Symbol,
			Historical: []HistoricalData{},
		}
	}

	// Convert Yahoo data to our format and add to existing data
	for _, yahooPoint := range newData.Data {
		newPoint := HistoricalData{
			Date:             yahooPoint.Date,
			Open:             yahooPoint.Open,
			High:             yahooPoint.High,
			Low:              yahooPoint.Low,
			Close:            yahooPoint.Close,
			AdjClose:         yahooPoint.AdjClose,
			Volume:           float64(yahooPoint.Volume),
			UnadjustedVolume: float64(yahooPoint.Volume),
			Change:           0,                // Will be calculated if needed
			ChangePercent:    0,                // Will be calculated if needed
			Vwap:             yahooPoint.Close, // Approximation
			Label:            yahooPoint.Date,
			ChangeOverTime:   0, // Will be calculated if needed
		}

		existingData.HistoricalPrices.Historical = append(existingData.HistoricalPrices.Historical, newPoint)
	}

	// Sort data by date (latest first, matching existing format)
	sort.Slice(existingData.HistoricalPrices.Historical, func(i, j int) bool {
		return existingData.HistoricalPrices.Historical[i].Date > existingData.HistoricalPrices.Historical[j].Date
	})

	return existingData
}

func getCurrentPrice(symbol string) (float64, error) {
	stockPrice, err := yahoo.GetStockPrice(symbol)
	if err != nil {
		return 0, err
	}
	return stockPrice.Price, nil
}
