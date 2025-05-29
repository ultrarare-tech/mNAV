package yahoo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Create a rate limiter for Yahoo Finance API (2 requests per second to be conservative)
var infoRateLimiter = rate.NewLimiter(rate.Every(500*time.Millisecond), 1)

// StockInfo represents fundamental stock information from Yahoo Finance
type StockInfo struct {
	Symbol                     string    `json:"symbol"`
	ShortName                  string    `json:"shortName"`
	LongName                   string    `json:"longName"`
	Currency                   string    `json:"currency"`
	MarketCap                  float64   `json:"marketCap"`
	SharesOutstanding          float64   `json:"sharesOutstanding"`
	FloatShares                float64   `json:"floatShares"`
	ImpliedSharesOutstanding   float64   `json:"impliedSharesOutstanding"`
	EnterpriseValue            float64   `json:"enterpriseValue"`
	PriceToBook                float64   `json:"priceToBook"`
	Beta                       float64   `json:"beta"`
	TrailingPE                 float64   `json:"trailingPE"`
	ForwardPE                  float64   `json:"forwardPE"`
	Volume                     int64     `json:"volume"`
	AverageVolume              int64     `json:"averageVolume"`
	AverageVolume10days        int64     `json:"averageVolume10days"`
	RegularMarketPrice         float64   `json:"regularMarketPrice"`
	RegularMarketPreviousClose float64   `json:"regularMarketPreviousClose"`
	FiftyDayAverage            float64   `json:"fiftyDayAverage"`
	TwoHundredDayAverage       float64   `json:"twoHundredDayAverage"`
	LastUpdated                time.Time `json:"lastUpdated"`
}

// GetStockInfo fetches comprehensive stock information using a hybrid approach
func GetStockInfo(symbol string) (*StockInfo, error) {
	// Initialize the info struct
	info := &StockInfo{
		Symbol:      symbol,
		LastUpdated: time.Now(),
	}

	// First, get basic price info from the existing working method
	stockPrice, err := GetStockPrice(symbol)
	if err == nil {
		info.RegularMarketPrice = stockPrice.Price
		info.MarketCap = stockPrice.MarketCap
		info.SharesOutstanding = stockPrice.OutstandingShares
		info.Volume = stockPrice.Volume
	}

	// Try to enhance with web scraping
	enhanceInfoFromWeb(info, symbol)

	// If we still don't have critical data, try the API as a last resort
	if info.SharesOutstanding == 0 || info.MarketCap == 0 {
		apiInfo, err := getStockInfoFromAPI(symbol)
		if err == nil {
			// Merge API data, preferring non-zero values
			if info.SharesOutstanding == 0 && apiInfo.SharesOutstanding > 0 {
				info.SharesOutstanding = apiInfo.SharesOutstanding
			}
			if info.MarketCap == 0 && apiInfo.MarketCap > 0 {
				info.MarketCap = apiInfo.MarketCap
			}
			if info.FloatShares == 0 && apiInfo.FloatShares > 0 {
				info.FloatShares = apiInfo.FloatShares
			}
			if info.Beta == 0 && apiInfo.Beta != 0 {
				info.Beta = apiInfo.Beta
			}
			if info.TrailingPE == 0 && apiInfo.TrailingPE > 0 {
				info.TrailingPE = apiInfo.TrailingPE
			}
			if info.ForwardPE == 0 && apiInfo.ForwardPE > 0 {
				info.ForwardPE = apiInfo.ForwardPE
			}
		}
	}

	// Calculate implied shares if we have market cap but not shares
	if info.SharesOutstanding == 0 && info.MarketCap > 0 && info.RegularMarketPrice > 0 {
		info.ImpliedSharesOutstanding = info.MarketCap / info.RegularMarketPrice
	}

	return info, nil
}

