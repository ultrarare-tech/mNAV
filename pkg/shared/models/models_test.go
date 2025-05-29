package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSharesOutstandingRecord(t *testing.T) {
	// Test creating and serializing a SharesOutstandingRecord
	record := SharesOutstandingRecord{
		Date:            time.Now(),
		FilingType:      "10-Q",
		FilingURL:       "https://www.sec.gov/example",
		AccessionNumber: "0001234567-24-000001",
		CommonShares:    100000000,
		PreferredShares: 0,
		TotalShares:     100000000,
		ExtractedFrom:   "Balance Sheet",
		ExtractedText:   "Common stock outstanding: 100,000,000 shares",
		ConfidenceScore: 0.95,
	}

	// Test JSON serialization
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal SharesOutstandingRecord: %v", err)
	}

	// Test JSON deserialization
	var decoded SharesOutstandingRecord
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal SharesOutstandingRecord: %v", err)
	}

	// Verify fields
	if decoded.CommonShares != record.CommonShares {
		t.Errorf("CommonShares mismatch: got %f, want %f", decoded.CommonShares, record.CommonShares)
	}
	if decoded.ConfidenceScore != record.ConfidenceScore {
		t.Errorf("ConfidenceScore mismatch: got %f, want %f", decoded.ConfidenceScore, record.ConfidenceScore)
	}
}

func TestCompanyFinancialData(t *testing.T) {
	// Test creating and serializing CompanyFinancialData
	cfd := CompanyFinancialData{
		Symbol:      "MSTR",
		CompanyName: "MicroStrategy Inc.",
		CIK:         "1050446",
		SharesHistory: []SharesOutstandingRecord{
			{
				Date:            time.Now().AddDate(0, -3, 0),
				FilingType:      "10-Q",
				CommonShares:    95000000,
				TotalShares:     95000000,
				ConfidenceScore: 0.9,
			},
			{
				Date:            time.Now(),
				FilingType:      "10-Q",
				CommonShares:    100000000,
				TotalShares:     100000000,
				ConfidenceScore: 0.95,
			},
		},
		BTCTransactions: []BitcoinTransaction{
			{
				Date:            time.Now().AddDate(0, -1, 0),
				FilingType:      "8-K",
				BTCPurchased:    1000,
				USDSpent:        50000000,
				AvgPriceUSD:     50000,
				ConfidenceScore: 0.9,
			},
		},
		LastUpdated:    time.Now(),
		LastFilingDate: time.Now().AddDate(0, 0, -1),
	}

	// Test JSON serialization
	data, err := json.MarshalIndent(cfd, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal CompanyFinancialData: %v", err)
	}

	// Test JSON deserialization
	var decoded CompanyFinancialData
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal CompanyFinancialData: %v", err)
	}

	// Verify fields
	if decoded.Symbol != cfd.Symbol {
		t.Errorf("Symbol mismatch: got %s, want %s", decoded.Symbol, cfd.Symbol)
	}
	if len(decoded.SharesHistory) != len(cfd.SharesHistory) {
		t.Errorf("SharesHistory length mismatch: got %d, want %d", len(decoded.SharesHistory), len(cfd.SharesHistory))
	}
	if len(decoded.BTCTransactions) != len(cfd.BTCTransactions) {
		t.Errorf("BTCTransactions length mismatch: got %d, want %d", len(decoded.BTCTransactions), len(cfd.BTCTransactions))
	}
}

func TestSharesChangeEvent(t *testing.T) {
	event := SharesChangeEvent{
		Date:            time.Now(),
		PreviousShares:  95000000,
		NewShares:       100000000,
		ChangeAmount:    5000000,
		ChangePercent:   5.26,
		Reason:          "Stock offering",
		FilingReference: "8-K filed on 2024-01-15",
	}

	// Test calculations
	expectedChangePercent := ((event.NewShares - event.PreviousShares) / event.PreviousShares) * 100
	if abs(event.ChangePercent-expectedChangePercent) > 0.01 {
		t.Errorf("ChangePercent calculation mismatch: got %f, want %f", event.ChangePercent, expectedChangePercent)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
