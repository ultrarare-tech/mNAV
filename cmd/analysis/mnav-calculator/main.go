package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jeffreykibler/mNAV/pkg/analysis/metrics"
	"github.com/jeffreykibler/mNAV/pkg/collection/coinmarketcap"
	"github.com/jeffreykibler/mNAV/pkg/collection/yahoo"
	"github.com/jeffreykibler/mNAV/pkg/shared/config"
)

func main() {
	// Define command-line flags
	var (
		symbolsFlag = flag.String("symbols", "MSTR", "Comma-separated list of symbols to analyze")
		verbose     = flag.Bool("verbose", false, "Verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n📊 DATA ANALYSIS - mNAV Calculator\n")
		fmt.Fprintf(os.Stderr, "Calculates net asset value metrics for Bitcoin treasury companies.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Analyze single company\n")
		fmt.Fprintf(os.Stderr, "  %s -symbols MSTR\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Analyze multiple companies\n")
		fmt.Fprintf(os.Stderr, "  %s -symbols MSTR,SMLR,MARA\n", os.Args[0])
	}

	flag.Parse()

	fmt.Printf("📊 DATA ANALYSIS - mNAV Calculator\n")
	fmt.Printf("==================================\n\n")

	// Load company configurations
	companiesConfig, err := config.LoadCompaniesConfig(".")
	if err != nil {
		log.Fatalf("Error loading companies configuration: %v", err)
	}

	// Fetch Bitcoin price
	fmt.Printf("💰 Fetching Bitcoin price...\n")
	btcPrice, err := coinmarketcap.GetBitcoinPrice()
	if err != nil {
		log.Fatalf("Error fetching Bitcoin price: %v", err)
	}

	fmt.Printf("✅ Bitcoin Price: $%.2f (24h change: %.2f%%)\n\n", btcPrice.Price, btcPrice.PercentChange24h)

	// Process companies
	symbols := strings.Split(*symbolsFlag, ",")
	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)

		fmt.Printf("🏢 Analyzing %s...\n", symbol)
		fmt.Printf("------------------------\n")

		companyData, found := companiesConfig.GetCompanyBySymbol(symbol)
		if !found {
			fmt.Printf("❌ No data for company %s, skipping...\n\n", symbol)
			continue
		}

		// Fetch current stock price
		stockPrice, err := yahoo.GetStockPrice(symbol)
		if err != nil {
			fmt.Printf("❌ Error fetching %s stock price: %v\n\n", symbol, err)
			continue
		}

		// Display basic info
		fmt.Printf("📈 Stock Price: $%.2f\n", stockPrice.Price)
		fmt.Printf("💎 Bitcoin Holdings: %.2f BTC\n", companyData.BTCHoldings)
		fmt.Printf("🏦 Market Cap: $%.2f million\n", stockPrice.MarketCap/1000000)

		// Only calculate Bitcoin metrics if the company has Bitcoin holdings
		if companyData.BTCHoldings > 0 {
			// Create Company object for metrics calculation
			company := metrics.Company{
				Symbol:            symbol,
				Name:              companyData.Name,
				StockPrice:        stockPrice.Price,
				MarketCap:         stockPrice.MarketCap,
				BTCHoldings:       companyData.BTCHoldings,
				BTCYield:          companyData.BTCYield,
				OutstandingShares: stockPrice.OutstandingShares,
			}

			// Calculate metrics
			btcMetrics, err := metrics.CalculateMetrics(company, btcPrice.Price)
			if err != nil {
				fmt.Printf("❌ Error calculating metrics for %s: %v\n\n", symbol, err)
				continue
			}

			// Display results
			fmt.Printf("💰 Bitcoin Value: $%.2f million\n", btcMetrics.BTCValue/1000000)
			fmt.Printf("📊 mNAV: %.2f\n", btcMetrics.MNAV)
			fmt.Printf("⏱️  Days to Cover mNAV: %.1f days\n", btcMetrics.DaysToCover)
			fmt.Printf("📈 Daily BTC Accumulation: %.4f BTC\n", btcMetrics.BTCYieldDaily)

			if *verbose {
				fmt.Printf("\n🎯 mNAV Price Targets:\n")
				fmt.Printf("%-8s %-15s\n", "mNAV", "Stock Price")
				fmt.Printf("%-8s %-15s\n", "----", "-----------")

				// Create sorted list of mNAV values
				var mnavValues []float64
				for mnav := range btcMetrics.MNAVPriceTargets {
					mnavValues = append(mnavValues, mnav)
				}
				sort.Float64s(mnavValues)

				// Show key targets
				for _, mnav := range mnavValues {
					if mnav == 1.0 || mnav == 1.5 || mnav == 2.0 || mnav == 3.0 || mnav == 5.0 {
						price := btcMetrics.MNAVPriceTargets[mnav]
						fmt.Printf("%-8.1f $%-14.2f\n", mnav, price)
					}
				}
			}
		} else {
			fmt.Printf("⚪ No Bitcoin holdings - metrics not applicable\n")
		}

		fmt.Printf("\n")
	}

	fmt.Printf("💡 Analysis complete!\n")
	fmt.Printf("   • Use collection commands to gather latest data\n")
	fmt.Printf("   • Use interpretation commands to extract new transactions\n")
}
