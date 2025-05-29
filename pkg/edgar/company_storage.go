package edgar

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CompanyDataStorage handles storage of comprehensive company financial data
type CompanyDataStorage struct {
	basePath string
}

// NewCompanyDataStorage creates a new company data storage instance
func NewCompanyDataStorage(basePath string) *CompanyDataStorage {
	return &CompanyDataStorage{
		basePath: basePath,
	}
}

// SaveCompanyData saves company financial data to disk
func (s *CompanyDataStorage) SaveCompanyData(data *CompanyFinancialData) error {
	// Create directory structure
	companyDir := filepath.Join(s.basePath, "companies", data.Symbol)
	if err := os.MkdirAll(companyDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Sort shares history by date
	sort.Slice(data.SharesHistory, func(i, j int) bool {
		return data.SharesHistory[i].Date.Before(data.SharesHistory[j].Date)
	})

	// Sort BTC transactions by date
	sort.Slice(data.BTCTransactions, func(i, j int) bool {
		return data.BTCTransactions[i].Date.Before(data.BTCTransactions[j].Date)
	})

	// Save main company data file
	mainFile := filepath.Join(companyDir, "financial_data.json")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling company data: %w", err)
	}

	if err := ioutil.WriteFile(mainFile, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing company data: %w", err)
	}

	// Save separate files for shares history and BTC transactions for easy access
	// Save shares history
	if len(data.SharesHistory) > 0 {
		sharesFile := filepath.Join(companyDir, "shares_history.json")
		sharesData, err := json.MarshalIndent(data.SharesHistory, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling shares history: %w", err)
		}
		if err := ioutil.WriteFile(sharesFile, sharesData, 0644); err != nil {
			return fmt.Errorf("error writing shares history: %w", err)
		}
	}

	// Save BTC transactions
	if len(data.BTCTransactions) > 0 {
		btcFile := filepath.Join(companyDir, "btc_transactions.json")
		btcData, err := json.MarshalIndent(data.BTCTransactions, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshaling BTC transactions: %w", err)
		}
		if err := ioutil.WriteFile(btcFile, btcData, 0644); err != nil {
			return fmt.Errorf("error writing BTC transactions: %w", err)
		}
	}

	// Save latest snapshot
	snapshot := s.createSnapshot(data)
	snapshotFile := filepath.Join(companyDir, "latest_snapshot.json")
	snapshotData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling snapshot: %w", err)
	}
	if err := ioutil.WriteFile(snapshotFile, snapshotData, 0644); err != nil {
		return fmt.Errorf("error writing snapshot: %w", err)
	}

	return nil
}

// LoadCompanyData loads company financial data from disk
func (s *CompanyDataStorage) LoadCompanyData(symbol string) (*CompanyFinancialData, error) {
	companyDir := filepath.Join(s.basePath, "companies", symbol)
	mainFile := filepath.Join(companyDir, "financial_data.json")

	data, err := ioutil.ReadFile(mainFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no data found for company %s", symbol)
		}
		return nil, fmt.Errorf("error reading company data: %w", err)
	}

	var companyData CompanyFinancialData
	if err := json.Unmarshal(data, &companyData); err != nil {
		return nil, fmt.Errorf("error unmarshaling company data: %w", err)
	}

	return &companyData, nil
}

// GetLatestSharesOutstanding returns the most recent shares outstanding for a company
func (s *CompanyDataStorage) GetLatestSharesOutstanding(symbol string) (*SharesOutstandingRecord, error) {
	companyData, err := s.LoadCompanyData(symbol)
	if err != nil {
		return nil, err
	}

	if len(companyData.SharesHistory) == 0 {
		return nil, fmt.Errorf("no shares outstanding data found for %s", symbol)
	}

	// Return the most recent record
	return &companyData.SharesHistory[len(companyData.SharesHistory)-1], nil
}

