package edgar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TransactionStorage handles storing and retrieving Bitcoin transactions
type TransactionStorage struct {
	basePath string
}

// NewTransactionStorage creates a new transaction storage manager
func NewTransactionStorage(basePath string) *TransactionStorage {
	return &TransactionStorage{
		basePath: basePath,
	}
}

// SaveTransactions saves company transactions to a JSON file
func (s *TransactionStorage) SaveTransactions(transactions *CompanyTransactions) error {
	// Create the data directory if it doesn't exist
	dataDir := filepath.Join(s.basePath, "data", "transactions")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file path for the company
	filePath := filepath.Join(dataDir, fmt.Sprintf("%s.json", transactions.Company))

	// Marshal the JSON with indentation for readability
	data, err := json.MarshalIndent(transactions, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling transactions: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// LoadTransactions loads company transactions from a JSON file
func (s *TransactionStorage) LoadTransactions(ticker string) (*CompanyTransactions, error) {
	// Create the file path for the company
	filePath := filepath.Join(s.basePath, "data", "transactions", fmt.Sprintf("%s.json", ticker))

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// If file doesn't exist, return an empty transactions struct
		return &CompanyTransactions{
			Company:      ticker,
			Transactions: []BitcoinTransaction{},
		}, nil
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Unmarshal the JSON
	var transactions CompanyTransactions
	if err := json.Unmarshal(data, &transactions); err != nil {
		return nil, fmt.Errorf("error unmarshaling transactions: %w", err)
	}

	return &transactions, nil
}

// MergeTransactions merges new transactions with existing ones, avoiding duplicates
func (s *TransactionStorage) MergeTransactions(existing *CompanyTransactions, new *CompanyTransactions) *CompanyTransactions {
	// If existing is empty, just return the new transactions
	if len(existing.Transactions) == 0 {
		return new
	}

	// Create a map of existing transactions for quick lookup
	existingMap := make(map[string]bool)
	for _, tx := range existing.Transactions {
		// Create a key using the date and BTC amount to identify unique transactions
		key := fmt.Sprintf("%s_%.2f", tx.Date.Format("2006-01-02"), tx.BTCPurchased)
		existingMap[key] = true
	}

	// Add new transactions that don't exist in the existing set
	for _, tx := range new.Transactions {
		key := fmt.Sprintf("%s_%.2f", tx.Date.Format("2006-01-02"), tx.BTCPurchased)
		if !existingMap[key] {
			existing.Transactions = append(existing.Transactions, tx)
			existingMap[key] = true
		}
	}

	// Update the last updated timestamp
	existing.LastUpdated = new.LastUpdated

	return existing
}
