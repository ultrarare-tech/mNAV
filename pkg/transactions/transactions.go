package transactions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Transaction represents a single Bitcoin transaction
type Transaction struct {
	Date            string  `json:"date"`
	Type            string  `json:"type"`
	Quantity        float64 `json:"quantity"`
	TotalCostUSD    float64 `json:"total_cost_usd,omitempty"`
	ProceedsUSD     float64 `json:"proceeds_usd,omitempty"`
	AveragePriceUSD float64 `json:"average_price_usd"`
	CumulativeBTC   float64 `json:"cumulative_btc,omitempty"`
}

// Summary represents a summary of all transactions
type Summary struct {
	TotalBTC        float64 `json:"total_btc"`
	TotalCostUSD    float64 `json:"total_cost_usd,omitempty"`
	AveragePriceUSD float64 `json:"average_price_usd,omitempty"`
	AsOfDate        string  `json:"as_of_date"`
	Note            string  `json:"note,omitempty"`
}

// CompanyTransactions represents the transactions for a single company
type CompanyTransactions struct {
	Transactions []Transaction `json:"transactions"`
	Summary      Summary       `json:"summary"`
}

// CombinedTransactions represents the combined transactions for all companies
type CombinedTransactions struct {
	Companies map[string]CompanyTransactions `json:"companies"`
}

// LoadTransactions loads the transactions from the combined transactions file
func LoadTransactions(basePath string) (*CombinedTransactions, error) {
	// Determine the path to the JSON file
	jsonPath := filepath.Join(basePath, "data", "transactions", "combined_bitcoin_transactions.json")

	// Read the file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transactions file: %w", err)
	}

	// Parse the JSON
	var combined CombinedTransactions
	if err := json.Unmarshal(data, &combined); err != nil {
		return nil, fmt.Errorf("failed to parse transactions JSON: %w", err)
	}

	return &combined, nil
}

// GetTransactionsForCompany returns the transactions for a specific company
func (c *CombinedTransactions) GetTransactionsForCompany(symbol string) (CompanyTransactions, bool) {
	company, ok := c.Companies[symbol]
	return company, ok
}

// CalculateYieldFromTransactions calculates the Bitcoin yield from the most recent transactions
// by looking at the recent purchase patterns over the last 90 days
func CalculateYieldFromTransactions(transactions []Transaction, totalHoldings float64) (float64, error) {
	if len(transactions) < 1 {
		return 0, fmt.Errorf("not enough transactions to calculate yield")
	}

	if totalHoldings <= 0 {
		return 0, fmt.Errorf("total holdings must be greater than zero")
	}

	// Use transactions from the last 90 days
	recentTransactions := getRecentTransactions(transactions, 90)

	// If no transactions in last 90 days, use all available transactions
	if len(recentTransactions) == 0 {
		recentTransactions = transactions
	}

	// Calculate total BTC acquired in the period
	var totalAcquired float64
	var validTransactionCount int
	for _, t := range recentTransactions {
		if t.Type == "purchase" {
			totalAcquired += t.Quantity
			validTransactionCount++
		} else if t.Type == "sale" {
			totalAcquired += t.Quantity // Note: sale quantities should be negative
			validTransactionCount++
		}
	}

	// If no valid transactions found, return minimum yield
	if validTransactionCount == 0 {
		return 0.0001, nil // 0.01% daily minimum
	}

	// Calculate the time period for the transactions
	daysDiff := 90.0 // Default to 90 days

	// Try to calculate actual days based on transaction dates
	if len(recentTransactions) >= 2 {
		firstDate, err1 := parseTransactionDate(recentTransactions[len(recentTransactions)-1].Date)
		lastDate, err2 := parseTransactionDate(recentTransactions[0].Date)

		if err1 == nil && err2 == nil && lastDate.After(firstDate) {
			calculatedDays := lastDate.Sub(firstDate).Hours() / 24
			if calculatedDays > 0 && calculatedDays <= 365 { // Reasonable range
				daysDiff = calculatedDays
			}
		}
	} else if len(recentTransactions) == 1 {
		// Single transaction - assume it represents recent activity pattern
		// Use a shorter period to avoid diluting the yield calculation
		daysDiff = 30.0
	}

	// Calculate the acquisition rate as percentage of total holdings
	acquisitionRate := totalAcquired / totalHoldings

	// Annualize the rate: (acquisition rate) * (365 / days in period)
	annualizedRate := acquisitionRate * (365 / daysDiff)

	// Convert to daily yield
	dailyYield := annualizedRate / 365

	// Apply reasonable bounds
	if dailyYield > 0.01 { // Cap at 1% daily (very aggressive)
		dailyYield = 0.01
	} else if dailyYield < 0.0001 { // Minimum of 0.01% daily
		dailyYield = 0.0001
	}

	// Handle negative yields (net selling) by setting to minimum
	if dailyYield < 0 {
		dailyYield = 0.0001
	}

	return dailyYield, nil
}

// getRecentTransactions returns transactions from the last N days
func getRecentTransactions(transactions []Transaction, days int) []Transaction {
	if len(transactions) == 0 {
		return nil
	}

	// Parse the most recent transaction date
	mostRecentDate, err := parseTransactionDate(transactions[0].Date)
	if err != nil {
		// If we can't parse the date, return all transactions
		return transactions
	}

	// Calculate the cutoff date
	cutoffDate := mostRecentDate.AddDate(0, 0, -days)

	// Filter transactions
	var recentTransactions []Transaction
	for _, t := range transactions {
		tDate, err := parseTransactionDate(t.Date)
		if err != nil {
			continue
		}

		if tDate.After(cutoffDate) || tDate.Equal(cutoffDate) {
			recentTransactions = append(recentTransactions, t)
		}
	}

	return recentTransactions
}

// parseTransactionDate parses a transaction date string
func parseTransactionDate(dateStr string) (time.Time, error) {
	// Handle date ranges like "2024-11-11 to 2024-11-18" by using the later date
	if len(dateStr) > 10 && dateStr[10:14] == " to " {
		dateStr = dateStr[15:] // Use the second date in the range
	}

	// Parse the date in YYYY-MM-DD format
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, err
	}

	// Check if date is in the future, if so, adjust the year to make it in the past
	now := time.Now()
	for t.After(now) {
		t = t.AddDate(-1, 0, 0) // Subtract 1 year
	}

	return t, nil
}
