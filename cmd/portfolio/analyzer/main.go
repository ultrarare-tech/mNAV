package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/portfolio/analyzer"
	"github.com/jeffreykibler/mNAV/pkg/portfolio/models"
	"github.com/jeffreykibler/mNAV/pkg/portfolio/tracker"
)

func main() {
	var (
		dataDir     = flag.String("data", "data/portfolio/processed", "Directory containing processed portfolio data")
		latest      = flag.Bool("latest", false, "Analyze latest portfolio")
		date        = flag.String("date", "", "Analyze portfolio for specific date (YYYY-MM-DD)")
		rebalance   = flag.String("rebalance", "", "Calculate rebalancing for target FBTC:MSTR ratio (e.g., '5.0')")
		historical  = flag.Bool("historical", false, "Show historical summary")
		performance = flag.Bool("performance", false, "Show performance metrics")
		verbose     = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	tracker := tracker.NewTracker(*dataDir)
	analyzer := analyzer.NewAnalyzer()

	if *historical {
		showHistoricalSummary(tracker)
		return
	}

	if *performance {
		showPerformanceMetrics(tracker)
		return
	}

	var portfolio *models.Portfolio
	var err error

	if *latest {
		portfolio, err = tracker.GetLatest()
		if err != nil {
			log.Fatalf("Failed to get latest portfolio: %v", err)
		}
	} else if *date != "" {
		targetDate, err := time.Parse("2006-01-02", *date)
		if err != nil {
			log.Fatalf("Invalid date format. Use YYYY-MM-DD: %v", err)
		}
		portfolio, err = tracker.Load(targetDate)
		if err != nil {
			log.Fatalf("Failed to load portfolio for date %s: %v", *date, err)
		}
	} else {
		// Show available dates
		dates, err := tracker.ListAll()
		if err != nil {
			log.Fatalf("Failed to list portfolio dates: %v", err)
		}

		if len(dates) == 0 {
			fmt.Println("No portfolio data found. Import data using the portfolio importer first.")
			return
		}

		fmt.Println("Available portfolio dates:")
		for _, d := range dates {
			fmt.Printf("  %s\n", d.Format("2006-01-02"))
		}
		fmt.Printf("\nUse -latest or -date YYYY-MM-DD to analyze a specific portfolio.\n")
		return
	}

	// Display portfolio analysis
	displayPortfolioAnalysis(portfolio, analyzer, *verbose)

	// Handle rebalancing calculation
	if *rebalance != "" {
		targetRatio, err := strconv.ParseFloat(*rebalance, 64)
		if err != nil {
			log.Fatalf("Invalid rebalance ratio: %v", err)
		}

		fmt.Printf("\n🔄 Rebalancing Analysis (Target FBTC:MSTR = %.1f:1)\n", targetRatio)
		fmt.Println(strings.Repeat("=", 60))

		recommendation := analyzer.CalculateRebalance(portfolio, targetRatio)
		displayRebalanceRecommendation(recommendation)
	}
}

func displayPortfolioAnalysis(portfolio *models.Portfolio, analyzer *analyzer.Analyzer, verbose bool) {
	fmt.Printf("📊 Portfolio Analysis - %s\n", portfolio.Date.Format("January 2, 2006"))
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("💰 Total Portfolio Value: $%.2f\n", portfolio.TotalValue)
	fmt.Printf("📈 Total Gain/Loss: $%.2f (%.2f%%)\n", portfolio.TotalGainLoss, portfolio.TotalGainLossPct)
	fmt.Printf("₿  Bitcoin Exposure: $%.2f (%.1f%%)\n", portfolio.AssetAllocation.BitcoinExposure, portfolio.AssetAllocation.BitcoinPercent)
	fmt.Printf("⚖️  FBTC/MSTR Ratio: %.2f:1\n", portfolio.AssetAllocation.FBTCMSTRRatio)

	fmt.Printf("\n🏦 Account Breakdown:\n")
	// Sort accounts by value
	type accountValue struct {
		name  string
		value float64
	}
	var accounts []accountValue
	for name, account := range portfolio.Accounts {
		accounts = append(accounts, accountValue{name, account.TotalValue})
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].value > accounts[j].value
	})

	for _, account := range accounts {
		percentage := (account.value / portfolio.TotalValue) * 100
		fmt.Printf("   %-25s $%10.2f (%5.1f%%)\n", account.name, account.value, percentage)
	}

	fmt.Printf("\n💎 Asset Allocation:\n")
	allocations := []struct {
		name    string
		value   float64
		percent float64
	}{
		{"FBTC (Bitcoin ETF)", portfolio.AssetAllocation.FBTCValue, portfolio.AssetAllocation.FBTCPercent},
		{"MSTR (MicroStrategy)", portfolio.AssetAllocation.MSTRValue, portfolio.AssetAllocation.MSTRPercent},
		{"GLD (Gold ETF)", portfolio.AssetAllocation.GLDValue, portfolio.AssetAllocation.GLDPercent},
		{"Other Assets", portfolio.AssetAllocation.OtherValue, portfolio.AssetAllocation.OtherPercent},
	}

	for _, alloc := range allocations {
		if alloc.value > 0 {
			fmt.Printf("   %-20s $%10.2f (%5.1f%%)\n", alloc.name, alloc.value, alloc.percent)
		}
	}

	fmt.Printf("\n📋 Top Holdings:\n")
	symbolSummaries := analyzer.GetSymbolSummary(portfolio)
	sort.Slice(symbolSummaries, func(i, j int) bool {
		return symbolSummaries[i].TotalValue > symbolSummaries[j].TotalValue
	})

	for i, summary := range symbolSummaries {
		if i >= 5 { // Show top 5
			break
		}
		fmt.Printf("   %-6s %-25s $%10.2f (%5.1f%%) [%.2f shares @ $%.2f]\n",
			summary.Symbol,
			truncateString(summary.Description, 25),
			summary.TotalValue,
			summary.PercentOfTotal,
			summary.TotalQuantity,
			summary.LastPrice)
	}

	if verbose {
		fmt.Printf("\n📄 All Positions:\n")
		for _, position := range portfolio.Positions {
			fmt.Printf("   %-6s %-20s %-25s $%8.2f [%.2f @ $%.2f]\n",
				position.Symbol,
				position.AccountName,
				truncateString(position.Description, 25),
				position.CurrentValue,
				position.Quantity,
				position.LastPrice)
		}
	}
}