// GetSharesAtDate returns shares outstanding at a specific date
func (s *CompanyDataStorage) GetSharesAtDate(symbol string, date time.Time) (*SharesOutstandingRecord, error) {
	companyData, err := s.LoadCompanyData(symbol)
	if err != nil {
		return nil, err
	}

	if len(companyData.SharesHistory) == 0 {
		return nil, fmt.Errorf("no shares outstanding data found for %s", symbol)
	}

	// Find the most recent record before or on the given date
	var bestRecord *SharesOutstandingRecord
	for i := range companyData.SharesHistory {
		record := &companyData.SharesHistory[i]
		if record.Date.After(date) {
			break
		}
		bestRecord = record
	}

	if bestRecord == nil {
		return nil, fmt.Errorf("no shares outstanding data found for %s before %s", symbol, date.Format("2006-01-02"))
	}

	return bestRecord, nil
}

// GetTotalBTCHoldings calculates total BTC holdings for a company
func (s *CompanyDataStorage) GetTotalBTCHoldings(symbol string) (float64, error) {
	companyData, err := s.LoadCompanyData(symbol)
	if err != nil {
		return 0, err
	}

	totalBTC := 0.0
	for _, tx := range companyData.BTCTransactions {
		totalBTC += tx.BTCPurchased
	}

	return totalBTC, nil
}

// GetBTCHoldingsAtDate calculates BTC holdings at a specific date
func (s *CompanyDataStorage) GetBTCHoldingsAtDate(symbol string, date time.Time) (float64, error) {
	companyData, err := s.LoadCompanyData(symbol)
	if err != nil {
		return 0, err
	}

	totalBTC := 0.0
	for _, tx := range companyData.BTCTransactions {
		if tx.Date.After(date) {
			break
		}
		totalBTC += tx.BTCPurchased
	}

	return totalBTC, nil
}

// createSnapshot creates a current snapshot of company data
func (s *CompanyDataStorage) createSnapshot(data *CompanyFinancialData) *CompanyDataSnapshot {
	snapshot := &CompanyDataSnapshot{
		Symbol:       data.Symbol,
		SnapshotDate: time.Now(),
		BTCHoldings:  0,
	}

	// Calculate total BTC holdings
	for _, tx := range data.BTCTransactions {
		snapshot.BTCHoldings += tx.BTCPurchased
	}

	// Get latest shares outstanding
	if len(data.SharesHistory) > 0 {
		latestShares := data.SharesHistory[len(data.SharesHistory)-1]
		snapshot.SharesOutstanding = AuditableSharesData{
			Value:    latestShares.TotalShares,
			AsOfDate: latestShares.Date,
			Source: DataSource{
				Type:       "SEC",
				URL:        latestShares.FilingURL,
				Date:       latestShares.Date,
				Confidence: latestShares.ConfidenceScore,
			},
			LastChecked: time.Now(),
		}
	}

	return snapshot
}

