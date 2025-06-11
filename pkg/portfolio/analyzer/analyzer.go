package analyzer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/portfolio/models"
)

// Analyzer handles portfolio analysis operations
type Analyzer struct {
	// Could add configuration here if needed
}

// NewAnalyzer creates a new portfolio analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// ParseCSV parses a Fidelity portfolio CSV file into a Portfolio struct
func (a *Analyzer) ParseCSV(filePath string) (*models.Portfolio, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read headers: %w", err)
	}

	// Create header map for flexible parsing
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[header] = i
	}

	var positions []models.Position
	var rawRecords [][]string

	// Read all records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}
		rawRecords = append(rawRecords, record)
	}

	// Parse positions, skipping disclaimer rows
	for _, record := range rawRecords {
		// Skip disclaimer rows (they don't have proper data structure)
		if len(record) < len(headers) ||
			strings.Contains(record[0], "The data and information") ||
			strings.Contains(record[0], "Brokerage services") ||
			strings.Contains(record[0], "Date downloaded") ||
			len(record) == 1 { // Single column rows are usually disclaimers
			continue
		}

		position, err := a.parsePosition(record, headerMap)
		if err != nil {
			continue // Skip invalid records
		}

		// Skip cash positions (SPAXX)
		if strings.Contains(position.Symbol, "SPAXX") {
			continue
		}

		// Only include positions with actual value
		if position.CurrentValue > 0 {
			positions = append(positions, *position)
		}
	}

	// Extract date from filename or use current date
	date := time.Now()
	if strings.Contains(filePath, "Jun-11-2025") {
		date, _ = time.Parse("Jan-2-2006", "Jun-11-2025")
	}

	// Create portfolio
	portfolio := &models.Portfolio{
		Date:       date,
		SourceFile: filePath,
		Positions:  positions,
		Accounts:   make(map[string]*models.Account),
		CreatedAt:  time.Now(),
	}

	// Calculate aggregations
	a.calculateAggregations(portfolio)

	return portfolio, nil
}