func displayRebalanceRecommendation(rec *models.RebalanceRecommendation) {
	fmt.Printf("Current FBTC:MSTR Ratio: %.2f:1\n", rec.CurrentRatio)
	fmt.Printf("Target FBTC:MSTR Ratio:  %.2f:1\n", rec.TargetRatio)

	if !rec.ReasonableRange {
		fmt.Printf("⚠️  Warning: Rebalancing would require large trades (>10%% of portfolio)\n")
	}

	if len(rec.Trades) > 0 {
		fmt.Printf("\n💱 Recommended Trades:\n")
		for _, trade := range rec.Trades {
			fmt.Printf("   %s %.2f shares of %s (~$%.2f)\n",
				trade.Action, trade.Shares, trade.Symbol, trade.EstimatedValue)
		}

		fmt.Printf("\n🎯 After Rebalancing:\n")
		fmt.Printf("   FBTC: $%.2f (%.1f%%)\n", rec.NewAllocation.FBTCValue, rec.NewAllocation.FBTCPercent)
		fmt.Printf("   MSTR: $%.2f (%.1f%%)\n", rec.NewAllocation.MSTRValue, rec.NewAllocation.MSTRPercent)
		fmt.Printf("   New Ratio: %.2f:1\n", rec.NewAllocation.FBTCMSTRRatio)
	}
}

func showHistoricalSummary(tracker *tracker.Tracker) {
	history, err := tracker.GetHistoricalSummary()
	if err != nil {
		log.Fatalf("Failed to get historical summary: %v", err)
	}

	if len(history) == 0 {
		fmt.Println("No historical data available.")
		return
	}

	fmt.Printf("📈 Historical Portfolio Summary (%d snapshots)\n", len(history))
	fmt.Println(strings.Repeat("=", 80))

	for _, snapshot := range history {
		fmt.Printf("%s | $%10.2f | ₿ %5.1f%% | Ratio: %5.2f:1",
			snapshot.Date.Format("2006-01-02"),
			snapshot.TotalValue,
			snapshot.AssetAllocation.BitcoinPercent,
			snapshot.AssetAllocation.FBTCMSTRRatio)

		if snapshot.Changes != nil {
			fmt.Printf(" | Δ $%+8.2f (%+5.2f%%)",
				snapshot.Changes.ValueChange,
				snapshot.Changes.ValueChangePercent)
		}
		fmt.Println()
	}

	// Show overall summary
	first := history[0]
	last := history[len(history)-1]
	totalReturn := last.TotalValue - first.TotalValue
	totalReturnPct := (totalReturn / first.TotalValue) * 100

	fmt.Printf("\n📊 Overall Performance:\n")
	fmt.Printf("   Period: %s to %s\n", first.Date.Format("2006-01-02"), last.Date.Format("2006-01-02"))
	fmt.Printf("   Total Return: $%.2f (%.2f%%)\n", totalReturn, totalReturnPct)
	fmt.Printf("   Starting Value: $%.2f\n", first.TotalValue)
	fmt.Printf("   Ending Value: $%.2f\n", last.TotalValue)
}

func showPerformanceMetrics(tracker *tracker.Tracker) {
	metrics, err := tracker.GetPerformanceMetrics()
	if err != nil {
		log.Fatalf("Failed to get performance metrics: %v", err)
	}

	fmt.Printf("📊 Portfolio Performance Metrics\n")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Period: %s to %s\n", metrics.StartDate.Format("2006-01-02"), metrics.EndDate.Format("2006-01-02"))
	fmt.Printf("Starting Value: $%.2f\n", metrics.StartValue)
	fmt.Printf("Ending Value: $%.2f\n", metrics.EndValue)
	fmt.Printf("Total Return: $%.2f (%.2f%%)\n", metrics.TotalReturn, metrics.TotalReturnPercent)
	fmt.Printf("CAGR: %.2f%%\n", metrics.CAGR)
	fmt.Printf("Volatility: %.2f%%\n", metrics.Volatility)
	fmt.Printf("Max Drawdown: %.2f%%\n", metrics.MaxDrawdown)
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
