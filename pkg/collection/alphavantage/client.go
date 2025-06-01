package alphavantage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Client represents an Alpha Vantage API client
type Client struct {
	APIKey  string
	BaseURL string
	client  *http.Client
}

// NewClient creates a new Alpha Vantage client
func NewClient(apiKey string) *Client {
	return &Client{
		APIKey:  apiKey,
		BaseURL: "https://www.alphavantage.co/query",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CompanyOverview represents the company overview response from Alpha Vantage
type CompanyOverview struct {
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
	FiscalYearEnd              string `json:"FiscalYearEnd"`
	LatestQuarter              string `json:"LatestQuarter"`
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
	FiftyTwoWeekHigh           string `json:"52WeekHigh"`
	FiftyTwoWeekLow            string `json:"52WeekLow"`
	FiftyDayMovingAverage      string `json:"50DayMovingAverage"`
	TwoHundredDayMovingAverage string `json:"200DayMovingAverage"`
	SharesOutstanding          string `json:"SharesOutstanding"`
	DividendDate               string `json:"DividendDate"`
	ExDividendDate             string `json:"ExDividendDate"`
}

// ParsedCompanyOverview represents the parsed company overview with numeric values
type ParsedCompanyOverview struct {
	Symbol                     string    `json:"symbol"`
	Name                       string    `json:"name"`
	Description                string    `json:"description"`
	Exchange                   string    `json:"exchange"`
	Currency                   string    `json:"currency"`
	Country                    string    `json:"country"`
	Sector                     string    `json:"sector"`
	Industry                   string    `json:"industry"`
	MarketCapitalization       float64   `json:"market_capitalization"`
	SharesOutstanding          float64   `json:"shares_outstanding"`
	PERatio                    float64   `json:"pe_ratio"`
	BookValue                  float64   `json:"book_value"`
	DividendPerShare           float64   `json:"dividend_per_share"`
	DividendYield              float64   `json:"dividend_yield"`
	EPS                        float64   `json:"eps"`
	Beta                       float64   `json:"beta"`
	FiftyTwoWeekHigh           float64   `json:"52_week_high"`
	FiftyTwoWeekLow            float64   `json:"52_week_low"`
	FiftyDayMovingAverage      float64   `json:"50_day_moving_average"`
	TwoHundredDayMovingAverage float64   `json:"200_day_moving_average"`
	FetchedAt                  time.Time `json:"fetched_at"`
}

// GetCompanyOverview fetches company overview data including shares outstanding
func (c *Client) GetCompanyOverview(symbol string) (*ParsedCompanyOverview, error) {
	url := fmt.Sprintf("%s?function=OVERVIEW&symbol=%s&apikey=%s",
		c.BaseURL, symbol, c.APIKey)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var overview CompanyOverview
	if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	// Check for API errors
	if overview.Symbol == "" {
		return nil, fmt.Errorf("no data found for symbol %s", symbol)
	}

	// Parse the overview data into numeric types
	parsed, err := c.parseOverview(&overview)
	if err != nil {
		return nil, fmt.Errorf("error parsing overview data: %w", err)
	}

	return parsed, nil
}

// GetSharesOutstanding is a convenience method to get just the shares outstanding
func (c *Client) GetSharesOutstanding(symbol string) (float64, error) {
	overview, err := c.GetCompanyOverview(symbol)
	if err != nil {
		return 0, err
	}

	if overview.SharesOutstanding == 0 {
		return 0, fmt.Errorf("shares outstanding data not available for %s", symbol)
	}

	return overview.SharesOutstanding, nil
}

// parseOverview converts string values to appropriate numeric types
func (c *Client) parseOverview(overview *CompanyOverview) (*ParsedCompanyOverview, error) {
	parsed := &ParsedCompanyOverview{
		Symbol:      overview.Symbol,
		Name:        overview.Name,
		Description: overview.Description,
		Exchange:    overview.Exchange,
		Currency:    overview.Currency,
		Country:     overview.Country,
		Sector:      overview.Sector,
		Industry:    overview.Industry,
		FetchedAt:   time.Now(),
	}

	// Parse numeric fields
	if overview.MarketCapitalization != "None" && overview.MarketCapitalization != "" {
		if val, err := strconv.ParseFloat(overview.MarketCapitalization, 64); err == nil {
			parsed.MarketCapitalization = val
		}
	}

	if overview.SharesOutstanding != "None" && overview.SharesOutstanding != "" {
		if val, err := strconv.ParseFloat(overview.SharesOutstanding, 64); err == nil {
			parsed.SharesOutstanding = val
		}
	}

	if overview.PERatio != "None" && overview.PERatio != "" {
		if val, err := strconv.ParseFloat(overview.PERatio, 64); err == nil {
			parsed.PERatio = val
		}
	}

	if overview.BookValue != "None" && overview.BookValue != "" {
		if val, err := strconv.ParseFloat(overview.BookValue, 64); err == nil {
			parsed.BookValue = val
		}
	}

	if overview.DividendPerShare != "None" && overview.DividendPerShare != "" {
		if val, err := strconv.ParseFloat(overview.DividendPerShare, 64); err == nil {
			parsed.DividendPerShare = val
		}
	}

	if overview.DividendYield != "None" && overview.DividendYield != "" {
		if val, err := strconv.ParseFloat(overview.DividendYield, 64); err == nil {
			parsed.DividendYield = val
		}
	}

	if overview.EPS != "None" && overview.EPS != "" {
		if val, err := strconv.ParseFloat(overview.EPS, 64); err == nil {
			parsed.EPS = val
		}
	}

	if overview.Beta != "None" && overview.Beta != "" {
		if val, err := strconv.ParseFloat(overview.Beta, 64); err == nil {
			parsed.Beta = val
		}
	}

	if overview.FiftyTwoWeekHigh != "None" && overview.FiftyTwoWeekHigh != "" {
		if val, err := strconv.ParseFloat(overview.FiftyTwoWeekHigh, 64); err == nil {
			parsed.FiftyTwoWeekHigh = val
		}
	}

	if overview.FiftyTwoWeekLow != "None" && overview.FiftyTwoWeekLow != "" {
		if val, err := strconv.ParseFloat(overview.FiftyTwoWeekLow, 64); err == nil {
			parsed.FiftyTwoWeekLow = val
		}
	}

	if overview.FiftyDayMovingAverage != "None" && overview.FiftyDayMovingAverage != "" {
		if val, err := strconv.ParseFloat(overview.FiftyDayMovingAverage, 64); err == nil {
			parsed.FiftyDayMovingAverage = val
		}
	}

	if overview.TwoHundredDayMovingAverage != "None" && overview.TwoHundredDayMovingAverage != "" {
		if val, err := strconv.ParseFloat(overview.TwoHundredDayMovingAverage, 64); err == nil {
			parsed.TwoHundredDayMovingAverage = val
		}
	}

	return parsed, nil
}