// parsePosition converts a CSV record to a Position struct
func (a *Analyzer) parsePosition(record []string, headerMap map[string]int) (*models.Position, error) {
	position := &models.Position{}

	// Helper function to safely get string values
	getString := func(header string) string {
		if idx, exists := headerMap[header]; exists && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	// Helper function to safely parse float values
	getFloat := func(header string) float64 {
		value := getString(header)
		if value == "" {
			return 0
		}
		// Remove currency symbols and commas
		value = strings.ReplaceAll(value, "$", "")
		value = strings.ReplaceAll(value, ",", "")
		value = strings.ReplaceAll(value, "+", "")

		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
		return 0
	}

	position.AccountNumber = getString("Account Number")
	position.AccountName = getString("Account Name")
	position.Symbol = getString("Symbol")
	position.Description = getString("Description")
	position.Quantity = getFloat("Quantity")
	position.LastPrice = getFloat("Last Price")
	position.LastPriceChange = getFloat("Last Price Change")
	position.CurrentValue = getFloat("Current Value")
	position.TodayGainLoss = getFloat("Today's Gain/Loss Dollar")
	position.TodayGainLossPct = getFloat("Today's Gain/Loss Percent")
	position.TotalGainLoss = getFloat("Total Gain/Loss Dollar")
	position.TotalGainLossPct = getFloat("Total Gain/Loss Percent")
	position.PercentOfAccount = getFloat("Percent Of Account")
	position.CostBasisTotal = getFloat("Cost Basis Total")
	position.AverageCostBasis = getFloat("Average Cost Basis")
	position.Type = getString("Type")

	return position, nil
}

// calculateAggregations calculates portfolio-level aggregations
func (a *Analyzer) calculateAggregations(portfolio *models.Portfolio) {
	accounts := make(map[string]*models.Account)

	// Aggregate by account
	for _, position := range portfolio.Positions {
		if _, exists := accounts[position.AccountName]; !exists {
			accounts[position.AccountName] = &models.Account{
				AccountNumber: position.AccountNumber,
				AccountName:   position.AccountName,
				Positions:     []models.Position{},
			}
		}

		account := accounts[position.AccountName]
		account.Positions = append(account.Positions, position)
		account.TotalValue += position.CurrentValue
		account.TotalCostBasis += position.CostBasisTotal
		account.TotalGainLoss += position.TotalGainLoss
	}

	// Calculate account-level percentages
	for _, account := range accounts {
		if account.TotalCostBasis > 0 {
			account.TotalGainLossPct = (account.TotalGainLoss / account.TotalCostBasis) * 100
		}
	}

	portfolio.Accounts = accounts

	// Calculate portfolio totals
	for _, position := range portfolio.Positions {
		portfolio.TotalValue += position.CurrentValue
		portfolio.TotalCostBasis += position.CostBasisTotal
		portfolio.TotalGainLoss += position.TotalGainLoss
	}

	if portfolio.TotalCostBasis > 0 {
		portfolio.TotalGainLossPct = (portfolio.TotalGainLoss / portfolio.TotalCostBasis) * 100
	}

	// Calculate asset allocation
	portfolio.AssetAllocation = a.calculateAssetAllocation(portfolio)
}

// calculateAssetAllocation calculates the asset allocation breakdown
func (a *Analyzer) calculateAssetAllocation(portfolio *models.Portfolio) models.AssetAllocation {
	allocation := models.AssetAllocation{}

	symbolTotals := make(map[string]float64)
	for _, position := range portfolio.Positions {
		symbolTotals[position.Symbol] += position.CurrentValue
	}

	allocation.FBTCValue = symbolTotals["FBTC"]
	allocation.MSTRValue = symbolTotals["MSTR"]
	allocation.GLDValue = symbolTotals["GLD"]

	// Calculate other assets
	for symbol, value := range symbolTotals {
		if symbol != "FBTC" && symbol != "MSTR" && symbol != "GLD" {
			allocation.OtherValue += value
		}
	}

	// Calculate percentages
	if portfolio.TotalValue > 0 {
		allocation.FBTCPercent = (allocation.FBTCValue / portfolio.TotalValue) * 100
		allocation.MSTRPercent = (allocation.MSTRValue / portfolio.TotalValue) * 100
		allocation.GLDPercent = (allocation.GLDValue / portfolio.TotalValue) * 100
		allocation.OtherPercent = (allocation.OtherValue / portfolio.TotalValue) * 100
	}

	// Calculate Bitcoin exposure
	allocation.BitcoinExposure = allocation.FBTCValue + allocation.MSTRValue
	if portfolio.TotalValue > 0 {
		allocation.BitcoinPercent = (allocation.BitcoinExposure / portfolio.TotalValue) * 100
	}

	// Calculate FBTC/MSTR ratio
	if allocation.MSTRValue > 0 {
		allocation.FBTCMSTRRatio = allocation.FBTCValue / allocation.MSTRValue
	}

	return allocation
}

// CalculateRebalance calculates how to rebalance to achieve a target FBTC:MSTR ratio
func (a *Analyzer) CalculateRebalance(portfolio *models.Portfolio, targetRatio float64) *models.RebalanceRecommendation {
	allocation := portfolio.AssetAllocation

	if allocation.MSTRValue == 0 {
		return &models.RebalanceRecommendation{
			CurrentRatio:    0,
			TargetRatio:     targetRatio,
			ReasonableRange: false,
		}
	}

	currentRatio := allocation.FBTCMSTRRatio

	// Calculate trade amount needed
	// X = amount to sell from FBTC and buy in MSTR
	// (FBTC - X) / (MSTR + X) = targetRatio
	tradeAmount := (allocation.FBTCValue - targetRatio*allocation.MSTRValue) / (1 + targetRatio)

	if tradeAmount <= 0 {
		return &models.RebalanceRecommendation{
			CurrentRatio:    currentRatio,
			TargetRatio:     targetRatio,
			ReasonableRange: false,
		}
	}

	// Get current prices for share calculations
	var fbtcPrice, mstrPrice float64
	for _, position := range portfolio.Positions {
		if position.Symbol == "FBTC" && fbtcPrice == 0 {
			fbtcPrice = position.LastPrice
		}
		if position.Symbol == "MSTR" && mstrPrice == 0 {
			mstrPrice = position.LastPrice
		}
	}

	var trades []models.RecommendedTrade
	if fbtcPrice > 0 && mstrPrice > 0 {
		trades = append(trades,
			models.RecommendedTrade{
				Action:         "SELL",
				Symbol:         "FBTC",
				Shares:         tradeAmount / fbtcPrice,
				EstimatedValue: tradeAmount,
			},
			models.RecommendedTrade{
				Action:         "BUY",
				Symbol:         "MSTR",
				Shares:         tradeAmount / mstrPrice,
				EstimatedValue: tradeAmount,
			},
		)
	}

	// Calculate new allocation after rebalancing
	newAllocation := allocation
	newAllocation.FBTCValue -= tradeAmount
	newAllocation.MSTRValue += tradeAmount
	newAllocation.FBTCPercent = (newAllocation.FBTCValue / portfolio.TotalValue) * 100
	newAllocation.MSTRPercent = (newAllocation.MSTRValue / portfolio.TotalValue) * 100
	newAllocation.FBTCMSTRRatio = newAllocation.FBTCValue / newAllocation.MSTRValue

	return &models.RebalanceRecommendation{
		CurrentRatio:    currentRatio,
		TargetRatio:     targetRatio,
		TradeAmount:     tradeAmount,
		FBTCToSell:      tradeAmount / fbtcPrice,
		MSTRToBuy:       tradeAmount / mstrPrice,
		NewAllocation:   newAllocation,
		ReasonableRange: tradeAmount/portfolio.TotalValue <= 0.10, // Less than 10% of portfolio
		Trades:          trades,
	}
}

// GetSymbolSummary returns aggregated data for all symbols
func (a *Analyzer) GetSymbolSummary(portfolio *models.Portfolio) []models.SymbolSummary {
	symbolData := make(map[string]*models.SymbolSummary)

	for _, position := range portfolio.Positions {
		if summary, exists := symbolData[position.Symbol]; exists {
			summary.TotalQuantity += position.Quantity
			summary.TotalValue += position.CurrentValue
			summary.TotalCostBasis += position.CostBasisTotal
			summary.TotalGainLoss += position.TotalGainLoss
		} else {
			symbolData[position.Symbol] = &models.SymbolSummary{
				Symbol:         position.Symbol,
				Description:    position.Description,
				TotalQuantity:  position.Quantity,
				TotalValue:     position.CurrentValue,
				TotalCostBasis: position.CostBasisTotal,
				TotalGainLoss:  position.TotalGainLoss,
				LastPrice:      position.LastPrice,
			}
		}
	}

	// Calculate percentages and convert to slice
	var summaries []models.SymbolSummary
	for _, summary := range symbolData {
		if summary.TotalCostBasis > 0 {
			summary.TotalGainLossPct = (summary.TotalGainLoss / summary.TotalCostBasis) * 100
		}
		summary.PercentOfTotal = (summary.TotalValue / portfolio.TotalValue) * 100
		summaries = append(summaries, *summary)
	}

	return summaries
}
