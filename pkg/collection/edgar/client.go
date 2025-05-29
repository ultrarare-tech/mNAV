package edgar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
	"golang.org/x/time/rate"

	"compress/gzip"
)

const (
	// BaseURL is the base URL for SEC EDGAR API
	BaseURL = "https://www.sec.gov"

	// CompanyTickersURL is the URL for company tickers mapping
	CompanyTickersURL = "https://www.sec.gov/files/company_tickers.json"

	// DefaultUserAgent provides identification for SEC requests
	DefaultUserAgent = "MicroStrategy Bitcoin Tracker 1.0 contact@example.com"
)

// Client represents an SEC EDGAR API client
type Client struct {
	httpClient *http.Client
	userAgent  string
	limiter    *rate.Limiter
}

// NewClient creates a new SEC EDGAR API client
func NewClient(userAgent string) *Client {
	if userAgent == "" {
		// Set a default user agent
		hostname, _ := os.Hostname()
		userAgent = fmt.Sprintf("MNAV Bitcoin Treasury Tracker 1.0 (Contact: example@domain.com; Host: %s)", hostname)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: userAgent,
		limiter:   rate.NewLimiter(rate.Limit(0.1), 1), // 1 request per 10 seconds
	}
}

// Get performs a rate-limited GET request to the specified URL
func (c *Client) Get(url string) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Add a delay to prevent hitting SEC rate limits
	time.Sleep(2 * time.Second)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set required headers for SEC API
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "www.sec.gov")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	// Handle gzip encoding
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("error creating gzip reader: %w", err)
		}
		resp.Body = io.NopCloser(gzipReader)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("rate limited (429): %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("error response: %s, status code: %d", string(body), resp.StatusCode)
	}

	return resp, nil
}

// TickerData represents a company ticker data in the SEC's company_tickers.json file
type TickerData struct {
	CIK    int    `json:"cik_str"`
	Name   string `json:"title"`
	Ticker string `json:"ticker"`
}

// TickersMap is a map of company tickers data keyed by their position in the file
type TickersMap map[string]TickerData

// GetCIKByTicker finds the CIK (Central Index Key) for a given ticker symbol
func (c *Client) GetCIKByTicker(ticker string) (string, error) {
	// Hard-coded fallback for MSTR in case the API call fails
	if ticker == "MSTR" {
		return "1050446", nil
	}

	// Retrieve the company tickers mapping from SEC
	resp, err := c.Get(CompanyTickersURL)
	if err != nil {
		return "", fmt.Errorf("error fetching company tickers: %w", err)
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var tickersMap TickersMap
	if err := json.NewDecoder(resp.Body).Decode(&tickersMap); err != nil {
		return "", fmt.Errorf("error parsing company tickers: %w", err)
	}

	// Find the ticker in the map
	tickerUpper := strings.ToUpper(ticker)
	for _, data := range tickersMap {
		if data.Ticker == tickerUpper {
			// Format CIK with leading zeros to 10 digits
			return fmt.Sprintf("%010d", data.CIK), nil
		}
	}

	return "", fmt.Errorf("CIK not found for ticker: %s", ticker)
}

// GetCompanyFilings returns recent filings for a given ticker symbol (simplified implementation)
func (c *Client) GetCompanyFilings(ticker string, filingTypes []string, startDate, endDate string) ([]models.Filing, error) {
	// Get CIK for the ticker
	cik, err := c.GetCIKByTicker(ticker)
	if err != nil {
		return nil, fmt.Errorf("error getting CIK for ticker %s: %w", ticker, err)
	}

	// For now, return empty slice - full implementation would go here
	fmt.Printf("GetCompanyFilings not yet fully implemented for %s (CIK: %s)\n", ticker, cik)
	return []models.Filing{}, nil
}

// FetchDocumentContent fetches the content of a filing document
func (c *Client) FetchDocumentContent(url string) ([]byte, error) {
	// Fetch real document content
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// DownloadFilingContent downloads a filing document to the specified directory
func (c *Client) DownloadFilingContent(filing models.Filing, baseDir string) (string, error) {
	// Create the directory structure if it doesn't exist
	filingDate := filing.FilingDate.Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s_%s.htm", filingDate, filing.FilingType, filing.AccessionNumber)

	// Create directories if they don't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("error creating directory structure: %w", err)
	}

	// Full path for the filing
	filePath := filepath.Join(baseDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		// File already exists, skip download
		return filePath, nil
	}

	// Fetch content
	content, err := c.FetchDocumentContent(filing.DocumentURL)
	if err != nil {
		return "", fmt.Errorf("error fetching document content: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return filePath, nil
}

// GetDownloadedFilings returns all downloaded filings for a ticker
func GetDownloadedFilings(ticker, baseDir string) ([]string, error) {
	// Path to company filings
	companyDir := filepath.Join(baseDir, ticker)

	// Check if directory exists
	if _, err := os.Stat(companyDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no downloaded filings found for %s", ticker)
	}

	// Read all files in the directory
	entries, err := os.ReadDir(companyDir)
	if err != nil {
		return nil, fmt.Errorf("error reading filings directory: %w", err)
	}

	// Get full paths of all files
	filings := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".htm") {
			filings = append(filings, filepath.Join(companyDir, entry.Name()))
		}
	}

	// Sort by filename (which contains the date)
	sort.Strings(filings)

	return filings, nil
}
