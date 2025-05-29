package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
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

// SaveBTCTransactions saves Bitcoin transactions to a JSON file
func (s *TransactionStorage) SaveBTCTransactions(symbol string, transactions []models.BitcoinTransaction) error {
	// Create the data directory if it doesn't exist
	dataDir := filepath.Join(s.basePath, "data", "transactions")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file path for the company
	filePath := filepath.Join(dataDir, fmt.Sprintf("%s_btc.json", symbol))

	// Create transaction data structure
	transactionData := struct {
		Symbol       string                      `json:"symbol"`
		Transactions []models.BitcoinTransaction `json:"transactions"`
		LastUpdated  time.Time                   `json:"lastUpdated"`
	}{
		Symbol:       symbol,
		Transactions: transactions,
		LastUpdated:  time.Now(),
	}

	// Marshal the JSON with indentation for readability
	data, err := json.MarshalIndent(transactionData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling transactions: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// LoadBTCTransactions loads Bitcoin transactions from a JSON file
func (s *TransactionStorage) LoadBTCTransactions(symbol string) ([]models.BitcoinTransaction, error) {
	// Create the file path for the company
	filePath := filepath.Join(s.basePath, "data", "transactions", fmt.Sprintf("%s_btc.json", symbol))

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// If file doesn't exist, return empty slice
		return []models.BitcoinTransaction{}, nil
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Unmarshal the JSON
	var transactionData struct {
		Symbol       string                      `json:"symbol"`
		Transactions []models.BitcoinTransaction `json:"transactions"`
		LastUpdated  time.Time                   `json:"lastUpdated"`
	}

	if err := json.Unmarshal(data, &transactionData); err != nil {
		return nil, fmt.Errorf("error unmarshaling transactions: %w", err)
	}

	return transactionData.Transactions, nil
}
