package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/analysis/metrics"
	"github.com/jeffreykibler/mNAV/pkg/collection/coinmarketcap"
	"github.com/jeffreykibler/mNAV/pkg/collection/scraper"
	"github.com/jeffreykibler/mNAV/pkg/collection/yahoo"
	"github.com/jeffreykibler/mNAV/pkg/shared/config"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
	"github.com/jeffreykibler/mNAV/pkg/shared/utils"
)

// Delay between API calls to prevent rate limiting
const apiCallDelay = 2 * time.Second

func main() {
	// Define command-line flags
	useScraper := flag.Bool("scrape", false, "Use web scraper to get real-time MSTR Bitcoin holdings")
	symbolsFlag := flag.String("symbols", "MSTR", "Comma-separated list of symbols to process")
	useTransactions := flag.Bool("transactions", false, "Use transaction history to calculate yield")
	flag.Parse()

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	// Determine the project root directory
	// If running from cmd/mnav, we need to go up two levels
	// If running from project root, we don't need to go up
	var envPath string
	var basePath string
	if filepath.Base(cwd) == "mnav" && filepath.Base(filepath.Dir(cwd)) == "cmd" {
		envPath = filepath.Join(cwd, "..", "..", ".env")
		basePath = filepath.Join(cwd, "..", "..")
	} else {
		envPath = ".env"
		basePath = "."
	}

	// Load environment variables
	if err := utils.LoadEnv(envPath); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
		log.Println("Continuing with system environment variables...")
	}

	// Load company configurations
	companiesConfig, err := config.LoadCompaniesConfig(basePath)
	if err != nil {
		log.Fatalf("Error loading companies configuration: %v", err)
	}

	// Load transaction data if enabled
	var transactionsData *models.CombinedTransactions
	if *useTransactions {
		transactionsData, err = models.LoadTransactions(basePath)
		if err != nil {
			log.Printf("Warning: Failed to load transaction data: %v", err)
			log.Println("Continuing without transaction data...")
		} else {
			log.Println("Transaction data loaded successfully")
		}
	}

	// If scraper is enabled, update MSTR Bitcoin holdings
	if *useScraper {
		log.Println("Fetching real-time MSTR Bitcoin holdings data...")
		mstrHoldings, err := scraper.GetMSTRBitcoinHoldings()
		if err != nil {
			log.Printf("Warning: Failed to scrape MSTR Bitcoin holdings: %v", err)
			log.Println("Using JSON values instead.")
		} else {
			// Update the company data with the scraped total BTC holdings
			if mstrHoldings.TotalBTC > 0 {
				mstrData, found := companiesConfig.GetCompanyBySymbol("MSTR")
				if found {
					// Update the BTC holdings field
					mstrData.BTCHoldings = mstrHoldings.TotalBTC
					// Put it back in the config
					companiesConfig.UpdateCompany(mstrData)
					// Save the updated config
					if err := companiesConfig.SaveCompaniesConfig(basePath); err != nil {
						log.Printf("Warning: Failed to save updated company data: %v", err)
					}
					log.Printf("Updated MSTR holdings to %.2f BTC", mstrHoldings.TotalBTC)
				}
			}

			// Try to calculate BTC yield if we have purchase data
			if len(mstrHoldings.Purchases) >= 2 {
				latestPurchase := mstrHoldings.Purchases[0]
				previousPurchase := mstrHoldings.Purchases[1]

				// Only calculate if both purchases have data
				if latestPurchase.BTCPurchased > 0 && previousPurchase.TotalBitcoin > 0 {
					mstrData, found := companiesConfig.GetCompanyBySymbol("MSTR")
					if found {
						// Approximate daily yield (assuming weekly purchases on average)
						// Use 7 days as a rough estimate between purchases
						dailyYield := (latestPurchase.BTCPurchased / previousPurchase.TotalBitcoin) / 7

						// Update the yield if it's reasonable (between 0.01% and 2%)
						if dailyYield >= 0.0001 && dailyYield <= 0.02 {
							mstrData.BTCYield = dailyYield
							// Put it back in the config
							companiesConfig.UpdateCompany(mstrData)
							// Save the updated config
							if err := companiesConfig.SaveCompaniesConfig(basePath); err != nil {
								log.Printf("Warning: Failed to save updated company data: %v", err)
							}
							log.Printf("Updated MSTR daily yield to %.4f%%", mstrData.BTCYield*100)
						}
					}
				}
			}
		}
	}

	// Fetch Bitcoin price
	btcPrice, err := coinmarketcap.GetBitcoinPrice()
	if err != nil {
		log.Fatalf("Error fetching Bitcoin price: %v", err)
	}

	fmt.Printf("--- Bitcoin Price ---\n")
	fmt.Printf("Price: $%.2f\n", btcPrice.Price)
	fmt.Printf("Last Updated: %s\n", btcPrice.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Printf("24h Change: %.2f%%\n\n", btcPrice.PercentChange24h)

	// Process companies - split the comma-separated symbols
	symbols := strings.Split(*symbolsFlag, ",")
	for i, symbol := range symbols {
		companyData, found := companiesConfig.GetCompanyBySymbol(symbol)
		if !found {
			log.Printf("Warning: No data for company %s, skipping...", symbol)
			continue
		}

		// Add delay between API calls, but not before the first one
		if i > 0 {
			fmt.Printf("Waiting %s before fetching next stock data...\n", apiCallDelay)
			time.Sleep(apiCallDelay)
		}

		// If transactions are enabled, update BTCYield from transaction history
		if *useTransactions && transactionsData != nil {
			companyTransactions, found := transactionsData.GetTransactionsForCompany(symbol)
			if found && len(companyTransactions.Transactions) > 0 {
				// If we have transactions, calculate the yield
				dailyYield, err := models.CalculateYieldFromTransactions(
					companyTransactions.Transactions,
					companyData.BTCHoldings,
				)
				if err == nil && dailyYield > 0 {
					log.Printf("Calculated %s yield from transactions: %.4f%%", symbol, dailyYield*100)
					companyData.BTCYield = dailyYield
					// Update the config
					companiesConfig.UpdateCompany(companyData)
				} else if err != nil {
					log.Printf("Warning: Failed to calculate yield from transactions: %v", err)
				}
			}
		}

		// Fetch stock price
		stockPrice, err := yahoo.GetStockPrice(symbol)
		if err != nil {
			log.Printf("Error fetching %s stock price: %v", symbol, err)
			continue
		}

		// Manual override for SMLR since Yahoo Finance API sometimes fails for it
		if symbol == "SMLR" && (stockPrice.MarketCap == 0 || stockPrice.OutstandingShares == 0) {
			// Use the values from our config
			if stockPrice.OutstandingShares == 0 && companyData.OutstandingShares > 0 {
				stockPrice.OutstandingShares = companyData.OutstandingShares
				log.Printf("Using config outstanding shares for SMLR: %.2f", stockPrice.OutstandingShares)
			}

			// Calculate market cap from price and shares or use config
			if stockPrice.MarketCap == 0 {
				if stockPrice.Price > 0 && stockPrice.OutstandingShares > 0 {
					stockPrice.MarketCap = stockPrice.Price * stockPrice.OutstandingShares
					log.Printf("Calculated market cap for SMLR: $%.2f million", stockPrice.MarketCap/1000000)
				} else if companyData.MarketCap > 0 {
					stockPrice.MarketCap = companyData.MarketCap
					log.Printf("Using config market cap for SMLR: $%.2f million", stockPrice.MarketCap/1000000)
				}

				// Ensure we set the company market cap
				companyData.MarketCap = stockPrice.MarketCap
			}
		}

		fmt.Printf("--- %s (%s) ---\n", companyData.Name, symbol)
		fmt.Printf("Stock Price: $%.2f\n", stockPrice.Price)

		// Calculate and set market cap
		if stockPrice.MarketCap > 0 {
			// Market cap from Yahoo Finance (either directly fetched or calculated from shares)
			companyData.MarketCap = stockPrice.MarketCap
			fmt.Printf("Market Cap: $%.2f million (from Yahoo Finance)\n", companyData.MarketCap/1000000)
		} else if companyData.OutstandingShares > 0 {
			// If Yahoo Finance didn't return market cap but we have shares from config
			companyData.MarketCap = stockPrice.Price * companyData.OutstandingShares
			fmt.Printf("Market Cap: $%.2f million (calculated from config shares)\n", companyData.MarketCap/1000000)
		} else {
			fmt.Printf("Market Cap: $%.2f million\n", companyData.MarketCap/1000000)
		}

		// If we have outstanding shares from Yahoo Finance, update our data
		if stockPrice.OutstandingShares > 0 {
			fmt.Printf("Shares Outstanding: %.2f million (from Yahoo Finance)\n", stockPrice.OutstandingShares/1000000)
			companyData.OutstandingShares = stockPrice.OutstandingShares
		} else if companyData.OutstandingShares > 0 {
			fmt.Printf("Shares Outstanding: %.2f million (from config)\n", companyData.OutstandingShares/1000000)
			// If Yahoo didn't return shares but stock price is available, update MarketCap
			if stockPrice.MarketCap == 0 && stockPrice.Price > 0 {
				stockPrice.MarketCap = stockPrice.Price * companyData.OutstandingShares
			}
		} else {
			fmt.Printf("Shares Outstanding: %.2f million (from config)\n", companyData.OutstandingShares/1000000)
		}

		// Display Bitcoin holdings and yield
		fmt.Printf("Bitcoin Holdings: %.2f BTC\n", companyData.BTCHoldings)
		fmt.Printf("Daily BTC Yield: %.4f%%\n", companyData.BTCYield*100)
		if !companyData.LastUpdated.IsZero() {
			fmt.Printf("Last Updated: %s\n", companyData.LastUpdated.Format("2006-01-02 15:04:05"))
		}

		// Create Company object for metrics calculation
		company := metrics.Company{
			Symbol:            symbol,
			Name:              companyData.Name,
			StockPrice:        stockPrice.Price,
			MarketCap:         companyData.MarketCap,
			BTCHoldings:       companyData.BTCHoldings,
			BTCYield:          companyData.BTCYield,
			OutstandingShares: companyData.OutstandingShares,
		}

		// Special handling for SMLR
		if symbol == "SMLR" {
			// Force the use of the correct market cap and shares
			if company.MarketCap < 1000 { // If value is too small, it's likely incorrect
				company.MarketCap = 501041000 // Hard-coded correct value
				log.Printf("Using hard-coded market cap for SMLR: $%.2f million", company.MarketCap/1000000)
			} else {
				company.MarketCap = companyData.MarketCap
				log.Printf("SMLR Market Cap from config: $%.2f million", company.MarketCap/1000000)
			}

			if company.OutstandingShares < 1000 { // If value is too small, it's likely incorrect
				company.OutstandingShares = 11150000 // Hard-coded correct value
				log.Printf("Using hard-coded shares for SMLR: %.2f million", company.OutstandingShares/1000000)
			}
		}

		// Special handling for MARA
		if symbol == "MARA" {
			// Force the use of the correct market cap and shares
			if company.MarketCap < 1000 { // If value is too small, it's likely incorrect
				company.MarketCap = 5510000000 // Hard-coded correct value from our research
				log.Printf("Using hard-coded market cap for MARA: $%.2f million", company.MarketCap/1000000)
			} else {
				company.MarketCap = companyData.MarketCap
				log.Printf("MARA Market Cap from config: $%.2f million", company.MarketCap/1000000)
			}

			if company.OutstandingShares < 1000 { // If value is too small, it's likely incorrect
				company.OutstandingShares = 351930000 // Hard-coded correct value from our research
				log.Printf("Using hard-coded shares for MARA: %.2f million", company.OutstandingShares/1000000)
			}
		}

		// Special handling for Metaplanet (Tokyo Stock Exchange)
		if symbol == "3350.T" {
			// Force the use of the correct market cap and shares
			if company.MarketCap < 1000 { // If value is too small, it's likely incorrect
				company.MarketCap = 3001311724 // Hard-coded correct value from our research (converted from JPY)
				log.Printf("Using hard-coded market cap for Metaplanet: $%.2f million", company.MarketCap/1000000)
			} else {
				company.MarketCap = companyData.MarketCap
				log.Printf("Metaplanet Market Cap from config: $%.2f million", company.MarketCap/1000000)
			}

			if company.OutstandingShares < 1000 { // If value is too small, it's likely incorrect
				company.OutstandingShares = 40860000 // Hard-coded correct value from our research
				log.Printf("Using hard-coded shares for Metaplanet: %.2f million", company.OutstandingShares/1000000)
			}
		}

		// Calculate metrics
		var btcMetrics *metrics.BitcoinMetrics
		var metricsErr error

		if companyData.BTCHoldings > 0 {
			// Only calculate Bitcoin metrics if the company has Bitcoin holdings
			btcMetrics, metricsErr = metrics.CalculateMetrics(company, btcPrice.Price)
			if metricsErr != nil {
				log.Printf("Error calculating metrics for %s: %v", symbol, metricsErr)
				continue
			}

			// Print metrics
			fmt.Printf("Bitcoin Value: $%.2f million\n", btcMetrics.BTCValue/1000000)
			fmt.Printf("Daily BTC Accumulation: %.4f BTC\n", btcMetrics.BTCYieldDaily)
			fmt.Printf("mNAV: %.2f\n", btcMetrics.MNAV)
			fmt.Printf("Days to Cover mNAV: %.2f\n\n", btcMetrics.DaysToCover)

			// Store mNAV price targets in company data
			companyData.UpdateMNAVPriceTargets(btcMetrics.MNAVPriceTargets)

			// Store days to cover in company data
			companyData.DaysToCover = btcMetrics.DaysToCover

			// Print mNAV price targets
			fmt.Println("mNAV Price Targets:")
			fmt.Println("------------------------------------------")
			fmt.Printf("%-8s %-15s\n", "mNAV", "Stock Price Target")
			fmt.Println("------------------------------------------")

			// Create a sorted list of mNAV values
			var mnavValues []float64
			for mnav := range btcMetrics.MNAVPriceTargets {
				mnavValues = append(mnavValues, mnav)
			}
			sort.Float64s(mnavValues)

			// Print each mNAV price target in order
			for _, mnav := range mnavValues {
				price := btcMetrics.MNAVPriceTargets[mnav]
				fmt.Printf("%-8.2f $%-15.2f\n", mnav, price)
			}
			fmt.Println()

			// Update company data with market cap and price targets
			companiesConfig.UpdateCompany(companyData)
			if err := companiesConfig.SaveCompaniesConfig(basePath); err != nil {
				log.Printf("Warning: Failed to save updated company data: %v", err)
			}
		} else {
			// For companies without Bitcoin holdings, just print a message
			fmt.Printf("No Bitcoin metrics available (no BTC holdings)\n\n")
		}
	}
}
