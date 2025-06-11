package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BitcoinPriceData represents a single day's Bitcoin price data
type BitcoinPriceData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// BitcoinHistoricalData represents the structure of historical Bitcoin data files
type BitcoinHistoricalData struct {
	Symbol    string             `json:"symbol"`
	StartDate string             `json:"start_date"`
	EndDate   string             `json:"end_date"`
	Data      []BitcoinPriceData `json:"data"`
}

func main() {
	fmt.Println("🪙 BITCOIN HISTORICAL DATA UPDATER")
	fmt.Println("===================================")

	var csvFile string

	// Check if a custom file path was provided
	if len(os.Args) > 1 {
		csvFile = os.Args[1]
	} else {
		csvFile = "data/bitcoin-prices/Bitcoin_5_1_2020-6_1_2025_historical_data_coinmarketcap.csv"
	}

	// Check if file exists
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		log.Fatalf("❌ Bitcoin CSV file not found: %s", csvFile)
	}

	fmt.Printf("📂 Processing file: %s\n", csvFile)

	// Load Bitcoin data from CoinMarketCap CSV file
	bitcoinData, err := loadFromCoinMarketCapCSV(csvFile)
	if err != nil {
		log.Fatalf("❌ Error loading Bitcoin data: %v", err)
	}

	fmt.Printf("✅ Loaded %d Bitcoin price records from CoinMarketCap CSV\n", len(bitcoinData))

	if len(bitcoinData) == 0 {
		log.Fatal("❌ No Bitcoin data found in CSV file")
	}

	// Sort data by date (oldest first)
	sort.Slice(bitcoinData, func(i, j int) bool {
		return bitcoinData[i].Date < bitcoinData[j].Date
	})

	fmt.Printf("📅 Date range: %s to %s\n", bitcoinData[0].Date, bitcoinData[len(bitcoinData)-1].Date)

	// Create historical JSON files
	err = updateHistoricalFiles(bitcoinData)
	if err != nil {
		log.Fatalf("❌ Error updating historical files: %v", err)
	}

	// Update the CSV exporter
	bitcoinFilePath := fmt.Sprintf("data/bitcoin-prices/historical/bitcoin_historical_%s_to_%s_coinmarketcap.json",
		bitcoinData[0].Date, bitcoinData[len(bitcoinData)-1].Date)

	fmt.Println("\n📝 Updating CSV exporter to use new Bitcoin data...")
	err = updateCSVExporter(bitcoinFilePath)
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not update CSV exporter automatically: %v\n", err)
		fmt.Printf("💡 Manually update the loadBitcoinData function in cmd/utilities/csv-exporter/main.go\n")
		fmt.Printf("   to use: %s\n", bitcoinFilePath)
	}

	fmt.Println("\n🎉 Bitcoin historical data update complete!")
}

// loadFromCoinMarketCapCSV loads Bitcoin price data from the CoinMarketCap CSV file
func loadFromCoinMarketCapCSV(filename string) ([]BitcoinPriceData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';' // CoinMarketCap CSV uses semicolon as delimiter

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or missing data")
	}

	var bitcoinData []BitcoinPriceData

	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 11 {
			continue // Skip incomplete records
		}

		// Parse the date from timeClose field (index 1)
		timeStr := strings.Trim(record[1], "\"")
		parsedTime, err := time.Parse("2006-01-02T15:04:05.000Z", timeStr)
		if err != nil {
			continue // Skip records with invalid dates
		}
		dateStr := parsedTime.Format("2006-01-02")

		// Parse price fields (removing quotes)
		openStr := strings.Trim(record[5], "\"")
		open, err := strconv.ParseFloat(openStr, 64)
		if err != nil {
			continue
		}

		highStr := strings.Trim(record[6], "\"")
		high, err := strconv.ParseFloat(highStr, 64)
		if err != nil {
			continue
		}

		lowStr := strings.Trim(record[7], "\"")
		low, err := strconv.ParseFloat(lowStr, 64)
		if err != nil {
			continue
		}

		closeStr := strings.Trim(record[8], "\"")
		closePrice, err := strconv.ParseFloat(closeStr, 64)
		if err != nil {
			continue
		}

		volumeStr := strings.Trim(record[9], "\"")
		volume, err := strconv.ParseFloat(volumeStr, 64)
		if err != nil {
			continue
		}

		bitcoinData = append(bitcoinData, BitcoinPriceData{
			Date:   dateStr,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		})
	}

	return bitcoinData, nil
}

