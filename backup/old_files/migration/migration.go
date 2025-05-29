package edgar

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrationService handles data migration from old format to new format
type MigrationService struct {
	storage *CompanyDataStorage
}

// NewMigrationService creates a new migration service
func NewMigrationService(storage *CompanyDataStorage) *MigrationService {
	return &MigrationService{
		storage: storage,
	}
}

// LegacyCompanyData represents the old company data format
type LegacyCompanyData struct {
	Symbol            string    `json:"symbol"`
	Name              string    `json:"name"`
	OutstandingShares float64   `json:"outstandingShares"`
	BTCHoldings       float64   `json:"btcHoldings"`
	BTCYield          float64   `json:"btcYield"`
	MarketCap         float64   `json:"marketCap"`
	LastUpdated       time.Time `json:"lastUpdated"`
}

// CompaniesJSONWrapper represents the wrapper structure in companies.json
type CompaniesJSONWrapper struct {
	Companies []LegacyCompanyData `json:"companies"`
}

// MigrateCompaniesJSON migrates data from companies.json to the new format
func (m *MigrationService) MigrateCompaniesJSON(companiesJSONPath string) error {
	// Read the companies.json file
	data, err := ioutil.ReadFile(companiesJSONPath)
	if err != nil {
		return fmt.Errorf("error reading companies.json: %w", err)
	}

	// Try parsing as wrapper first
	var wrapper CompaniesJSONWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Try parsing as array directly
		var companies []LegacyCompanyData
		if err := json.Unmarshal(data, &companies); err != nil {
			return fmt.Errorf("error parsing companies.json: %w", err)
		}
		wrapper.Companies = companies
	}

	// Migrate each company
	for _, legacy := range wrapper.Companies {
		fmt.Printf("Migrating %s...\n", legacy.Symbol)

		// Create new format data
		companyData := &CompanyFinancialData{
			Symbol:      legacy.Symbol,
			CompanyName: legacy.Name,
			CIK:         "", // Will be filled by EDGAR lookups
			LastUpdated: legacy.LastUpdated,
		}

		// Add shares history with a single record
		if legacy.OutstandingShares > 0 {
			sharesRecord := SharesOutstandingRecord{
				Date:            legacy.LastUpdated,
				FilingType:      "Manual",
				FilingURL:       "companies.json",
				AccessionNumber: "manual-import",
				CommonShares:    legacy.OutstandingShares,
				TotalShares:     legacy.OutstandingShares,
				ExtractedFrom:   "companies.json",
				ExtractedText:   fmt.Sprintf("Outstanding shares: %.0f", legacy.OutstandingShares),
				ConfidenceScore: 0.5, // Low confidence for manual data
				Notes:           "Imported from legacy companies.json",
			}
			companyData.SharesHistory = append(companyData.SharesHistory, sharesRecord)
		}

		// Add BTC holdings as a transaction
		if legacy.BTCHoldings > 0 {
			btcTransaction := BitcoinTransaction{
				Date:            legacy.LastUpdated,
				FilingType:      "Manual",
				FilingURL:       "companies.json",
				BTCPurchased:    legacy.BTCHoldings,
				USDSpent:        0, // Unknown from legacy data
				AvgPriceUSD:     0, // Unknown from legacy data
				TotalBTCAfter:   legacy.BTCHoldings,
				ExtractedText:   fmt.Sprintf("BTC Holdings: %.2f", legacy.BTCHoldings),
				ConfidenceScore: 0.5, // Low confidence for manual data
			}
			companyData.BTCTransactions = append(companyData.BTCTransactions, btcTransaction)
		}

		// Save the migrated data
		if err := m.storage.SaveCompanyData(companyData); err != nil {
			fmt.Printf("Error saving %s: %v\n", legacy.Symbol, err)
			continue
		}

		fmt.Printf("Successfully migrated %s\n", legacy.Symbol)
	}

	return nil
}

// MigrateTransactionFiles migrates old transaction files to the new format
func (m *MigrationService) MigrateTransactionFiles(transactionsDir string) error {
	// Read all transaction files
	entries, err := ioutil.ReadDir(transactionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No transactions to migrate
		}
		return fmt.Errorf("error reading transactions directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			filePath := filepath.Join(transactionsDir, entry.Name())

			// Read the transaction file
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading %s: %v\n", entry.Name(), err)
				continue
			}

			// Parse the transactions
			var companyTx CompanyTransactions
			if err := json.Unmarshal(data, &companyTx); err != nil {
				fmt.Printf("Error parsing %s: %v\n", entry.Name(), err)
				continue
			}

			// Extract ticker from filename (e.g., "MSTR_transactions.json" -> "MSTR")
			ticker := companyTx.Company
			if ticker == "" {
				// Try to extract from filename
				ticker = entry.Name()
				ticker = strings.TrimSuffix(ticker, "_transactions.json")
				ticker = strings.TrimSuffix(ticker, ".json")
			}

			fmt.Printf("Migrating transactions for %s...\n", ticker)

			// Load existing company data or create new
			companyData, err := m.storage.LoadCompanyData(ticker)
			if err != nil {
				// Create new company data
				companyData = &CompanyFinancialData{
					Symbol:      ticker,
					CompanyName: ticker,
					CIK:         companyTx.CIK,
					LastUpdated: companyTx.LastUpdated,
				}
			}

			// Add transactions
			companyData.BTCTransactions = append(companyData.BTCTransactions, companyTx.Transactions...)

			// Save the updated data
			if err := m.storage.SaveCompanyData(companyData); err != nil {
				fmt.Printf("Error saving %s: %v\n", ticker, err)
				continue
			}

			fmt.Printf("Successfully migrated %d transactions for %s\n", len(companyTx.Transactions), ticker)
		}
	}

	return nil
}

// CreateCompatibilityAdapter creates a compatibility layer for old code
func (m *MigrationService) CreateCompatibilityAdapter(outputPath string) error {
	// Get all companies with data
	companies, err := m.storage.ListCompanies()
	if err != nil {
		return fmt.Errorf("error listing companies: %w", err)
	}

	// Create legacy format array
	var legacyCompanies []LegacyCompanyData

	for _, symbol := range companies {
		// Load company data
		companyData, err := m.storage.LoadCompanyData(symbol)
		if err != nil {
			fmt.Printf("Error loading %s: %v\n", symbol, err)
			continue
		}

		// Get latest shares
		var latestShares float64
		if len(companyData.SharesHistory) > 0 {
			latestShares = companyData.SharesHistory[len(companyData.SharesHistory)-1].TotalShares
		}

		// Calculate total BTC holdings
		var totalBTC float64
		for _, tx := range companyData.BTCTransactions {
			totalBTC += tx.BTCPurchased
		}

		// Create legacy format entry
		legacy := LegacyCompanyData{
			Symbol:            companyData.Symbol,
			Name:              companyData.CompanyName,
			OutstandingShares: latestShares,
			BTCHoldings:       totalBTC,
			BTCYield:          0, // Would need to be calculated separately
			MarketCap:         0, // Would need current stock price
			LastUpdated:       companyData.LastUpdated,
		}

		legacyCompanies = append(legacyCompanies, legacy)
	}

	// Write to output file
	data, err := json.MarshalIndent(legacyCompanies, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling legacy data: %w", err)
	}

	if err := ioutil.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("error writing compatibility file: %w", err)
	}

	fmt.Printf("Created compatibility file with %d companies at %s\n", len(legacyCompanies), outputPath)
	return nil
}
