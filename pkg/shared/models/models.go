package models

import (
	"time"
)

// SharesOutstandingRecord represents a record of shares outstanding from an SEC filing
type SharesOutstandingRecord struct {
	Date            time.Time `json:"date"`
	FilingType      string    `json:"filingType"`
	FilingURL       string    `json:"filingUrl"`
	AccessionNumber string    `json:"accessionNumber"`
	CommonShares    float64   `json:"commonShares"`
	PreferredShares float64   `json:"preferredShares,omitempty"`
	TotalShares     float64   `json:"totalShares"`
	ExtractedFrom   string    `json:"extractedFrom"`   // Section of filing where data was found
	ExtractedText   string    `json:"extractedText"`   // Raw text that was parsed
	ConfidenceScore float64   `json:"confidenceScore"` // 0.0 to 1.0
	Notes           string    `json:"notes,omitempty"` // Any additional notes
}

// CompanyFinancialData represents comprehensive financial data for a company from SEC filings
type CompanyFinancialData struct {
	Symbol            string                    `json:"symbol"`
	CompanyName       string                    `json:"companyName"`
	CIK               string                    `json:"cik"`
	SharesHistory     []SharesOutstandingRecord `json:"sharesHistory"`
	BTCTransactions   []BitcoinTransaction      `json:"btcTransactions"`
	LastUpdated       time.Time                 `json:"lastUpdated"`
	LastFilingDate    time.Time                 `json:"lastFilingDate"`
	LastProcessedDate time.Time                 `json:"lastProcessedDate"`
}

// FilingMetadata represents metadata about a processed filing
type FilingMetadata struct {
	AccessionNumber string    `json:"accessionNumber"`
	FilingType      string    `json:"filingType"`
	FilingDate      time.Time `json:"filingDate"`
	ProcessedDate   time.Time `json:"processedDate"`
	SharesExtracted bool      `json:"sharesExtracted"`
	BTCExtracted    bool      `json:"btcExtracted"`
	Errors          []string  `json:"errors,omitempty"`
}

// SharesChangeEvent represents a change in shares outstanding
type SharesChangeEvent struct {
	Date            time.Time `json:"date"`
	PreviousShares  float64   `json:"previousShares"`
	NewShares       float64   `json:"newShares"`
	ChangeAmount    float64   `json:"changeAmount"`
	ChangePercent   float64   `json:"changePercent"`
	Reason          string    `json:"reason,omitempty"` // e.g., "Stock offering", "Stock split", "Buyback"
	FilingReference string    `json:"filingReference"`
}

// ExtractedFinancialData represents all financial data extracted from a single filing
type ExtractedFinancialData struct {
	Filing            Filing                   `json:"filing"`
	SharesOutstanding *SharesOutstandingRecord `json:"sharesOutstanding,omitempty"`
	BTCTransactions   []BitcoinTransaction     `json:"btcTransactions,omitempty"`
	ProcessingErrors  []string                 `json:"processingErrors,omitempty"`
	ProcessedAt       time.Time                `json:"processedAt"`
}

// DataSource represents where a piece of data came from
type DataSource struct {
	Type       string    `json:"type"` // "SEC", "Yahoo", "Manual", etc.
	URL        string    `json:"url,omitempty"`
	Date       time.Time `json:"date"`
	Confidence float64   `json:"confidence"`
}

// AuditableSharesData represents shares data with full audit trail
type AuditableSharesData struct {
	Value       float64    `json:"value"`
	AsOfDate    time.Time  `json:"asOfDate"`
	Source      DataSource `json:"source"`
	LastChecked time.Time  `json:"lastChecked"`
}

// CompanyDataSnapshot represents a point-in-time snapshot of all company data
type CompanyDataSnapshot struct {
	Symbol            string              `json:"symbol"`
	SnapshotDate      time.Time           `json:"snapshotDate"`
	SharesOutstanding AuditableSharesData `json:"sharesOutstanding"`
	BTCHoldings       float64             `json:"btcHoldings"`
	MarketCap         float64             `json:"marketCap,omitempty"`
	StockPrice        float64             `json:"stockPrice,omitempty"`
}

// RawFilingDocument represents a raw SEC filing document
type RawFilingDocument struct {
	AccessionNumber string    `json:"accessionNumber"`
	FilingType      string    `json:"filingType"`
	FilingDate      time.Time `json:"filingDate"`
	CompanySymbol   string    `json:"companySymbol"`
	CompanyCIK      string    `json:"companyCik"`
	DocumentURL     string    `json:"documentUrl"`
	ContentType     string    `json:"contentType"` // "text/html", "text/plain", etc.
	ContentLength   int64     `json:"contentLength"`
	DownloadedAt    time.Time `json:"downloadedAt"`
	ProcessedAt     time.Time `json:"processedAt,omitempty"`
	ProcessingNotes string    `json:"processingNotes,omitempty"`
	Checksum        string    `json:"checksum"` // SHA256 hash for integrity
}

// FilingProcessingResult represents the result of processing a raw filing
type FilingProcessingResult struct {
	Document         RawFilingDocument       `json:"document"`
	ExtractedData    *ExtractedFinancialData `json:"extractedData,omitempty"`
	ProcessingErrors []string                `json:"processingErrors,omitempty"`
	ProcessedAt      time.Time               `json:"processedAt"`
}

// Filing represents an SEC filing document
type Filing struct {
	AccessionNumber string    `json:"accessionNumber"`
	FilingType      string    `json:"filingType"`
	FilingDate      time.Time `json:"filingDate"`
	ReportDate      time.Time `json:"reportDate"`
	URL             string    `json:"url"`
	DocumentURL     string    `json:"documentUrl"`
}

// BitcoinTransaction represents a Bitcoin transaction found in an SEC filing
type BitcoinTransaction struct {
	Date            time.Time `json:"date"`
	FilingType      string    `json:"filingType"`
	FilingURL       string    `json:"filingUrl"`
	BTCPurchased    float64   `json:"btcPurchased"`
	USDSpent        float64   `json:"usdSpent"`
	AvgPriceUSD     float64   `json:"avgPriceUsd"`
	TotalBTCAfter   float64   `json:"totalBtcAfter,omitempty"`
	ExtractedText   string    `json:"extractedText"`
	ConfidenceScore float64   `json:"confidenceScore"`
}

// FilingParseResult represents the result of parsing a filing with enhanced parser
type FilingParseResult struct {
	Filing              Filing                   `json:"filing"`
	BitcoinTransactions []BitcoinTransaction     `json:"bitcoinTransactions"`
	SharesOutstanding   *SharesOutstandingRecord `json:"sharesOutstanding,omitempty"`
	ParsedAt            time.Time                `json:"parsedAt"`
	ParsingMethod       string                   `json:"parsingMethod"`
	ProcessingTimeMs    int                      `json:"processingTimeMs"`
	Errors              []string                 `json:"errors,omitempty"`
}
