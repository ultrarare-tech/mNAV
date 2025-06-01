package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/collection/alphavantage"
	"github.com/jeffreykibler/mNAV/pkg/collection/fmp"
)

// StockDataCollection represents collected stock data from multiple sources
type StockDataCollection struct {
	Symbol           string                              `json:"symbol"`
	CollectedAt      time.Time                           `json:"collected_at"`
	HistoricalPrices *fmp.HistoricalData                 `json:"historical_prices,omitempty"`
	CompanyProfile   *fmp.CompanyProfile                 `json:"company_profile,omitempty"`
	CompanyOverview  *alphavantage.ParsedCompanyOverview `json:"company_overview,omitempty"`
	CurrentPrice     float64                             `json:"current_price,omitempty"`
	Sources          map[string]string                   `json:"sources"`
}

func main() {
	var (
		symbol      = flag.String("symbol", "MSTR", "Stock symbol")
		startDate   = flag.String("start", "", "Start date for historical data (YYYY-MM-DD), defaults to 1 year ago")
		endDate     = flag.String("end", "", "End date for historical data (YYYY-MM-DD), defaults to today")
		outputDir   = flag.String("output", "data/stock-data", "Output directory")
		fmpAPIKey   = flag.String("fmp-api-key", "", "Financial Modeling Prep API key (or set FMP_API_KEY env var)")
		avAPIKey    = flag.String("av-api-key", "", "Alpha Vantage API key (or set ALPHA_VANTAGE_API_KEY env var)")
		skipHist    = flag.Bool("skip-historical", false, "Skip historical price collection")
		skipCurrent = flag.Bool("skip-current", false, "Skip current data collection")
	)
	flag.Parse()

	fmt.Printf("📈 STOCK DATA COLLECTOR\n")
	fmt.Printf("======================\n\n")

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

	// Set default dates
	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}
	if *startDate == "" {
		*startDate = time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	}

	fmt.Printf("🏢 Symbol: %s\n", *symbol)
	fmt.Printf("📅 Historical Period: %s to %s\n", *startDate, *endDate)
	fmt.Printf("💾 Output Directory: %s\n\n", *outputDir)

	// Initialize API clients
	fmpClient := fmp.NewClient(*fmpAPIKey)
	avClient := alphavantage.NewClient(*avAPIKey)

	// Initialize collection structure
	collection := &StockDataCollection{
		Symbol:      *symbol,
		CollectedAt: time.Now(),
		Sources: map[string]string{
			"historical_prices": "Financial Modeling Prep",
			"company_profile":   "Financial Modeling Prep",
			"company_overview":  "Alpha Vantage",
			"current_price":     "Financial Modeling Prep",
		},
	}

	// Collect data
	fmt.Printf("📊 Collecting stock data...\n")

	// 1. Get historical prices from FMP
	if !*skipHist {
		fmt.Printf("   📈 Fetching historical prices from FMP...\n")
		histData, err := fmpClient.GetHistoricalData(*symbol, *startDate, *endDate)
		if err != nil {
			fmt.Printf("   ❌ Error fetching historical data: %v\n", err)
		} else {
			collection.HistoricalPrices = histData
			fmt.Printf("   ✅ Retrieved %d historical price points\n", len(histData.Historical))
		}
	}

	// 2. Get current price and company profile from FMP
	if !*skipCurrent {
		fmt.Printf("   💰 Fetching current price from FMP...\n")
		currentPrice, err := fmpClient.GetCurrentPrice(*symbol)
		if err != nil {
			fmt.Printf("   ❌ Error fetching current price: %v\n", err)
		} else {
			collection.CurrentPrice = currentPrice
			fmt.Printf("   ✅ Current price: $%.2f\n", currentPrice)
		}

		fmt.Printf("   🏢 Fetching company profile from FMP...\n")
		profile, err := fmpClient.GetCompanyProfile(*symbol)
		if err != nil {
			fmt.Printf("   ❌ Error fetching company profile: %v\n", err)
		} else {
			collection.CompanyProfile = profile
			fmt.Printf("   ✅ Company profile retrieved: %s\n", profile.CompanyName)
			fmt.Printf("      • Market Cap: $%.2fB\n", float64(profile.MktCap)/1e9)
			fmt.Printf("      • Industry: %s\n", profile.Industry)
		}
	}

	// 3. Get company overview from Alpha Vantage (includes shares outstanding)
	fmt.Printf("   📋 Fetching company overview from Alpha Vantage...\n")
	overview, err := avClient.GetCompanyOverview(*symbol)
	if err != nil {
		fmt.Printf("   ❌ Error fetching company overview: %v\n", err)
	} else {
		collection.CompanyOverview = overview
		fmt.Printf("   ✅ Company overview retrieved\n")
		fmt.Printf("      • Shares Outstanding: %.0f\n", overview.SharesOutstanding)
		fmt.Printf("      • Market Cap (AV): $%.2fB\n", overview.MarketCapitalization/1e9)
		fmt.Printf("      • P/E Ratio: %.2f\n", overview.PERatio)
	}

	// Save collected data
	fmt.Printf("\n💾 Saving collected data...\n")
	if err := saveStockData(collection, *outputDir); err != nil {
		log.Fatalf("❌ Error saving data: %v", err)
	}

	// Print summary
	printSummary(collection)
}

