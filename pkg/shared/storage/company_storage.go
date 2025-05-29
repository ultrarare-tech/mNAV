package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

// CompanyDataStorage manages storage of company financial data
type CompanyDataStorage struct {
	baseDir string
}

// NewCompanyDataStorage creates a new company data storage manager
func NewCompanyDataStorage(baseDir string) *CompanyDataStorage {
	return &CompanyDataStorage{
		baseDir: baseDir,
	}
}

// SaveCompanyData saves company financial data to JSON file
func (s *CompanyDataStorage) SaveCompanyData(data *models.CompanyFinancialData) error {
	// Create company directory
	companyDir := filepath.Join(s.baseDir, data.Symbol)
	if err := os.MkdirAll(companyDir, 0755); err != nil {
		return fmt.Errorf("failed to create company directory: %w", err)
	}

	// Save main data file
	dataPath := filepath.Join(companyDir, "financial_data.json")
	dataBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal company data: %w", err)
	}

	if err := os.WriteFile(dataPath, dataBytes, 0644); err != nil {
		return fmt.Errorf("failed to write company data: %w", err)
	}

	return nil
}

// LoadCompanyData loads company financial data from JSON file
func (s *CompanyDataStorage) LoadCompanyData(symbol string) (*models.CompanyFinancialData, error) {
	dataPath := filepath.Join(s.baseDir, symbol, "financial_data.json")

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no data found for symbol %s", symbol)
	}

	dataBytes, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read company data: %w", err)
	}

	var companyData models.CompanyFinancialData
	if err := json.Unmarshal(dataBytes, &companyData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal company data: %w", err)
	}

	return &companyData, nil
}

// AddSharesRecord adds a new shares outstanding record to the company data
func (s *CompanyDataStorage) AddSharesRecord(symbol string, record *models.SharesOutstandingRecord) error {
	data, err := s.LoadCompanyData(symbol)
	if err != nil {
		// Create new company data if it doesn't exist
		data = &models.CompanyFinancialData{
			Symbol:        symbol,
			SharesHistory: []models.SharesOutstandingRecord{},
		}
	}

	// Add the new record
	data.SharesHistory = append(data.SharesHistory, *record)

	// Sort by date
	sort.Slice(data.SharesHistory, func(i, j int) bool {
		return data.SharesHistory[i].Date.Before(data.SharesHistory[j].Date)
	})

	data.LastUpdated = time.Now()
	return s.SaveCompanyData(data)
}

// AddBitcoinTransaction adds a new Bitcoin transaction to the company data
func (s *CompanyDataStorage) AddBitcoinTransaction(symbol string, transaction *models.BitcoinTransaction) error {
	data, err := s.LoadCompanyData(symbol)
	if err != nil {
		// Create new company data if it doesn't exist
		data = &models.CompanyFinancialData{
			Symbol:          symbol,
			BTCTransactions: []models.BitcoinTransaction{},
		}
	}

	// Add the new transaction
	data.BTCTransactions = append(data.BTCTransactions, *transaction)

	// Sort by date
	sort.Slice(data.BTCTransactions, func(i, j int) bool {
		return data.BTCTransactions[i].Date.Before(data.BTCTransactions[j].Date)
	})

	data.LastUpdated = time.Now()
	return s.SaveCompanyData(data)
}

// GetLatestShares returns the most recent shares outstanding for a company
func (s *CompanyDataStorage) GetLatestShares(symbol string) (float64, error) {
	data, err := s.LoadCompanyData(symbol)
	if err != nil {
		return 0, err
	}

	if len(data.SharesHistory) == 0 {
		return 0, fmt.Errorf("no shares data found for %s", symbol)
	}

	// Get the most recent record
	latest := data.SharesHistory[len(data.SharesHistory)-1]
	return latest.TotalShares, nil
}

// GetTotalBTC returns the total Bitcoin holdings for a company
func (s *CompanyDataStorage) GetTotalBTC(symbol string) (float64, error) {
	data, err := s.LoadCompanyData(symbol)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, tx := range data.BTCTransactions {
		total += tx.BTCPurchased
	}

	return total, nil
}
