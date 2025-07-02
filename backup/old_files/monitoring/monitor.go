package monitoring

import (
	"fmt"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/shared/storage"
)

// MonitoringService provides monitoring and alerting for EDGAR data
type MonitoringService struct {
	storage *storage.CompanyDataStorage
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(storage *storage.CompanyDataStorage) *MonitoringService {
	return &MonitoringService{
		storage: storage,
	}
}

// DataHealth represents the health status of company data
type DataHealth struct {
	Symbol              string        `json:"symbol"`
	LastUpdated         time.Time     `json:"lastUpdated"`
	LastFilingDate      time.Time     `json:"lastFilingDate"`
	DataAge             time.Duration `json:"dataAge"`
	IsStale             bool          `json:"isStale"`
	SharesDataAvailable bool          `json:"sharesDataAvailable"`
	BTCDataAvailable    bool          `json:"btcDataAvailable"`
	ConfidenceScore     float64       `json:"confidenceScore"`
	Issues              []string      `json:"issues,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Timestamp         time.Time    `json:"timestamp"`
	TotalCompanies    int          `json:"totalCompanies"`
	StaleCompanies    int          `json:"staleCompanies"`
	CompaniesWithData int          `json:"companiesWithData"`
	DataHealth        []DataHealth `json:"dataHealth"`
	SystemIssues      []string     `json:"systemIssues,omitempty"`
}

// CheckDataHealth checks the health of data for a specific company
func (m *MonitoringService) CheckDataHealth(symbol string, maxAge time.Duration) (*DataHealth, error) {
	health := &DataHealth{
		Symbol: symbol,
		Issues: []string{},
	}

	// Load company data
	data, err := m.storage.LoadCompanyData(symbol)
	if err != nil {
		health.Issues = append(health.Issues, fmt.Sprintf("Failed to load data: %v", err))
		return health, nil
	}

	health.LastUpdated = data.LastUpdated
	health.LastFilingDate = data.LastFilingDate
	health.DataAge = time.Since(data.LastUpdated)
	health.IsStale = health.DataAge > maxAge

	// Check shares data
	if len(data.SharesHistory) > 0 {
		health.SharesDataAvailable = true
		// Calculate average confidence
		totalConf := 0.0
		for _, share := range data.SharesHistory {
			totalConf += share.ConfidenceScore
		}
		health.ConfidenceScore = totalConf / float64(len(data.SharesHistory))
	} else {
		health.Issues = append(health.Issues, "No shares outstanding data available")
	}

	// Check BTC data
	if len(data.BTCTransactions) > 0 {
		health.BTCDataAvailable = true
	} else {
		health.Issues = append(health.Issues, "No Bitcoin transaction data available")
	}

	// Check data staleness
	if health.IsStale {
		health.Issues = append(health.Issues, fmt.Sprintf("Data is stale (age: %s)", health.DataAge))
	}

	// Check last filing date
	filingAge := time.Since(health.LastFilingDate)
	if filingAge > 90*24*time.Hour { // 90 days
		health.Issues = append(health.Issues, fmt.Sprintf("No recent filings (last: %s ago)", filingAge))
	}

	// Check low confidence scores
	if health.ConfidenceScore > 0 && health.ConfidenceScore < 0.7 {
		health.Issues = append(health.Issues, fmt.Sprintf("Low confidence score: %.2f", health.ConfidenceScore))
	}

	return health, nil
}

// CheckSystemHealth checks the overall system health
func (m *MonitoringService) CheckSystemHealth(maxAge time.Duration) (*SystemHealth, error) {
	health := &SystemHealth{
		Timestamp:    time.Now(),
		SystemIssues: []string{},
	}

	// Get all companies
	companies, err := m.storage.ListCompanies()
	if err != nil {
		health.SystemIssues = append(health.SystemIssues, fmt.Sprintf("Failed to list companies: %v", err))
		return health, nil
	}

	health.TotalCompanies = len(companies)

	// Check each company
	for _, symbol := range companies {
		companyHealth, err := m.CheckDataHealth(symbol, maxAge)
		if err != nil {
			health.SystemIssues = append(health.SystemIssues, fmt.Sprintf("Error checking %s: %v", symbol, err))
			continue
		}

		health.DataHealth = append(health.DataHealth, *companyHealth)

		if companyHealth.IsStale {
			health.StaleCompanies++
		}

		if companyHealth.SharesDataAvailable || companyHealth.BTCDataAvailable {
			health.CompaniesWithData++
		}
	}

	// Add system-level checks
	if health.StaleCompanies > health.TotalCompanies/2 {
		health.SystemIssues = append(health.SystemIssues, "More than half of companies have stale data")
	}

	if health.CompaniesWithData == 0 {
		health.SystemIssues = append(health.SystemIssues, "No companies have any data")
	}

	return health, nil
}

// Alert represents a monitoring alert
type Alert struct {
	Level     string    `json:"level"` // "warning", "error", "critical"
	Company   string    `json:"company,omitempty"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// GenerateAlerts generates alerts based on system health
func (m *MonitoringService) GenerateAlerts(health *SystemHealth) []Alert {
	alerts := []Alert{}

	// System-level alerts
	for _, issue := range health.SystemIssues {
		alerts = append(alerts, Alert{
			Level:     "critical",
			Message:   "System issue detected",
			Details:   issue,
			Timestamp: time.Now(),
		})
	}

	// Company-level alerts
	for _, companyHealth := range health.DataHealth {
		// Critical: No data at all
		if !companyHealth.SharesDataAvailable && !companyHealth.BTCDataAvailable {
			alerts = append(alerts, Alert{
				Level:     "critical",
				Company:   companyHealth.Symbol,
				Message:   "No data available",
				Details:   "Company has neither shares nor BTC transaction data",
				Timestamp: time.Now(),
			})
		}

		// Error: Stale data
		if companyHealth.IsStale {
			alerts = append(alerts, Alert{
				Level:     "error",
				Company:   companyHealth.Symbol,
				Message:   "Stale data detected",
				Details:   fmt.Sprintf("Data age: %s", companyHealth.DataAge),
				Timestamp: time.Now(),
			})
		}

		// Warning: Low confidence or other issues
		for _, issue := range companyHealth.Issues {
			level := "warning"
			if companyHealth.ConfidenceScore < 0.5 {
				level = "error"
			}

			alerts = append(alerts, Alert{
				Level:     level,
				Company:   companyHealth.Symbol,
				Message:   "Data quality issue",
				Details:   issue,
				Timestamp: time.Now(),
			})
		}
	}

	return alerts
}

// RefreshRecommendation provides recommendations for which companies need updates
type RefreshRecommendation struct {
	Symbol             string        `json:"symbol"`
	Priority           string        `json:"priority"` // "high", "medium", "low"
	Reason             string        `json:"reason"`
	LastUpdated        time.Time     `json:"lastUpdated"`
	DataAge            time.Duration `json:"dataAge"`
	RecommendedActions []string      `json:"recommendedActions"`
}

// GetRefreshRecommendations provides recommendations for which companies need data refresh
func (m *MonitoringService) GetRefreshRecommendations(maxAge time.Duration) ([]RefreshRecommendation, error) {
	recommendations := []RefreshRecommendation{}

	companies, err := m.storage.ListCompanies()
	if err != nil {
		return nil, err
	}

	for _, symbol := range companies {
		health, err := m.CheckDataHealth(symbol, maxAge)
		if err != nil {
			continue
		}

		rec := RefreshRecommendation{
			Symbol:             symbol,
			LastUpdated:        health.LastUpdated,
			DataAge:            health.DataAge,
			RecommendedActions: []string{},
		}

		// Determine priority and recommendations
		if !health.SharesDataAvailable && !health.BTCDataAvailable {
			rec.Priority = "high"
			rec.Reason = "No data available"
			rec.RecommendedActions = append(rec.RecommendedActions, "Fetch all historical filings")
		} else if health.IsStale {
			if health.DataAge > maxAge*2 {
				rec.Priority = "high"
				rec.Reason = "Very stale data"
			} else {
				rec.Priority = "medium"
				rec.Reason = "Stale data"
			}
			rec.RecommendedActions = append(rec.RecommendedActions, "Fetch recent filings")
		} else if health.ConfidenceScore < 0.7 {
			rec.Priority = "low"
			rec.Reason = "Low confidence data"
			rec.RecommendedActions = append(rec.RecommendedActions, "Re-process existing filings with improved parsers")
		} else {
			continue // No recommendation needed
		}

		// Add specific filing type recommendations
		if !health.SharesDataAvailable {
			rec.RecommendedActions = append(rec.RecommendedActions, "Focus on 10-Q and 10-K filings for shares data")
		}
		if !health.BTCDataAvailable && (symbol == "MSTR" || symbol == "SMLR" || symbol == "MARA") {
			rec.RecommendedActions = append(rec.RecommendedActions, "Focus on 8-K filings for Bitcoin transactions")
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}
