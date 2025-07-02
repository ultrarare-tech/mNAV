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

	"github.com/ultrarare-tech/mNAV/pkg/shared/models"
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
		return "0001050446", nil
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

// GetCompanyFilings returns recent filings for a given ticker symbol using the official SEC EDGAR API
func (c *Client) GetCompanyFilings(ticker string, filingTypes []string, startDate, endDate string) ([]models.Filing, error) {
	// Get CIK for the ticker
	cik, err := c.GetCIKByTicker(ticker)
	if err != nil {
		return nil, fmt.Errorf("error getting CIK for ticker %s: %w", ticker, err)
	}

	// Build the submissions URL using the official SEC API
	submissionsURL := fmt.Sprintf("https://data.sec.gov/submissions/CIK%s.json", cik)

	// Fetch submissions data
	resp, err := c.Get(submissionsURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching submissions for CIK %s: %w", cik, err)
	}
	defer resp.Body.Close()

	// Parse the submissions response
	var submissionsData SubmissionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&submissionsData); err != nil {
		return nil, fmt.Errorf("error parsing submissions response: %w", err)
	}

	// Parse date filters
	var startTime, endTime time.Time
	if startDate != "" {
		startTime, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date format: %w", err)
		}
	}
	if endDate != "" {
		endTime, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format: %w", err)
		}
		// Set end time to end of day
		endTime = endTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	// Convert filing types to a map for quick lookup
	filingTypeMap := make(map[string]bool)
	for _, ft := range filingTypes {
		filingTypeMap[strings.ToUpper(strings.TrimSpace(ft))] = true
	}

	// Process the filings from the recent submissions
	var filings []models.Filing
	recent := submissionsData.Filings.Recent

	// Iterate through all recent filings
	for i := 0; i < len(recent.AccessionNumber); i++ {
		// Check if this filing type is requested
		formType := recent.Form[i]
		if len(filingTypeMap) > 0 && !filingTypeMap[strings.ToUpper(formType)] {
			continue
		}

		// Parse filing date
		filingDate, err := time.Parse("2006-01-02", recent.FilingDate[i])
		if err != nil {
			continue // Skip invalid dates
		}

		// Check date range
		if !startTime.IsZero() && filingDate.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && filingDate.After(endTime) {
			continue
		}

		// Parse report date (may be empty)
		var reportDate time.Time
		if recent.ReportDate[i] != "" {
			reportDate, _ = time.Parse("2006-01-02", recent.ReportDate[i])
		}

		// Build document URL
		accessionNumber := recent.AccessionNumber[i]
		primaryDocument := recent.PrimaryDocument[i]

		// Format accession number for URL (remove dashes)
		accessionForURL := strings.ReplaceAll(accessionNumber, "-", "")

		// Build the document URL
		documentURL := fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%s/%s/%s",
			strings.TrimLeft(cik, "0"), // Remove leading zeros for URL
			accessionForURL,
			primaryDocument)

		// Build the filing detail URL
		filingURL := fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%s/%s-index.htm",
			strings.TrimLeft(cik, "0"), // Remove leading zeros for URL
			accessionForURL)

		filing := models.Filing{
			AccessionNumber: accessionNumber,
			FilingType:      formType,
			FilingDate:      filingDate,
			ReportDate:      reportDate,
			URL:             filingURL,
			DocumentURL:     documentURL,
		}

		filings = append(filings, filing)
	}

	// Sort filings by date (newest first)
	sort.Slice(filings, func(i, j int) bool {
		return filings[i].FilingDate.After(filings[j].FilingDate)
	})

	return filings, nil
}

// SubmissionsResponse represents the response from the SEC submissions API
type SubmissionsResponse struct {
	CIK        string   `json:"cik"`
	EntityType string   `json:"entityType"`
	Name       string   `json:"name"`
	Tickers    []string `json:"tickers"`
	Exchanges  []string `json:"exchanges"`
	Filings    struct {
		Recent struct {
			AccessionNumber []string `json:"accessionNumber"`
			FilingDate      []string `json:"filingDate"`
			ReportDate      []string `json:"reportDate"`
			Form            []string `json:"form"`
			FileNumber      []string `json:"fileNumber"`
			Items           []string `json:"items"`
			Size            []int    `json:"size"`
			PrimaryDocument []string `json:"primaryDocument"`
		} `json:"recent"`
		Files []struct {
			Name        string `json:"name"`
			FilingCount int    `json:"filingCount"`
			FilingFrom  string `json:"filingFrom"`
			FilingTo    string `json:"filingTo"`
		} `json:"files"`
	} `json:"filings"`
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

// ListDownloadedFilings returns information about all downloaded filings for a ticker
func (c *Client) ListDownloadedFilings(ticker, baseDir string) ([]models.Filing, error) {
	// Get file paths
	filePaths, err := GetDownloadedFilings(ticker, baseDir)
	if err != nil {
		return nil, err
	}

	var filings []models.Filing
	for _, filePath := range filePaths {
		// Parse filename to extract filing information
		filename := filepath.Base(filePath)

		// Expected format: YYYY-MM-DD_FORM-TYPE_ACCESSION-NUMBER.htm
		parts := strings.Split(filename, "_")
		if len(parts) < 3 {
			continue // Skip files that don't match expected format
		}

		// Parse date
		filingDate, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			continue // Skip files with invalid dates
		}

		// Extract form type and accession number
		formType := parts[1]
		accessionWithExt := parts[2]
		accessionNumber := strings.TrimSuffix(accessionWithExt, ".htm")

		// Get file info
		_, err = os.Stat(filePath)
		if err != nil {
			continue
		}

		filing := models.Filing{
			AccessionNumber: accessionNumber,
			FilingType:      formType,
			FilingDate:      filingDate,
			DocumentURL:     filePath, // Use local file path
		}

		filings = append(filings, filing)
	}

	// Sort by date (newest first)
	sort.Slice(filings, func(i, j int) bool {
		return filings[i].FilingDate.After(filings[j].FilingDate)
	})

	return filings, nil
}
