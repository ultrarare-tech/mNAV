package alphavantage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// SharesOutstandingClient handles Alpha Vantage API calls for shares outstanding data
type SharesOutstandingClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewSharesOutstandingClient creates a new Alpha Vantage client for shares outstanding data
func NewSharesOutstandingClient() *SharesOutstandingClient {
	apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
	if apiKey == "" {
		panic("ALPHA_VANTAGE_API_KEY environment variable is required")
	}

	return &SharesOutstandingClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CompanyOverviewResponse represents Alpha Vantage company overview
type CompanyOverviewResponse struct {
	Symbol                     string `json:"Symbol"`
	AssetType                  string `json:"AssetType"`
	Name                       string `json:"Name"`
	Description                string `json:"Description"`
	CIK                        string `json:"CIK"`
	Exchange                   string `json:"Exchange"`
	Currency                   string `json:"Currency"`
	Country                    string `json:"Country"`
	Sector                     string `json:"Sector"`
	Industry                   string `json:"Industry"`
	Address                    string `json:"Address"`
	SharesOutstanding          string `json:"SharesOutstanding"`
	MarketCapitalization       string `json:"MarketCapitalization"`
	EBITDA                     string `json:"EBITDA"`
	PERatio                    string `json:"PERatio"`
	PEGRatio                   string `json:"PEGRatio"`
	BookValue                  string `json:"BookValue"`
	DividendPerShare           string `json:"DividendPerShare"`
	DividendYield              string `json:"DividendYield"`
	EPS                        string `json:"EPS"`
	RevenuePerShareTTM         string `json:"RevenuePerShareTTM"`
	ProfitMargin               string `json:"ProfitMargin"`
	OperatingMarginTTM         string `json:"OperatingMarginTTM"`
	ReturnOnAssetsTTM          string `json:"ReturnOnAssetsTTM"`
	ReturnOnEquityTTM          string `json:"ReturnOnEquityTTM"`
	RevenueTTM                 string `json:"RevenueTTM"`
	GrossProfitTTM             string `json:"GrossProfitTTM"`
	DilutedEPSTTM              string `json:"DilutedEPSTTM"`
	QuarterlyEarningsGrowthYOY string `json:"QuarterlyEarningsGrowthYOY"`
	QuarterlyRevenueGrowthYOY  string `json:"QuarterlyRevenueGrowthYOY"`
	AnalystTargetPrice         string `json:"AnalystTargetPrice"`
	TrailingPE                 string `json:"TrailingPE"`
	ForwardPE                  string `json:"ForwardPE"`
	PriceToSalesRatioTTM       string `json:"PriceToSalesRatioTTM"`
	PriceToBookRatio           string `json:"PriceToBookRatio"`
	EVToRevenue                string `json:"EVToRevenue"`
	EVToEBITDA                 string `json:"EVToEBITDA"`
	Beta                       string `json:"Beta"`
	Week52High                 string `json:"52WeekHigh"`
	Week52Low                  string `json:"52WeekLow"`
	Day50MovingAverage         string `json:"50DayMovingAverage"`
	Day200MovingAverage        string `json:"200DayMovingAverage"`
}

// BalanceSheetResponse represents Alpha Vantage balance sheet data
type BalanceSheetResponse struct {
	Symbol           string               `json:"symbol"`
	AnnualReports    []BalanceSheetReport `json:"annualReports"`
	QuarterlyReports []BalanceSheetReport `json:"quarterlyReports"`
}

// BalanceSheetReport represents a single balance sheet report
type BalanceSheetReport struct {
	FiscalDateEnding                       string `json:"fiscalDateEnding"`
	ReportedCurrency                       string `json:"reportedCurrency"`
	TotalAssets                            string `json:"totalAssets"`
	TotalCurrentAssets                     string `json:"totalCurrentAssets"`
	CashAndCashEquivalentsAtCarryingValue  string `json:"cashAndCashEquivalentsAtCarryingValue"`
	CashAndShortTermInvestments            string `json:"cashAndShortTermInvestments"`
	Inventory                              string `json:"inventory"`
	CurrentNetReceivables                  string `json:"currentNetReceivables"`
	TotalNonCurrentAssets                  string `json:"totalNonCurrentAssets"`
	PropertyPlantEquipment                 string `json:"propertyPlantEquipment"`
	AccumulatedDepreciationAmortizationPPE string `json:"accumulatedDepreciationAmortizationPPE"`
	IntangibleAssets                       string `json:"intangibleAssets"`
	IntangibleAssetsExcludingGoodwill      string `json:"intangibleAssetsExcludingGoodwill"`
	Goodwill                               string `json:"goodwill"`
	Investments                            string `json:"investments"`
	LongTermInvestments                    string `json:"longTermInvestments"`
	ShortTermInvestments                   string `json:"shortTermInvestments"`
	OtherCurrentAssets                     string `json:"otherCurrentAssets"`
	OtherNonCurrentAssets                  string `json:"otherNonCurrentAssets"`
	TotalLiabilities                       string `json:"totalLiabilities"`
	TotalCurrentLiabilities                string `json:"totalCurrentLiabilities"`
	CurrentAccountsPayable                 string `json:"currentAccountsPayable"`
	DeferredRevenue                        string `json:"deferredRevenue"`
	CurrentDebt                            string `json:"currentDebt"`
	ShortTermDebt                          string `json:"shortTermDebt"`
	TotalNonCurrentLiabilities             string `json:"totalNonCurrentLiabilities"`
	CapitalLeaseObligations                string `json:"capitalLeaseObligations"`
	LongTermDebt                           string `json:"longTermDebt"`
	CurrentLongTermDebt                    string `json:"currentLongTermDebt"`
	LongTermDebtNoncurrent                 string `json:"longTermDebtNoncurrent"`
	ShortLongTermDebtTotal                 string `json:"shortLongTermDebtTotal"`
	OtherCurrentLiabilities                string `json:"otherCurrentLiabilities"`
	OtherNonCurrentLiabilities             string `json:"otherNonCurrentLiabilities"`
	TotalShareholderEquity                 string `json:"totalShareholderEquity"`
	TreasuryStock                          string `json:"treasuryStock"`
	RetainedEarnings                       string `json:"retainedEarnings"`
	CommonStock                            string `json:"commonStock"`
	CommonStockSharesOutstanding           string `json:"commonStockSharesOutstanding"`
}

// HistoricalSharesData represents historical shares outstanding data
type HistoricalSharesData struct {
	Symbol                   string            `json:"symbol"`
	Source                   string            `json:"source"`
	LastUpdated              string            `json:"last_updated"`
	CurrentSharesOutstanding float64           `json:"current_shares_outstanding"`
	HistoricalData           []SharesDataPoint `json:"historical_data"`
}

// SharesDataPoint represents a single shares outstanding data point
type SharesDataPoint struct {
	Date              string  `json:"date"`
	SharesOutstanding float64 `json:"shares_outstanding"`
	Source            string  `json:"source"`
	ReportType        string  `json:"report_type"` // "annual", "quarterly"
}

// GetCompanyOverview fetches current company overview including shares outstanding
func (c *SharesOutstandingClient) GetCompanyOverview(symbol string) (*CompanyOverviewResponse, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=OVERVIEW&symbol=%s&apikey=%s", symbol, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var overview CompanyOverviewResponse
	if err := json.Unmarshal(body, &overview); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &overview, nil
}

// GetBalanceSheet fetches historical balance sheet data including shares outstanding
func (c *SharesOutstandingClient) GetBalanceSheet(symbol string) (*BalanceSheetResponse, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=BALANCE_SHEET&symbol=%s&apikey=%s", symbol, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var balanceSheet BalanceSheetResponse
	if err := json.Unmarshal(body, &balanceSheet); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &balanceSheet, nil
}

// GetHistoricalSharesOutstanding fetches comprehensive historical shares outstanding data
func (c *SharesOutstandingClient) GetHistoricalSharesOutstanding(symbol string) (*HistoricalSharesData, error) {
	// Get current overview
	overview, err := c.GetCompanyOverview(symbol)
	if err != nil {
		return nil, fmt.Errorf("error getting company overview: %w", err)
	}

	// Get historical balance sheet data
	balanceSheet, err := c.GetBalanceSheet(symbol)
	if err != nil {
		return nil, fmt.Errorf("error getting balance sheet: %w", err)
	}

	// Parse current shares outstanding
	currentShares, _ := strconv.ParseFloat(overview.SharesOutstanding, 64)

	historicalData := &HistoricalSharesData{
		Symbol:                   symbol,
		Source:                   "Alpha Vantage Fundamental Data API",
		LastUpdated:              time.Now().Format("2006-01-02T15:04:05Z"),
		CurrentSharesOutstanding: currentShares,
		HistoricalData:           make([]SharesDataPoint, 0),
	}

	// Process annual reports
	for _, report := range balanceSheet.AnnualReports {
		if report.CommonStockSharesOutstanding != "" && report.CommonStockSharesOutstanding != "None" {
			shares, err := strconv.ParseFloat(report.CommonStockSharesOutstanding, 64)
			if err == nil {
				dataPoint := SharesDataPoint{
					Date:              report.FiscalDateEnding,
					SharesOutstanding: shares,
					Source:            "Alpha Vantage Balance Sheet",
					ReportType:        "annual",
				}
				historicalData.HistoricalData = append(historicalData.HistoricalData, dataPoint)
			}
		}
	}

	// Process quarterly reports
	for _, report := range balanceSheet.QuarterlyReports {
		if report.CommonStockSharesOutstanding != "" && report.CommonStockSharesOutstanding != "None" {
			shares, err := strconv.ParseFloat(report.CommonStockSharesOutstanding, 64)
			if err == nil {
				dataPoint := SharesDataPoint{
					Date:              report.FiscalDateEnding,
					SharesOutstanding: shares,
					Source:            "Alpha Vantage Balance Sheet",
					ReportType:        "quarterly",
				}
				historicalData.HistoricalData = append(historicalData.HistoricalData, dataPoint)
			}
		}
	}

	return historicalData, nil
}
