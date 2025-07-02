package tracker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/portfolio/models"
)

// Tracker handles historical portfolio data management
type Tracker struct {
	dataDir string
}

// NewTracker creates a new portfolio tracker
func NewTracker(dataDir string) *Tracker {
	return &Tracker{
		dataDir: dataDir,
	}
}

// Store saves a portfolio snapshot to persistent storage
func (t *Tracker) Store(portfolio *models.Portfolio) error {
	// Ensure directory exists
	if err := os.MkdirAll(t.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Generate filename with date
	filename := fmt.Sprintf("portfolio_%s.json", portfolio.Date.Format("2006-01-02"))
	filepath := filepath.Join(t.dataDir, filename)

	// Convert to JSON
	data, err := json.MarshalIndent(portfolio, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal portfolio: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write portfolio file: %w", err)
	}

	return nil
}

// Load retrieves a portfolio snapshot by date
func (t *Tracker) Load(date time.Time) (*models.Portfolio, error) {
	filename := fmt.Sprintf("portfolio_%s.json", date.Format("2006-01-02"))
	filepath := filepath.Join(t.dataDir, filename)

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read portfolio file: %w", err)
	}

	var portfolio models.Portfolio
	if err := json.Unmarshal(data, &portfolio); err != nil {
		return nil, fmt.Errorf("failed to unmarshal portfolio: %w", err)
	}

	return &portfolio, nil
}

// ListAll returns all stored portfolio dates
func (t *Tracker) ListAll() ([]time.Time, error) {
	files, err := ioutil.ReadDir(t.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var dates []time.Time
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			dateStr := file.Name()
			dateStr = dateStr[10 : len(dateStr)-5] // Remove "portfolio_" prefix and ".json" suffix

			if date, err := time.Parse("2006-01-02", dateStr); err == nil {
				dates = append(dates, date)
			}
		}
	}

	// Sort dates
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	return dates, nil
}

// GetLatest returns the most recent portfolio snapshot
func (t *Tracker) GetLatest() (*models.Portfolio, error) {
	dates, err := t.ListAll()
	if err != nil {
		return nil, err
	}

	if len(dates) == 0 {
		return nil, fmt.Errorf("no portfolio data found")
	}

	return t.Load(dates[len(dates)-1])
}

// GetHistoricalSummary returns a summary of portfolio changes over time
func (t *Tracker) GetHistoricalSummary() ([]models.HistoricalTracking, error) {
	dates, err := t.ListAll()
	if err != nil {
		return nil, err
	}

	var history []models.HistoricalTracking
	var previousPortfolio *models.Portfolio

	for _, date := range dates {
		portfolio, err := t.Load(date)
		if err != nil {
			continue // Skip corrupted files
		}

		// Create summary for this date
		summary := models.HistoricalTracking{
			Date:             portfolio.Date,
			TotalValue:       portfolio.TotalValue,
			TotalGainLoss:    portfolio.TotalGainLoss,
			AssetAllocation:  portfolio.AssetAllocation,
			AccountBreakdown: make(map[string]float64),
		}

		// Account breakdown
		for name, account := range portfolio.Accounts {
			summary.AccountBreakdown[name] = account.TotalValue
		}

		// Top holdings (top 5 by value)
		symbolSummaries := t.getSymbolSummaries(portfolio)
		sort.Slice(symbolSummaries, func(i, j int) bool {
			return symbolSummaries[i].TotalValue > symbolSummaries[j].TotalValue
		})

		maxHoldings := 5
		if len(symbolSummaries) < maxHoldings {
			maxHoldings = len(symbolSummaries)
		}
		summary.TopHoldings = symbolSummaries[:maxHoldings]

		// Calculate changes from previous period
		if previousPortfolio != nil {
			summary.Changes = t.calculateChanges(previousPortfolio, portfolio)
		}

		history = append(history, summary)
		previousPortfolio = portfolio
	}

	return history, nil
}