// MergeCompanyData merges new data with existing data
func (s *CompanyDataStorage) MergeCompanyData(symbol string, newData *CompanyFinancialData) error {
	// Load existing data if it exists
	existingData, err := s.LoadCompanyData(symbol)
	if err != nil {
		// If no existing data, just save the new data
		if os.IsNotExist(err) || err.Error() == fmt.Sprintf("no data found for company %s", symbol) {
			return s.SaveCompanyData(newData)
		}
		return err
	}

	// Merge shares history
	sharesMap := make(map[string]*SharesOutstandingRecord)
	for i := range existingData.SharesHistory {
		record := &existingData.SharesHistory[i]
		key := fmt.Sprintf("%s_%s", record.AccessionNumber, record.Date.Format("2006-01-02"))
		sharesMap[key] = record
	}
	for i := range newData.SharesHistory {
		record := &newData.SharesHistory[i]
		key := fmt.Sprintf("%s_%s", record.AccessionNumber, record.Date.Format("2006-01-02"))
		// Only add if it's a new record or has higher confidence
		if existing, exists := sharesMap[key]; !exists || record.ConfidenceScore > existing.ConfidenceScore {
			sharesMap[key] = record
		}
	}

	// Convert back to slice
	mergedShares := make([]SharesOutstandingRecord, 0, len(sharesMap))
	for _, record := range sharesMap {
		mergedShares = append(mergedShares, *record)
	}
	existingData.SharesHistory = mergedShares

	// Merge BTC transactions
	btcMap := make(map[string]*BitcoinTransaction)
	for i := range existingData.BTCTransactions {
		tx := &existingData.BTCTransactions[i]
		key := fmt.Sprintf("%s_%s_%.2f", tx.FilingURL, tx.Date.Format("2006-01-02"), tx.BTCPurchased)
		btcMap[key] = tx
	}
	for i := range newData.BTCTransactions {
		tx := &newData.BTCTransactions[i]
		key := fmt.Sprintf("%s_%s_%.2f", tx.FilingURL, tx.Date.Format("2006-01-02"), tx.BTCPurchased)
		// Only add if it's a new transaction or has higher confidence
		if existing, exists := btcMap[key]; !exists || tx.ConfidenceScore > existing.ConfidenceScore {
			btcMap[key] = tx
		}
	}

	// Convert back to slice
	mergedBTC := make([]BitcoinTransaction, 0, len(btcMap))
	for _, tx := range btcMap {
		mergedBTC = append(mergedBTC, *tx)
	}
	existingData.BTCTransactions = mergedBTC

	// Update metadata
	existingData.LastUpdated = time.Now()
	if newData.LastFilingDate.After(existingData.LastFilingDate) {
		existingData.LastFilingDate = newData.LastFilingDate
	}
	existingData.LastProcessedDate = time.Now()

	// Save the merged data
	return s.SaveCompanyData(existingData)
}

// GetProcessingMetadata returns metadata about the last processing for a company
func (s *CompanyDataStorage) GetProcessingMetadata(symbol string) (*FilingMetadata, error) {
	companyData, err := s.LoadCompanyData(symbol)
	if err != nil {
		return nil, err
	}

	metadata := &FilingMetadata{
		ProcessedDate:   companyData.LastProcessedDate,
		SharesExtracted: len(companyData.SharesHistory) > 0,
		BTCExtracted:    len(companyData.BTCTransactions) > 0,
	}

	// Get the most recent filing info
	if companyData.LastFilingDate.After(time.Time{}) {
		metadata.FilingDate = companyData.LastFilingDate
	}

	return metadata, nil
}

// ListCompanies returns a list of all companies with stored data
func (s *CompanyDataStorage) ListCompanies() ([]string, error) {
	companiesDir := filepath.Join(s.basePath, "companies")

	entries, err := ioutil.ReadDir(companiesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("error reading companies directory: %w", err)
	}

	var companies []string
	for _, entry := range entries {
		if entry.IsDir() {
			companies = append(companies, entry.Name())
		}
	}

	return companies, nil
}