func saveStockData(data *StockDataCollection, outputDir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create filename with timestamp
	timestamp := data.CollectedAt.Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_stock_data_%s.json", data.Symbol, timestamp)
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

	fmt.Printf("   ✅ Data saved to: %s\n", filepath)

	// Also save individual files for easy access
	if err := saveIndividualFiles(data, outputDir); err != nil {
		fmt.Printf("   ⚠️  Warning: failed to save individual files: %v\n", err)
	}

	return nil
}

func saveIndividualFiles(data *StockDataCollection, outputDir string) error {
	timestamp := data.CollectedAt.Format("2006-01-02")

	// Save historical prices
	if data.HistoricalPrices != nil {
		histDir := filepath.Join(outputDir, "historical")
		if err := os.MkdirAll(histDir, 0755); err == nil {
			filename := fmt.Sprintf("%s_historical_prices_%s.json", data.Symbol, timestamp)
			path := filepath.Join(histDir, filename)
			jsonData, _ := json.MarshalIndent(data.HistoricalPrices, "", "  ")
			os.WriteFile(path, jsonData, 0644)
		}
	}

	// Save company profile
	if data.CompanyProfile != nil {
		profileDir := filepath.Join(outputDir, "profiles")
		if err := os.MkdirAll(profileDir, 0755); err == nil {
			filename := fmt.Sprintf("%s_profile_%s.json", data.Symbol, timestamp)
			path := filepath.Join(profileDir, filename)
			jsonData, _ := json.MarshalIndent(data.CompanyProfile, "", "  ")
			os.WriteFile(path, jsonData, 0644)
		}
	}

	// Save company overview
	if data.CompanyOverview != nil {
		overviewDir := filepath.Join(outputDir, "overviews")
		if err := os.MkdirAll(overviewDir, 0755); err == nil {
			filename := fmt.Sprintf("%s_overview_%s.json", data.Symbol, timestamp)
			path := filepath.Join(overviewDir, filename)
			jsonData, _ := json.MarshalIndent(data.CompanyOverview, "", "  ")
			os.WriteFile(path, jsonData, 0644)
		}
	}

	return nil
}

func printSummary(data *StockDataCollection) {
	fmt.Printf("\n📊 STOCK DATA COLLECTION SUMMARY\n")
	fmt.Printf("================================\n")

	fmt.Printf("\n🏢 Company: %s\n", data.Symbol)
	if data.CompanyProfile != nil {
		fmt.Printf("   • Name: %s\n", data.CompanyProfile.CompanyName)
		fmt.Printf("   • Industry: %s\n", data.CompanyProfile.Industry)
		fmt.Printf("   • Sector: %s\n", data.CompanyProfile.Sector)
	}

	fmt.Printf("\n💰 Financial Data:\n")
	if data.CurrentPrice > 0 {
		fmt.Printf("   • Current Price: $%.2f\n", data.CurrentPrice)
	}
	if data.CompanyProfile != nil {
		fmt.Printf("   • Market Cap (FMP): $%.2fB\n", float64(data.CompanyProfile.MktCap)/1e9)
	}
	if data.CompanyOverview != nil {
		fmt.Printf("   • Market Cap (AV): $%.2fB\n", data.CompanyOverview.MarketCapitalization/1e9)
		fmt.Printf("   • Shares Outstanding: %.0f\n", data.CompanyOverview.SharesOutstanding)
		fmt.Printf("   • P/E Ratio: %.2f\n", data.CompanyOverview.PERatio)
		fmt.Printf("   • Beta: %.2f\n", data.CompanyOverview.Beta)
	}

	fmt.Printf("\n📈 Historical Data:\n")
	if data.HistoricalPrices != nil {
		fmt.Printf("   • Price Points: %d\n", len(data.HistoricalPrices.Historical))
		if len(data.HistoricalPrices.Historical) > 0 {
			first := data.HistoricalPrices.Historical[len(data.HistoricalPrices.Historical)-1]
			last := data.HistoricalPrices.Historical[0]
			fmt.Printf("   • Date Range: %s to %s\n", first.Date, last.Date)
			fmt.Printf("   • Price Range: $%.2f to $%.2f\n", first.Close, last.Close)
		}
	}

	fmt.Printf("\n📋 Data Sources:\n")
	for dataType, source := range data.Sources {
		fmt.Printf("   • %s: %s\n", dataType, source)
	}

	fmt.Printf("\n✅ Collection complete!\n")
}