// enhanceInfoFromWeb scrapes additional data from Yahoo Finance website
func enhanceInfoFromWeb(info *StockInfo, symbol string) error {
	// Rate limit the request
	ctx := context.Background()
	if err := infoRateLimiter.Wait(ctx); err != nil {
		return err
	}

	// Try key statistics page
	url := fmt.Sprintf("https://finance.yahoo.com/quote/%s/key-statistics", symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Set browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyStr := string(body)

	// Extract various metrics using regex patterns
	patterns := map[string]*regexp.Regexp{
		"beta":             regexp.MustCompile(`Beta \(5Y Monthly\)[^<]*<[^>]+>([^<]+)<`),
		"pe_trailing":      regexp.MustCompile(`Trailing P/E[^<]*<[^>]+>([^<]+)<`),
		"pe_forward":       regexp.MustCompile(`Forward P/E[^<]*<[^>]+>([^<]+)<`),
		"float":            regexp.MustCompile(`Float[^<]*<[^>]+>([^<]+)<`),
		"avg_volume":       regexp.MustCompile(`Avg\. Volume[^<]*<[^>]+>([^<]+)<`),
		"enterprise_value": regexp.MustCompile(`Enterprise Value[^<]*<[^>]+>([^<]+)<`),
		"price_book":       regexp.MustCompile(`Price/Book[^<]*<[^>]+>([^<]+)<`),
	}

	// Extract beta
	if matches := patterns["beta"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if beta, err := strconv.ParseFloat(strings.TrimSpace(matches[1]), 64); err == nil {
			info.Beta = beta
		}
	}

	// Extract trailing PE
	if matches := patterns["pe_trailing"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if pe, err := strconv.ParseFloat(strings.TrimSpace(matches[1]), 64); err == nil {
			info.TrailingPE = pe
		}
	}

	// Extract forward PE
	if matches := patterns["pe_forward"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if pe, err := strconv.ParseFloat(strings.TrimSpace(matches[1]), 64); err == nil {
			info.ForwardPE = pe
		}
	}

	// Extract float shares
	if matches := patterns["float"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if float, err := ParseMarketCap(strings.TrimSpace(matches[1])); err == nil {
			info.FloatShares = float
		}
	}

	// Extract average volume
	if matches := patterns["avg_volume"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if vol, err := ParseMarketCap(strings.TrimSpace(matches[1])); err == nil {
			info.AverageVolume = int64(vol)
		}
	}

	// Extract enterprise value
	if matches := patterns["enterprise_value"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if ev, err := ParseMarketCap(strings.TrimSpace(matches[1])); err == nil {
			info.EnterpriseValue = ev
		}
	}

	// Extract price to book
	if matches := patterns["price_book"].FindStringSubmatch(bodyStr); len(matches) > 1 {
		if pb, err := strconv.ParseFloat(strings.TrimSpace(matches[1]), 64); err == nil {
			info.PriceToBook = pb
		}
	}

	// Try to get company name from the page
	namePattern := regexp.MustCompile(`<h1[^>]*>([^<]+)\s*\(` + regexp.QuoteMeta(symbol) + `\)`)
	if matches := namePattern.FindStringSubmatch(bodyStr); len(matches) > 1 {
		info.LongName = strings.TrimSpace(matches[1])
		info.ShortName = strings.TrimSpace(matches[1])
	}

	return nil
}

// GetSharesOutstanding fetches only the shares outstanding for a symbol
func GetSharesOutstanding(symbol string) (float64, error) {
	info, err := GetStockInfo(symbol)
	if err != nil {
		return 0, err
	}

	// Return shares outstanding, or implied shares if not available
	if info.SharesOutstanding > 0 {
		return info.SharesOutstanding, nil
	}

	if info.ImpliedSharesOutstanding > 0 {
		return info.ImpliedSharesOutstanding, nil
	}

	// If neither is available, try to calculate from market cap and price
	if info.MarketCap > 0 && info.RegularMarketPrice > 0 {
		return info.MarketCap / info.RegularMarketPrice, nil
	}

	return 0, fmt.Errorf("shares outstanding data not available for %s", symbol)
}

// GetMultipleStockInfo fetches stock info for multiple symbols efficiently
func GetMultipleStockInfo(symbols []string) (map[string]*StockInfo, error) {
	results := make(map[string]*StockInfo)
	errors := make(map[string]error)

	for _, symbol := range symbols {
		info, err := GetStockInfo(symbol)
		if err != nil {
			errors[symbol] = err
			continue
		}
		results[symbol] = info

		// Add a small delay between requests to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	// If all requests failed, return an error
	if len(errors) == len(symbols) {
		return nil, fmt.Errorf("failed to fetch info for all symbols")
	}

	return results, nil
}

// getStockInfoFromAPI attempts to get stock info from Yahoo Finance API
// This is used as a fallback when web scraping fails
func getStockInfoFromAPI(symbol string) (*StockInfo, error) {
	// For now, just return an error since the API requires authentication
	// In the future, this could be implemented with proper API credentials
	return nil, fmt.Errorf("API not available")
}
