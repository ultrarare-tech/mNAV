package edgar

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompanyDataStorage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "edgar_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := NewCompanyDataStorage(tempDir)

	// Create test data
	testData := &CompanyFinancialData{
		Symbol:      "MSTR",
		CompanyName: "MicroStrategy Inc.",
		CIK:         "1050446",
		SharesHistory: []SharesOutstandingRecord{
			{
				Date:            time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
				FilingType:      "10-K",
				FilingURL:       "https://sec.gov/example1",
				AccessionNumber: "0001234567-23-000001",
				CommonShares:    95000000,
				TotalShares:     95000000,
				ExtractedFrom:   "Balance Sheet",
				ConfidenceScore: 0.9,
			},
			{
				Date:            time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
				FilingType:      "10-Q",
				FilingURL:       "https://sec.gov/example2",
				AccessionNumber: "0001234567-24-000001",
				CommonShares:    100000000,
				TotalShares:     100000000,
				ExtractedFrom:   "Balance Sheet",
				ConfidenceScore: 0.95,
			},
		},
		BTCTransactions: []BitcoinTransaction{
			{
				Date:            time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				FilingType:      "8-K",
				FilingURL:       "https://sec.gov/example3",
				BTCPurchased:    1000,
				USDSpent:        45000000,
				AvgPriceUSD:     45000,
				ConfidenceScore: 0.9,
			},
		},
		LastUpdated:    time.Now(),
		LastFilingDate: time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
	}

	// Test SaveCompanyData
	t.Run("SaveCompanyData", func(t *testing.T) {
		err := storage.SaveCompanyData(testData)
		if err != nil {
			t.Fatalf("Failed to save company data: %v", err)
		}

		// Check that files were created
		companyDir := filepath.Join(tempDir, "companies", "MSTR")
		if _, err := os.Stat(companyDir); os.IsNotExist(err) {
			t.Errorf("Company directory was not created")
		}

		mainFile := filepath.Join(companyDir, "financial_data.json")
		if _, err := os.Stat(mainFile); os.IsNotExist(err) {
			t.Errorf("Main data file was not created")
		}

		sharesFile := filepath.Join(companyDir, "shares_history.json")
		if _, err := os.Stat(sharesFile); os.IsNotExist(err) {
			t.Errorf("Shares history file was not created")
		}

		btcFile := filepath.Join(companyDir, "btc_transactions.json")
		if _, err := os.Stat(btcFile); os.IsNotExist(err) {
			t.Errorf("BTC transactions file was not created")
		}

		snapshotFile := filepath.Join(companyDir, "latest_snapshot.json")
		if _, err := os.Stat(snapshotFile); os.IsNotExist(err) {
			t.Errorf("Snapshot file was not created")
		}
	})

	// Test LoadCompanyData
	t.Run("LoadCompanyData", func(t *testing.T) {
		loadedData, err := storage.LoadCompanyData("MSTR")
		if err != nil {
			t.Fatalf("Failed to load company data: %v", err)
		}

		if loadedData.Symbol != testData.Symbol {
			t.Errorf("Symbol mismatch: got %s, want %s", loadedData.Symbol, testData.Symbol)
		}

		if len(loadedData.SharesHistory) != len(testData.SharesHistory) {
			t.Errorf("Shares history length mismatch: got %d, want %d",
				len(loadedData.SharesHistory), len(testData.SharesHistory))
		}

		if len(loadedData.BTCTransactions) != len(testData.BTCTransactions) {
			t.Errorf("BTC transactions length mismatch: got %d, want %d",
				len(loadedData.BTCTransactions), len(testData.BTCTransactions))
		}
	})

	// Test GetLatestSharesOutstanding
	t.Run("GetLatestSharesOutstanding", func(t *testing.T) {
		latest, err := storage.GetLatestSharesOutstanding("MSTR")
		if err != nil {
			t.Fatalf("Failed to get latest shares: %v", err)
		}

		if latest.TotalShares != 100000000 {
			t.Errorf("Expected 100000000 shares, got %f", latest.TotalShares)
		}

		if latest.Date != time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC) {
			t.Errorf("Expected date 2024-03-31, got %s", latest.Date.Format("2006-01-02"))
		}
	})

	// Test GetSharesAtDate
	t.Run("GetSharesAtDate", func(t *testing.T) {
		// Test getting shares at a date between two records
		sharesAt, err := storage.GetSharesAtDate("MSTR", time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("Failed to get shares at date: %v", err)
		}

		if sharesAt.TotalShares != 95000000 {
			t.Errorf("Expected 95000000 shares at 2024-02-15, got %f", sharesAt.TotalShares)
		}

		// Test getting shares at the exact date of a record
		sharesAt2, err := storage.GetSharesAtDate("MSTR", time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("Failed to get shares at date: %v", err)
		}

		if sharesAt2.TotalShares != 100000000 {
			t.Errorf("Expected 100000000 shares at 2024-03-31, got %f", sharesAt2.TotalShares)
		}
	})

	// Test GetTotalBTCHoldings
	t.Run("GetTotalBTCHoldings", func(t *testing.T) {
		total, err := storage.GetTotalBTCHoldings("MSTR")
		if err != nil {
			t.Fatalf("Failed to get total BTC holdings: %v", err)
		}

		if total != 1000 {
			t.Errorf("Expected 1000 BTC total, got %f", total)
		}
	})

	// Test MergeCompanyData
	t.Run("MergeCompanyData", func(t *testing.T) {
		// Create new data with additional records
		newData := &CompanyFinancialData{
			Symbol:      "MSTR",
			CompanyName: "MicroStrategy Inc.",
			CIK:         "1050446",
			SharesHistory: []SharesOutstandingRecord{
				{
					Date:            time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
					FilingType:      "10-Q",
					FilingURL:       "https://sec.gov/example4",
					AccessionNumber: "0001234567-24-000002",
					CommonShares:    105000000,
					TotalShares:     105000000,
					ExtractedFrom:   "Balance Sheet",
					ConfidenceScore: 0.95,
				},
			},
			BTCTransactions: []BitcoinTransaction{
				{
					Date:            time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
					FilingType:      "8-K",
					FilingURL:       "https://sec.gov/example5",
					BTCPurchased:    500,
					USDSpent:        30000000,
					AvgPriceUSD:     60000,
					ConfidenceScore: 0.9,
				},
			},
		}

		err := storage.MergeCompanyData("MSTR", newData)
		if err != nil {
			t.Fatalf("Failed to merge company data: %v", err)
		}

		// Load and verify merged data
		mergedData, err := storage.LoadCompanyData("MSTR")
		if err != nil {
			t.Fatalf("Failed to load merged data: %v", err)
		}

		if len(mergedData.SharesHistory) != 3 {
			t.Errorf("Expected 3 shares history records after merge, got %d", len(mergedData.SharesHistory))
		}

		if len(mergedData.BTCTransactions) != 2 {
			t.Errorf("Expected 2 BTC transactions after merge, got %d", len(mergedData.BTCTransactions))
		}

		// Check total BTC holdings after merge
		totalBTC, err := storage.GetTotalBTCHoldings("MSTR")
		if err != nil {
			t.Fatalf("Failed to get total BTC after merge: %v", err)
		}

		if totalBTC != 1500 {
			t.Errorf("Expected 1500 BTC total after merge, got %f", totalBTC)
		}
	})

	// Test ListCompanies
	t.Run("ListCompanies", func(t *testing.T) {
		companies, err := storage.ListCompanies()
		if err != nil {
			t.Fatalf("Failed to list companies: %v", err)
		}

		if len(companies) != 1 {
			t.Errorf("Expected 1 company, got %d", len(companies))
		}

		if companies[0] != "MSTR" {
			t.Errorf("Expected MSTR in companies list, got %s", companies[0])
		}
	})
}