// SaveRawFiling saves a raw filing document to disk
func (s *CompanyDataStorage) SaveRawFiling(symbol string, filing Filing, content []byte) (*RawFilingDocument, error) {
	// Create directory structure
	companyDir := filepath.Join(s.basePath, "companies", symbol)
	rawFilingsDir := filepath.Join(companyDir, "raw_filings")
	if err := os.MkdirAll(rawFilingsDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating raw filings directory: %w", err)
	}

	// Calculate checksum
	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	// Determine content type
	contentType := "text/html"
	if strings.Contains(string(content[:min(1024, len(content))]), "<?xml") {
		contentType = "application/xml"
	} else if !strings.Contains(string(content[:min(1024, len(content))]), "<") {
		contentType = "text/plain"
	}

	// Create raw filing document metadata
	rawDoc := &RawFilingDocument{
		AccessionNumber: filing.AccessionNumber,
		FilingType:      filing.FilingType,
		FilingDate:      filing.FilingDate,
		CompanySymbol:   symbol,
		DocumentURL:     filing.DocumentURL,
		ContentType:     contentType,
		ContentLength:   int64(len(content)),
		DownloadedAt:    time.Now(),
		Checksum:        checksum,
	}

	// Generate filename: YYYY-MM-DD_TYPE_ACCESSION.ext
	ext := ".html"
	if contentType == "text/plain" {
		ext = ".txt"
	} else if contentType == "application/xml" {
		ext = ".xml"
	}

	filename := fmt.Sprintf("%s_%s_%s%s",
		filing.FilingDate.Format("2006-01-02"),
		filing.FilingType,
		filing.AccessionNumber,
		ext)

	// Save the raw content
	contentPath := filepath.Join(rawFilingsDir, filename)
	if err := ioutil.WriteFile(contentPath, content, 0644); err != nil {
		return nil, fmt.Errorf("error writing raw filing content: %w", err)
	}

	// Save the metadata
	metadataPath := filepath.Join(rawFilingsDir, strings.TrimSuffix(filename, ext)+".json")
	metadataJSON, err := json.MarshalIndent(rawDoc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling raw filing metadata: %w", err)
	}

	if err := ioutil.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return nil, fmt.Errorf("error writing raw filing metadata: %w", err)
	}

	return rawDoc, nil
}

// LoadRawFiling loads a raw filing document and its content
func (s *CompanyDataStorage) LoadRawFiling(symbol, accessionNumber string) (*RawFilingDocument, []byte, error) {
	companyDir := filepath.Join(s.basePath, "companies", symbol)
	rawFilingsDir := filepath.Join(companyDir, "raw_filings")

	// Find the metadata file
	entries, err := ioutil.ReadDir(rawFilingsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading raw filings directory: %w", err)
	}

	var metadataPath, contentPath string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), accessionNumber) && strings.HasSuffix(entry.Name(), ".json") {
			metadataPath = filepath.Join(rawFilingsDir, entry.Name())
			// Find corresponding content file
			baseName := strings.TrimSuffix(entry.Name(), ".json")
			for _, contentEntry := range entries {
				if strings.HasPrefix(contentEntry.Name(), baseName) && !strings.HasSuffix(contentEntry.Name(), ".json") {
					contentPath = filepath.Join(rawFilingsDir, contentEntry.Name())
					break
				}
			}
			break
		}
	}

	if metadataPath == "" {
		return nil, nil, fmt.Errorf("raw filing not found for accession number: %s", accessionNumber)
	}

	// Load metadata
	metadataData, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading raw filing metadata: %w", err)
	}

	var rawDoc RawFilingDocument
	if err := json.Unmarshal(metadataData, &rawDoc); err != nil {
		return nil, nil, fmt.Errorf("error unmarshaling raw filing metadata: %w", err)
	}

	// Load content
	var content []byte
	if contentPath != "" {
		content, err = ioutil.ReadFile(contentPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading raw filing content: %w", err)
		}

		// Verify checksum
		hash := sha256.Sum256(content)
		checksum := hex.EncodeToString(hash[:])
		if checksum != rawDoc.Checksum {
			return nil, nil, fmt.Errorf("checksum mismatch for raw filing: expected %s, got %s", rawDoc.Checksum, checksum)
		}
	}

	return &rawDoc, content, nil
}

// ListRawFilings returns a list of all raw filings for a company
func (s *CompanyDataStorage) ListRawFilings(symbol string) ([]RawFilingDocument, error) {
	companyDir := filepath.Join(s.basePath, "companies", symbol)
	rawFilingsDir := filepath.Join(companyDir, "raw_filings")

	entries, err := ioutil.ReadDir(rawFilingsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RawFilingDocument{}, nil
		}
		return nil, fmt.Errorf("error reading raw filings directory: %w", err)
	}

	var rawFilings []RawFilingDocument
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			metadataPath := filepath.Join(rawFilingsDir, entry.Name())
			metadataData, err := ioutil.ReadFile(metadataPath)
			if err != nil {
				continue // Skip files we can't read
			}

			var rawDoc RawFilingDocument
			if err := json.Unmarshal(metadataData, &rawDoc); err != nil {
				continue // Skip files we can't parse
			}

			rawFilings = append(rawFilings, rawDoc)
		}
	}

	// Sort by filing date
	sort.Slice(rawFilings, func(i, j int) bool {
		return rawFilings[i].FilingDate.Before(rawFilings[j].FilingDate)
	})

	return rawFilings, nil
}