// updateHistoricalFiles creates or updates historical JSON files with the new data
func updateHistoricalFiles(bitcoinData []BitcoinPriceData) error {
	if len(bitcoinData) == 0 {
		return fmt.Errorf("no Bitcoin data to update")
	}

	// Ensure the historical directory exists
	historicalDir := "data/bitcoin-prices/historical"
	if err := os.MkdirAll(historicalDir, 0755); err != nil {
		return fmt.Errorf("error creating historical directory: %w", err)
	}

	// Determine date range
	startDate := bitcoinData[len(bitcoinData)-1].Date // CoinMarketCap data is reverse chronological
	endDate := bitcoinData[0].Date

	// Create filename for the comprehensive dataset
	filename := fmt.Sprintf("bitcoin_historical_%s_to_%s_coinmarketcap.json", startDate, endDate)
	filePath := filepath.Join(historicalDir, filename)

	// Create the historical data structure
	historicalData := BitcoinHistoricalData{
		Symbol:    "BTC",
		StartDate: startDate,
		EndDate:   endDate,
		Data:      make([]BitcoinPriceData, len(bitcoinData)),
	}

	// Reverse the data to be chronological (oldest first)
	for i, j := 0, len(bitcoinData)-1; i < len(bitcoinData); i, j = i+1, j-1 {
		historicalData.Data[i] = bitcoinData[j]
	}

	// Write the comprehensive file
	if err := writeJSONFile(filePath, historicalData); err != nil {
		return fmt.Errorf("error writing comprehensive file: %w", err)
	}

	fmt.Printf("✅ Created comprehensive file: %s\n", filename)

	// Also create yearly files for better organization
	yearlyData := make(map[string][]BitcoinPriceData)
	for _, data := range historicalData.Data {
		year := data.Date[:4]
		yearlyData[year] = append(yearlyData[year], data)
	}

	for year, data := range yearlyData {
		if len(data) == 0 {
			continue
		}

		yearFilename := fmt.Sprintf("bitcoin_historical_%s_coinmarketcap.json", year)
		yearFilePath := filepath.Join(historicalDir, yearFilename)

		yearHistoricalData := BitcoinHistoricalData{
			Symbol:    "BTC",
			StartDate: data[0].Date,
			EndDate:   data[len(data)-1].Date,
			Data:      data,
		}

		if err := writeJSONFile(yearFilePath, yearHistoricalData); err != nil {
			return fmt.Errorf("error writing yearly file for %s: %w", year, err)
		}

		fmt.Printf("✅ Created yearly file: %s (%d records)\n", yearFilename, len(data))
	}

	// Update the CSV exporter's loadBitcoinData function to use the new comprehensive file
	fmt.Println("\n📝 Updating CSV exporter to use new Bitcoin data...")
	if err := updateCSVExporter(filePath); err != nil {
		fmt.Printf("⚠️  Warning: Could not update CSV exporter automatically: %v\n", err)
		fmt.Printf("💡 Manually update the loadBitcoinData function in cmd/utilities/csv-exporter/main.go\n")
		fmt.Printf("   to use: %s\n", filePath)
	} else {
		fmt.Println("✅ CSV exporter updated to use new Bitcoin data")
	}

	return nil
}

// writeJSONFile writes the historical data to a JSON file
func writeJSONFile(filePath string, data BitcoinHistoricalData) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", filePath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}

	return nil
}

// updateCSVExporter updates the CSV exporter to use the new comprehensive Bitcoin data file
func updateCSVExporter(bitcoinFilePath string) error {
	csvExporterPath := "cmd/utilities/csv-exporter/main.go"

	// Read the current file
	content, err := os.ReadFile(csvExporterPath)
	if err != nil {
		return fmt.Errorf("error reading CSV exporter file: %w", err)
	}

	// Find and replace the CoinMarketCap CSV file reference with the new JSON file
	oldContent := string(content)

	// Look for the CoinMarketCap CSV loading section and replace it
	newContent := strings.ReplaceAll(oldContent,
		`// First try to load from the CoinMarketCap CSV file
	csvFile := "data/bitcoin-prices/Bitcoin_5_1_2020-6_1_2025_historical_data_coinmarketcap.csv"
	if _, err := os.Stat(csvFile); err == nil {
		return loadFromCoinMarketCapCSV(csvFile)
	}`,
		fmt.Sprintf(`// First try to load from the comprehensive CoinMarketCap JSON file
	jsonFile := "%s"
	if _, err := os.Stat(jsonFile); err == nil {
		return loadFromBitcoinHistoricalJSON(jsonFile)
	}

	// Fallback to CoinMarketCap CSV file
	csvFile := "data/bitcoin-prices/Bitcoin_5_1_2020-6_1_2025_historical_data_coinmarketcap.csv"
	if _, err := os.Stat(csvFile); err == nil {
		return loadFromCoinMarketCapCSV(csvFile)
	}`, bitcoinFilePath))

	if newContent == oldContent {
		return fmt.Errorf("could not find the expected pattern to replace in CSV exporter")
	}

	// Write the updated content back
	if err := os.WriteFile(csvExporterPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("error writing updated CSV exporter file: %w", err)
	}

	return nil
}
