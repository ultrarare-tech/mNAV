package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/ultrarare-tech/mNAV/pkg/portfolio/analyzer"
	"github.com/ultrarare-tech/mNAV/pkg/portfolio/models"
)

func main() {
	var (
		latest     = flag.Bool("latest", false, "Analyze latest portfolio snapshot")
		date       = flag.String("date", "", "Analyze specific date (YYYY-MM-DD)")
		historical = flag.Bool("historical", false, "Show historical portfolio summary")
		mnav       = flag.Bool("mnav", true, "Include mNAV-based dynamic rebalancing analysis")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *historical {
		showHistoricalSummary()
		return
	}

	var targetDate string
	if *latest {
		targetDate = getLatestPortfolioDate()
	} else if *date != "" {
		targetDate = *date
	} else {
		targetDate = getLatestPortfolioDate()
		*latest = true
	}

	if targetDate == "" {
		log.Fatal("❌ No portfolio data found")
	}

	// Load portfolio data
	portfolio, err := loadPortfolioData(targetDate)
	if err != nil {
		log.Fatalf("❌ Error loading portfolio data: %v", err)
	}

	// Display basic portfolio analysis
	displayPortfolioAnalysis(portfolio, *verbose)

	// Add dynamic rebalancing analysis if requested
	if *mnav {
		fmt.Printf("\n")
		performDynamicRebalancingAnalysis(portfolio, *verbose)
	}
}

func performDynamicRebalancingAnalysis(portfolio *models.Portfolio, verbose bool) {
	if verbose {
		fmt.Printf("🔄 Performing mNAV-based dynamic rebalancing analysis...\n")
	}

	// Current market data - using the same values from our mNAV update
	currentMNAV := 1.54              // From our mNAV analysis
	currentBitcoinPrice := 106108.00 // Current Bitcoin price

	// Calculate total FBTC and MSTR values across all accounts
	var totalFBTCValue, totalMSTRValue float64
	var totalFBTCShares, totalMSTRShares float64
	var fbtcPrice, mstrPrice float64

	for _, pos := range portfolio.Positions {
		if pos.Symbol == "FBTC" {
			totalFBTCValue += pos.CurrentValue
			totalFBTCShares += pos.Quantity
			fbtcPrice = pos.LastPrice // All should be the same price
		} else if pos.Symbol == "MSTR" {
			totalMSTRValue += pos.CurrentValue
			totalMSTRShares += pos.Quantity
			mstrPrice = pos.LastPrice // All should be the same price
		}
	}

	if totalFBTCValue == 0 || totalMSTRValue == 0 {
		fmt.Printf("⚠️  Dynamic rebalancing requires both FBTC and MSTR positions\n")
		return
	}

	// Create dynamic rebalancing table
	rebalanceTable, err := analyzer.NewDynamicRebalancingTable()
	if err != nil {
		fmt.Printf("❌ Error loading rebalancing configuration: %v\n", err)
		return
	}

	if verbose {
		fmt.Printf("📋 Loaded Configuration:\n%s\n", rebalanceTable.GetConfigSummary())
	}

	// Calculate recommendation
	recommendation, err := rebalanceTable.CalculateRebalanceRecommendation(
		currentMNAV,
		totalFBTCValue,
		totalMSTRValue,
		fbtcPrice,
		mstrPrice,
	)
	if err != nil {
		fmt.Printf("❌ Error calculating dynamic rebalancing: %v\n", err)
		return
	}

	// Print holdings info and recommendation
	fmt.Printf("\n💰 Current Holdings:\n")
	fmt.Printf("   FBTC: %.2f shares ($%.2f total)\n", totalFBTCShares, totalFBTCValue)
	fmt.Printf("   MSTR: %.2f shares ($%.2f total)\n", totalMSTRShares, totalMSTRValue)
	fmt.Printf("\n")

	// Print the recommendation (includes its own header)
	recommendation.Print()

	// Add context about mNAV
	fmt.Printf("📊 mNAV Context:\n")
	fmt.Printf("   • Current MSTR mNAV: %.2f\n", currentMNAV)
	fmt.Printf("   • Bitcoin Price: $%.2f\n", currentBitcoinPrice)
	fmt.Printf("   • MSTR Premium: %.1f%% above Bitcoin NAV\n", (currentMNAV-1.0)*100)

	// Calculate Bitcoin exposure through MSTR
	mstrBitcoinHoldings := 597325.0 // From our data update
	mstrSharesOutstanding := 256473000.0
	mstrBitcoinPerShare := mstrBitcoinHoldings / mstrSharesOutstanding
	yourMSTRBitcoinExposure := totalMSTRShares * mstrBitcoinPerShare

	fmt.Printf("   • Your MSTR Bitcoin Exposure: %.4f BTC\n", yourMSTRBitcoinExposure)
	fmt.Printf("   • Strategy: %s\n", getStrategyDescription(currentMNAV))
	fmt.Printf("\n")
}

func getStrategyDescription(mnav float64) string {
	switch {
	case mnav < 1.5:
		return "MSTR trading at very high premium - maximize Bitcoin allocation"
	case mnav < 1.75:
		return "MSTR moderately expensive - balanced Bitcoin/MSTR allocation"
	case mnav < 2.0:
		return "MSTR fairly valued - moderate MSTR allocation"
	case mnav < 2.25:
		return "MSTR getting cheaper - increase MSTR allocation"
	default:
		return "MSTR very cheap - maximize MSTR allocation"
	}
}

func displayPortfolioAnalysis(portfolio *models.Portfolio, verbose bool) {
	fmt.Printf("📊 Portfolio Analysis - %s\n", portfolio.Date.Format("January 2, 2006"))
	fmt.Printf("============================================================\n")
	fmt.Printf("💰 Total Portfolio Value: $%.2f\n", portfolio.TotalValue)
	fmt.Printf("📈 Total Gain/Loss: $%.2f (%.2f%%)\n",
		portfolio.TotalGainLoss, portfolio.TotalGainLossPct)

	// Calculate Bitcoin exposure
	bitcoinExposure := 0.0
	fbtcValue := 0.0
	mstrValue := 0.0

	for _, pos := range portfolio.Positions {
		if pos.Symbol == "FBTC" || pos.Symbol == "MSTR" {
			bitcoinExposure += pos.CurrentValue
			if pos.Symbol == "FBTC" {
				fbtcValue += pos.CurrentValue // Sum all FBTC positions
			} else if pos.Symbol == "MSTR" {
				mstrValue += pos.CurrentValue // Sum all MSTR positions
			}
		}
	}

	bitcoinPercent := (bitcoinExposure / portfolio.TotalValue) * 100
	fmt.Printf("₿  Bitcoin Exposure: $%.2f (%.1f%%)\n", bitcoinExposure, bitcoinPercent)

	if fbtcValue > 0 && mstrValue > 0 {
		ratio := fbtcValue / mstrValue
		fmt.Printf("⚖️  FBTC/MSTR Ratio: %.2f:1\n", ratio)
	}

	// Account breakdown
	fmt.Printf("\n🏦 Account Breakdown:\n")
	for name, account := range portfolio.Accounts {
		percent := (account.TotalValue / portfolio.TotalValue) * 100
		fmt.Printf("   %-25s $%9.2f (%5.1f%%)\n", name, account.TotalValue, percent)
	}

	// Top holdings
	fmt.Printf("\n💎 Asset Allocation:\n")

	// Group by symbol and sum values
	symbolTotals := make(map[string]float64)
	symbolDescriptions := make(map[string]string)

	for _, pos := range portfolio.Positions {
		symbolTotals[pos.Symbol] += pos.CurrentValue
		if _, exists := symbolDescriptions[pos.Symbol]; !exists {
			symbolDescriptions[pos.Symbol] = pos.Description
		}
	}

	// Sort by value
	type symbolValue struct {
		symbol      string
		value       float64
		description string
	}

	var sorted []symbolValue
	for symbol, value := range symbolTotals {
		sorted = append(sorted, symbolValue{
			symbol:      symbol,
			value:       value,
			description: symbolDescriptions[symbol],
		})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	for _, sv := range sorted {
		if sv.value > 100 { // Only show holdings over $100
			percent := (sv.value / portfolio.TotalValue) * 100

			// Get total shares for this symbol
			totalShares := 0.0
			avgPrice := 0.0
			for _, pos := range portfolio.Positions {
				if pos.Symbol == sv.symbol {
					totalShares += pos.Quantity
				}
			}
			if totalShares > 0 {
				avgPrice = sv.value / totalShares
			}

			description := sv.description
			if len(description) > 25 {
				description = description[:25] + "..."
			}

			fmt.Printf("   %-4s %-25s $%9.2f (%5.1f%%) [%.2f shares @ $%.2f]\n",
				sv.symbol, description, sv.value, percent, totalShares, avgPrice)
		}
	}

	if verbose {
		fmt.Printf("\n📋 Detailed Holdings:\n")
		for _, pos := range portfolio.Positions {
			if pos.CurrentValue > 50 { // Show positions over $50
				fmt.Printf("   %s: %.2f shares @ $%.2f = $%.2f\n",
					pos.Symbol, pos.Quantity, pos.LastPrice, pos.CurrentValue)
			}
		}
	}
}

// Helper functions (simplified versions)
func getLatestPortfolioDate() string {
	files, err := filepath.Glob("data/portfolio/processed/portfolio_*.json")
	if err != nil || len(files) == 0 {
		return ""
	}
	sort.Strings(files)
	latest := files[len(files)-1]

	// Extract date from filename
	base := filepath.Base(latest)
	if len(base) >= 19 {
		return base[10:20] // Extract YYYY-MM-DD from portfolio_YYYY-MM-DD.json
	}
	return ""
}

func loadPortfolioData(date string) (*models.Portfolio, error) {
	filename := fmt.Sprintf("data/portfolio/processed/portfolio_%s.json", date)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var portfolio models.Portfolio
	if err := json.Unmarshal(data, &portfolio); err != nil {
		return nil, err
	}

	return &portfolio, nil
}

func showHistoricalSummary() {
	files, err := filepath.Glob("data/portfolio/processed/portfolio_*.json")
	if err != nil || len(files) == 0 {
		fmt.Printf("❌ No portfolio data found\n")
		return
	}

	sort.Strings(files)

	fmt.Printf("📈 Historical Portfolio Summary (%d snapshots)\n", len(files))
	fmt.Printf("================================================================================\n")

	var previousValue float64
	for i, file := range files {
		base := filepath.Base(file)
		if len(base) < 19 {
			continue
		}

		date := base[10:20]
		portfolio, err := loadPortfolioData(date)
		if err != nil {
			continue
		}

		// Calculate Bitcoin exposure
		bitcoinExposure := 0.0
		fbtcValue := 0.0
		mstrValue := 0.0

		for _, pos := range portfolio.Positions {
			if pos.Symbol == "FBTC" || pos.Symbol == "MSTR" {
				bitcoinExposure += pos.CurrentValue
				if pos.Symbol == "FBTC" {
					fbtcValue = pos.CurrentValue
				} else if pos.Symbol == "MSTR" {
					mstrValue = pos.CurrentValue
				}
			}
		}

		bitcoinPercent := (bitcoinExposure / portfolio.TotalValue) * 100
		ratio := 0.0
		if mstrValue > 0 {
			ratio = fbtcValue / mstrValue
		}

		changeText := ""
		if i > 0 && previousValue > 0 {
			change := portfolio.TotalValue - previousValue
			changePercent := (change / previousValue) * 100
			if change >= 0 {
				changeText = fmt.Sprintf(" | Δ $+%.2f (+%.2f%%)", change, changePercent)
			} else {
				changeText = fmt.Sprintf(" | Δ $%.2f (%.2f%%)", change, changePercent)
			}
		}

		fmt.Printf("%s | $%9.2f | ₿ %4.1f%% | Ratio: %5.2f:1%s\n",
			date, portfolio.TotalValue, bitcoinPercent, ratio, changeText)

		previousValue = portfolio.TotalValue
	}

	if len(files) >= 2 {
		firstPortfolio, _ := loadPortfolioData(files[0][10:20])
		lastPortfolio, _ := loadPortfolioData(files[len(files)-1][10:20])

		if firstPortfolio != nil && lastPortfolio != nil {
			totalReturn := lastPortfolio.TotalValue - firstPortfolio.TotalValue
			totalReturnPercent := (totalReturn / firstPortfolio.TotalValue) * 100

			fmt.Printf("\n📊 Overall Performance:\n")
			fmt.Printf("   Period: %s to %s\n",
				firstPortfolio.Date.Format("2006-01-02"),
				lastPortfolio.Date.Format("2006-01-02"))
			fmt.Printf("   Total Return: $%.2f (%.2f%%)\n", totalReturn, totalReturnPercent)
			fmt.Printf("   Starting Value: $%.2f\n", firstPortfolio.TotalValue)
			fmt.Printf("   Ending Value: $%.2f\n", lastPortfolio.TotalValue)
		}
	}
}