// getSymbolSummaries creates symbol summaries for a portfolio
func (t *Tracker) getSymbolSummaries(portfolio *models.Portfolio) []models.SymbolSummary {
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

// calculateChanges computes changes between two portfolio snapshots
func (t *Tracker) calculateChanges(previous, current *models.Portfolio) *models.PortfolioChanges {
	changes := &models.PortfolioChanges{
		AllocationChanges: make(map[string]float64),
	}

	// Value changes
	changes.ValueChange = current.TotalValue - previous.TotalValue
	if previous.TotalValue > 0 {
		changes.ValueChangePercent = (changes.ValueChange / previous.TotalValue) * 100
	}

	// Position changes
	previousSymbols := make(map[string]bool)
	currentSymbols := make(map[string]bool)

	for _, pos := range previous.Positions {
		previousSymbols[pos.Symbol] = true
	}

	for _, pos := range current.Positions {
		currentSymbols[pos.Symbol] = true
		if !previousSymbols[pos.Symbol] {
			changes.NewPositions = append(changes.NewPositions, pos.Symbol)
		}
	}

	for symbol := range previousSymbols {
		if !currentSymbols[symbol] {
			changes.ClosedPositions = append(changes.ClosedPositions, symbol)
		}
	}

	// Allocation changes
	changes.AllocationChanges["FBTC"] = current.AssetAllocation.FBTCPercent - previous.AssetAllocation.FBTCPercent
	changes.AllocationChanges["MSTR"] = current.AssetAllocation.MSTRPercent - previous.AssetAllocation.MSTRPercent
	changes.AllocationChanges["GLD"] = current.AssetAllocation.GLDPercent - previous.AssetAllocation.GLDPercent
	changes.AllocationChanges["Other"] = current.AssetAllocation.OtherPercent - previous.AssetAllocation.OtherPercent

	return changes
}

// GetPerformanceMetrics calculates performance metrics over time
func (t *Tracker) GetPerformanceMetrics() (*PerformanceMetrics, error) {
	history, err := t.GetHistoricalSummary()
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no historical data available")
	}

	metrics := &PerformanceMetrics{
		StartDate:  history[0].Date,
		EndDate:    history[len(history)-1].Date,
		StartValue: history[0].TotalValue,
		EndValue:   history[len(history)-1].TotalValue,
	}

	// Calculate total return
	metrics.TotalReturn = metrics.EndValue - metrics.StartValue
	if metrics.StartValue > 0 {
		metrics.TotalReturnPercent = (metrics.TotalReturn / metrics.StartValue) * 100
	}

	// Calculate CAGR (Compound Annual Growth Rate)
	days := metrics.EndDate.Sub(metrics.StartDate).Hours() / 24
	years := days / 365.25
	if years > 0 && metrics.StartValue > 0 {
		metrics.CAGR = (math.Pow(metrics.EndValue/metrics.StartValue, 1/years) - 1) * 100
	}

	// Track volatility and max drawdown
	var returns []float64
	var maxValue float64 = metrics.StartValue
	var maxDrawdown float64

	for i, snapshot := range history {
		if i > 0 {
			previousValue := history[i-1].TotalValue
			if previousValue > 0 {
				dailyReturn := (snapshot.TotalValue - previousValue) / previousValue
				returns = append(returns, dailyReturn)
			}
		}

		// Track max value and drawdown
		if snapshot.TotalValue > maxValue {
			maxValue = snapshot.TotalValue
		}
		drawdown := (maxValue - snapshot.TotalValue) / maxValue
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	metrics.MaxDrawdown = maxDrawdown * 100

	// Calculate volatility (standard deviation of returns)
	if len(returns) > 1 {
		var sum, mean float64
		for _, ret := range returns {
			sum += ret
		}
		mean = sum / float64(len(returns))

		var variance float64
		for _, ret := range returns {
			variance += math.Pow(ret-mean, 2)
		}
		variance /= float64(len(returns) - 1)
		metrics.Volatility = math.Sqrt(variance) * math.Sqrt(365) * 100 // Annualized
	}

	return metrics, nil
}

// PerformanceMetrics represents portfolio performance over time
type PerformanceMetrics struct {
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	StartValue         float64   `json:"start_value"`
	EndValue           float64   `json:"end_value"`
	TotalReturn        float64   `json:"total_return"`
	TotalReturnPercent float64   `json:"total_return_percent"`
	CAGR               float64   `json:"cagr"`
	Volatility         float64   `json:"volatility"`
	MaxDrawdown        float64   `json:"max_drawdown"`
}