// SaveProcessingResult saves the result of processing a raw filing
func (s *CompanyDataStorage) SaveProcessingResult(symbol string, result *FilingProcessingResult) error {
	companyDir := filepath.Join(s.basePath, "companies", symbol)
	processingDir := filepath.Join(companyDir, "processing_results")
	if err := os.MkdirAll(processingDir, 0755); err != nil {
		return fmt.Errorf("error creating processing results directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%s_%s_%s_result.json",
		result.Document.FilingDate.Format("2006-01-02"),
		result.Document.FilingType,
		result.Document.AccessionNumber)

	filePath := filepath.Join(processingDir, filename)
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling processing result: %w", err)
	}

	if err := ioutil.WriteFile(filePath, resultJSON, 0644); err != nil {
		return fmt.Errorf("error writing processing result: %w", err)
	}

	return nil
}

// GetRawFilingStats returns statistics about raw filings for a company
func (s *CompanyDataStorage) GetRawFilingStats(symbol string) (map[string]interface{}, error) {
	rawFilings, err := s.ListRawFilings(symbol)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_filings":    len(rawFilings),
		"filing_types":     make(map[string]int),
		"total_size_bytes": int64(0),
		"date_range":       map[string]interface{}{},
	}

	if len(rawFilings) == 0 {
		return stats, nil
	}

	filingTypes := make(map[string]int)
	var totalSize int64
	var earliestDate, latestDate time.Time

	for i, filing := range rawFilings {
		filingTypes[filing.FilingType]++
		totalSize += filing.ContentLength

		if i == 0 {
			earliestDate = filing.FilingDate
			latestDate = filing.FilingDate
		} else {
			if filing.FilingDate.Before(earliestDate) {
				earliestDate = filing.FilingDate
			}
			if filing.FilingDate.After(latestDate) {
				latestDate = filing.FilingDate
			}
		}
	}

	stats["filing_types"] = filingTypes
	stats["total_size_bytes"] = totalSize
	stats["date_range"] = map[string]interface{}{
		"earliest": earliestDate.Format("2006-01-02"),
		"latest":   latestDate.Format("2006-01-02"),
	}

	return stats, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MergeExtractedData merges extracted financial data from a single filing into existing company data
func (s *CompanyDataStorage) MergeExtractedData(symbol string, extractedData *ExtractedFinancialData) error {
	// Load existing data or create new
	existingData, err := s.LoadCompanyData(symbol)
	if err != nil {
		// Create new company data if none exists
		existingData = &CompanyFinancialData{
			Symbol:      symbol,
			CompanyName: symbol,
			LastUpdated: time.Now(),
		}
	}

	// Merge BTC transactions
	if len(extractedData.BTCTransactions) > 0 {
		existingData.BTCTransactions = append(existingData.BTCTransactions, extractedData.BTCTransactions...)
	}

	// Merge shares outstanding
	if extractedData.SharesOutstanding != nil {
		existingData.SharesHistory = append(existingData.SharesHistory, *extractedData.SharesOutstanding)
	}

	// Update timestamps
	existingData.LastUpdated = time.Now()
	if extractedData.Filing.FilingDate.After(existingData.LastFilingDate) {
		existingData.LastFilingDate = extractedData.Filing.FilingDate
	}
	existingData.LastProcessedDate = time.Now()

	// Save the merged data
	return s.SaveCompanyData(existingData)
}
