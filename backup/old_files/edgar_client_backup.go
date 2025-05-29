package edgar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

	// SearchURL is the URL for the EDGAR search API
	SearchURL = "https://efts.sec.gov/LATEST/search-index"

	// WebSearchURL is the URL for the web search interface
	WebSearchURL = "https://www.sec.gov/edgar/search/"

	// CompanyTickersURL is the URL for company tickers mapping
	CompanyTickersURL = "https://www.sec.gov/files/company_tickers.json"

	// BrowseEdgarURL is the URL for browsing EDGAR filings
	BrowseEdgarURL = "https://www.sec.gov/cgi-bin/browse-edgar"

	// DefaultUserAgent provides identification for SEC requests
	DefaultUserAgent = "MicroStrategy Bitcoin Tracker 1.0 contact@example.com"

	// RateLimit defines the maximum requests per second to comply with SEC guidelines
	RateLimit = 10
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
		limiter:   rate.NewLimiter(rate.Limit(0.1), 1), // 1 request per 10 seconds to avoid rate limits
	}
}

// Get performs a rate-limited GET request to the specified URL
func (c *Client) Get(url string) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Add a delay to prevent hitting SEC rate limits
	time.Sleep(2 * time.Second) // Increased from 1 second

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

// Post performs a rate-limited POST request to the specified URL
func (c *Client) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json, text/html, */*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
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

// GetCompanyFilings returns recent filings for a given ticker symbol
func (c *Client) GetCompanyFilings(ticker string, filingTypes []string, startDate, endDate string) ([]models.Filing, error) {
	// Get CIK for the ticker
	cik, err := c.GetCIKByTicker(ticker)
	if err != nil {
		return nil, fmt.Errorf("error getting CIK for ticker %s: %w", ticker, err)
	}

	// Get filings using the CIK
	return c.GetFilingsByCIK(cik, filingTypes, startDate, endDate)
}

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

// Filing represents an SEC filing
type Filing struct {
	AccessionNumber string    `json:"accessionNumber"`
	FilingType      string    `json:"filingType"`
	FilingDate      time.Time `json:"filingDate"`
	ReportDate      time.Time `json:"reportDate"`
	URL             string    `json:"url"`
	DocumentURL     string    `json:"documentUrl"`
}

// GetFilingsByCIK retrieves filings for a company by CIK
func (c *Client) GetFilingsByCIK(cik string, filingTypes []string, startDate, endDate string) ([]Filing, error) {
	// Use the web search interface instead of the API
	return c.GetFilingsUsingWebSearch(cik, filingTypes, startDate, endDate)
}

// GetFilingsUsingWebSearch uses the SEC's web search interface to find filings
func (c *Client) GetFilingsUsingWebSearch(cik string, filingTypes []string, startDate, endDate string) ([]Filing, error) {
	allFilings := []Filing{}

	// Format CIK without leading zeros for web search
	cikNum := strings.TrimLeft(cik, "0")

	// Make separate requests for each filing type since SEC EDGAR only accepts one type per request
	for _, filingType := range filingTypes {
		fmt.Printf("Fetching %s filings...\n", filingType)

		// Build base URL for this filing type
		browseURL := fmt.Sprintf("%s?CIK=%s&owner=exclude&action=getcompany&type=%s", BrowseEdgarURL, cikNum, filingType)

		// Add date range if specified
		if startDate != "" && endDate != "" {
			// For SEC EDGAR browser interface:
			// dateb is the "before" date (end date)
			// dated is the "after" date (start date)
			startDateFormatted := formatDateForEdgar(startDate)
			endDateFormatted := formatDateForEdgar(endDate)
			browseURL = fmt.Sprintf("%s&dateb=%s&dated=%s", browseURL, endDateFormatted, startDateFormatted)
		} else if startDate != "" {
			startDateFormatted := formatDateForEdgar(startDate)
			browseURL = fmt.Sprintf("%s&dateb=%s", browseURL, startDateFormatted)
		} else if endDate != "" {
			endDateFormatted := formatDateForEdgar(endDate)
			browseURL = fmt.Sprintf("%s&dateb=%s", browseURL, endDateFormatted)
		}

		// Add count parameter to get more results
		browseURL = fmt.Sprintf("%s&count=100", browseURL)

		// For debugging, save the browse URL
		debugDir := "debug"
		if _, err := os.Stat(debugDir); os.IsNotExist(err) {
			os.Mkdir(debugDir, 0755)
		}
		os.WriteFile(fmt.Sprintf("%s/browse_url_%s_%s.txt", debugDir, cikNum, filingType), []byte(browseURL), 0644)

		// Get the browse page
		resp, err := c.Get(browseURL)
		if err != nil {
			fmt.Printf("Warning: error accessing SEC browse page for %s: %v\n", filingType, err)
			continue
		}

		// Read and save the response for debugging
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Warning: error reading browse response for %s: %v\n", filingType, err)
			continue
		}

		// Save the response for debugging
		os.WriteFile(fmt.Sprintf("%s/edgar_response_%s_%s.html", debugDir, cikNum, filingType), body, 0644)

		// Parse the HTML response
		parsedFilings, err := c.parseEdgarBrowseResponse(body, cikNum)
		if err != nil {
			fmt.Printf("Warning: error parsing browse response for %s: %v\n", filingType, err)
			continue
		}

		fmt.Printf("Found %d %s filings\n", len(parsedFilings), filingType)
		allFilings = append(allFilings, parsedFilings...)

		// Rate limiting between requests
		time.Sleep(1 * time.Second)
	}

	// Remove duplicates (in case a filing appears in multiple searches)
	uniqueFilings := []Filing{}
	seen := make(map[string]bool)

	for _, filing := range allFilings {
		key := fmt.Sprintf("%s_%s_%s", filing.AccessionNumber, filing.FilingType, filing.FilingDate.Format("2006-01-02"))
		if !seen[key] {
			seen[key] = true
			uniqueFilings = append(uniqueFilings, filing)
		}
	}

	// Sort by filing date (newest first)
	sort.Slice(uniqueFilings, func(i, j int) bool {
		return uniqueFilings[i].FilingDate.After(uniqueFilings[j].FilingDate)
	})

	// If no filings found, provide clear instructions
	if len(uniqueFilings) == 0 {
		fmt.Println("No filings found through the SEC EDGAR browser interface.")
		fmt.Printf("You can manually check filings at: %s?CIK=%s&owner=exclude&action=getcompany\n", BrowseEdgarURL, cikNum)
	} else {
		fmt.Printf("Found %d total filings (%d unique) through the SEC EDGAR browser interface.\n", len(allFilings), len(uniqueFilings))
	}

	return uniqueFilings, nil
}

// parseEdgarBrowseResponse parses the HTML response from the EDGAR browse endpoint
func (c *Client) parseEdgarBrowseResponse(body []byte, cik string) ([]Filing, error) {
	filings := []Filing{}

	// Convert to string
	html := string(body)

	// Debug: Write the HTML to a file for inspection
	debugDir := "debug"
	if _, err := os.Stat(debugDir); os.IsNotExist(err) {
		os.Mkdir(debugDir, 0755)
	}

	// Look for filing table
	tableRegex := regexp.MustCompile(`(?s)<table class="tableFile[2]?"[^>]*>(.*?)</table>`)
	tableMatch := tableRegex.FindStringSubmatch(html)

	if len(tableMatch) < 2 {
		// No table found, try alternative pattern
		fmt.Println("No filing table found in the response.")
		return filings, nil
	}

	tableHTML := tableMatch[1]

	// Parse rows in the table
	rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*>.*?href="([^"]+)"[^>]*>&nbsp;Documents</a>.*?</td>\s*<td[^>]*>([^<]+).*?</td>\s*<td>([^<]+)</td>`)
	rowMatches := rowRegex.FindAllStringSubmatch(tableHTML, -1)

	for _, match := range rowMatches {
		if len(match) < 5 {
			continue
		}

		// Extract filing type
		filingType := strings.TrimSpace(match[1])

		// Extract document URL (the index.htm link)
		documentPath := match[2]
		if !strings.HasPrefix(documentPath, "http") {
			documentPath = BaseURL + documentPath
		}

		// Extract filing date
		filingDateStr := strings.TrimSpace(match[4])
		filingDate, err := time.Parse("2006-01-02", filingDateStr)
		if err != nil {
			// Try alternative date format
			filingDate, err = time.Parse("01/02/2006", filingDateStr)
			if err != nil {
				continue // Skip if we can't parse the date
			}
		}

		// Extract accession number from the URL
		accessionNumber := ""
		if matches := regexp.MustCompile(`(\d{10}-\d{2}-\d{6})`).FindStringSubmatch(documentPath); len(matches) > 1 {
			accessionNumber = matches[1]
		} else {
			// Fallback: use filing type and date
			accessionNumber = fmt.Sprintf("%s-%d", filingType, filingDate.Unix())
		}

		// Create the filing object
		filing := Filing{
			AccessionNumber: accessionNumber,
			FilingType:      filingType,
			FilingDate:      filingDate,
			ReportDate:      filingDate, // Use filing date as report date for now
			URL:             documentPath,
			DocumentURL:     documentPath, // Will be updated when we process the index page
		}

		filings = append(filings, filing)
	}

	// If no filings found with the first regex, try a different pattern
	if len(filings) == 0 {
		// Alternative regex pattern for the newer EDGAR interface
		altRowRegex := regexp.MustCompile(`<tr[^>]*>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*>([^<]+)</td>.*?<a[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
		altMatches := altRowRegex.FindAllStringSubmatch(html, -1)

		for _, match := range altMatches {
			if len(match) < 5 {
				continue
			}

			// Extract filing date
			filingDateStr := strings.TrimSpace(match[1])
			filingDate, err := time.Parse("2006-01-02", filingDateStr)
			if err != nil {
				// Try alternative date format
				filingDate, err = time.Parse("01/02/2006", filingDateStr)
				if err != nil {
					continue // Skip if we can't parse the date
				}
			}

			// Extract filing type
			filingType := strings.TrimSpace(match[2])

			// Extract document URL
			documentPath := match[3]
			if !strings.HasPrefix(documentPath, "http") {
				documentPath = BaseURL + documentPath
			}

			// Generate accession number
			accessionNumber := fmt.Sprintf("%s-%d", filingType, filingDate.Unix())

			filing := Filing{
				AccessionNumber: accessionNumber,
				FilingType:      filingType,
				FilingDate:      filingDate,
				ReportDate:      filingDate,
				URL:             documentPath,
				DocumentURL:     documentPath,
			}

			filings = append(filings, filing)
		}
	}

	// Process the index page URLs to find the actual document URLs
	for i, filing := range filings {
		// Get the index page which contains links to the actual documents
		indexResp, err := c.Get(filing.URL)
		if err != nil {
			fmt.Printf("Warning: Error accessing index page for %s: %v\n", filing.AccessionNumber, err)
			continue
		}

		indexBody, err := io.ReadAll(indexResp.Body)
		indexResp.Body.Close()
		if err != nil {
			fmt.Printf("Warning: Error reading index page for %s: %v\n", filing.AccessionNumber, err)
			continue
		}

		// Save the index page for debugging
		os.WriteFile(fmt.Sprintf("%s/index_%s.html", debugDir, filing.AccessionNumber), indexBody, 0644)

		// Look for the primary document in the Document Format Files table
		indexHTML := string(indexBody)

		// Find the Document Format Files table
		docTableRegex := regexp.MustCompile(`(?s)<p>Document Format Files</p>\s*<table class="tableFile"[^>]*>(.*?)</table>`)
		docTableMatch := docTableRegex.FindStringSubmatch(indexHTML)

		if len(docTableMatch) > 1 {
			docTableHTML := docTableMatch[1]

			// Look for the first row with sequence number 1 and matching filing type
			// Pattern: <td scope="row">1</td><td scope="row">FILING_TYPE</td><td scope="row"><a href="...">document.htm</a>
			rowRegex := regexp.MustCompile(`(?s)<tr[^>]*>\s*<td[^>]*>1</td>\s*<td[^>]*>` + regexp.QuoteMeta(filing.FilingType) + `</td>\s*<td[^>]*><a[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
			rowMatch := rowRegex.FindStringSubmatch(docTableHTML)

			if len(rowMatch) >= 2 {
				docPath := rowMatch[1]

				// Check if this is an iXBRL viewer URL (starts with /ix?doc=)
				if strings.HasPrefix(docPath, "/ix?doc=") {
					// Extract the actual document path from the iXBRL viewer URL
					actualDocRegex := regexp.MustCompile(`/ix\?doc=(.+)`)
					actualDocMatch := actualDocRegex.FindStringSubmatch(docPath)
					if len(actualDocMatch) >= 2 {
						docPath = actualDocMatch[1]
					}
				}

				// Construct the full URL
				var docURL string
				if strings.HasPrefix(docPath, "http") {
					docURL = docPath
				} else if strings.HasPrefix(docPath, "/") {
					docURL = fmt.Sprintf("%s%s", BaseURL, docPath)
				} else {
					// Relative path - construct URL based on the index page
					baseDir := filepath.Dir(filing.URL)
					docURL = fmt.Sprintf("%s/%s", baseDir, docPath)
				}

				// Update the document URL
				filings[i].DocumentURL = docURL
				continue
			}
		}

		// Fallback: Look for any .htm/.html document in the Document Format Files table (less specific)
		if len(docTableMatch) > 1 {
			docTableHTML := docTableMatch[1]

			// Look for the first .htm/.html file that's not an exhibit
			primaryDocRegex := regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*>([^<]+\.htm[l]?)</a>`)
			primaryDocMatches := primaryDocRegex.FindAllStringSubmatch(docTableHTML, -1)

			for _, match := range primaryDocMatches {
				if len(match) < 3 {
					continue
				}

				// Check if this is likely the primary document (avoid exhibits and graphics)
				docName := strings.ToLower(match[2])
				if strings.Contains(docName, "exhibit") || strings.Contains(docName, "ex-") ||
					strings.Contains(docName, "graphic") || strings.Contains(docName, ".jpg") ||
					strings.Contains(docName, ".png") || strings.Contains(docName, ".gif") {
					continue // Skip exhibits and graphics
				}

				docPath := match[1]

				// Check if this is an iXBRL viewer URL (starts with /ix?doc=)
				if strings.HasPrefix(docPath, "/ix?doc=") {
					// Extract the actual document path from the iXBRL viewer URL
					actualDocRegex := regexp.MustCompile(`/ix\?doc=(.+)`)
					actualDocMatch := actualDocRegex.FindStringSubmatch(docPath)
					if len(actualDocMatch) >= 2 {
						docPath = actualDocMatch[1]
					}
				}

				// Construct the full URL
				var docURL string
				if strings.HasPrefix(docPath, "http") {
					docURL = docPath
				} else if strings.HasPrefix(docPath, "/") {
					docURL = fmt.Sprintf("%s%s", BaseURL, docPath)
				} else {
					// Relative path - construct URL based on the index page
					baseDir := filepath.Dir(filing.URL)
					docURL = fmt.Sprintf("%s/%s", baseDir, docPath)
				}

				// Update the document URL
				filings[i].DocumentURL = docURL
				break
			}
		}
	}

	return filings, nil
}

// cleanHTML removes HTML tags from a string
func cleanHTML(html string) string {
	// Remove HTML tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text := tagRegex.ReplaceAllString(html, "")

	// Trim whitespace
	text = strings.TrimSpace(text)

	return text
}

// formatDateForEdgar formats a date string in YYYY-MM-DD format to YYYYMMDD format for EDGAR
func formatDateForEdgar(date string) string {
	// The date format for the SEC EDGAR browser interface is different:
	// dateb is the "before" date (end date)
	// dated is the "after" date (start date)
	// Format YYYY-MM-DD needs to be YYYYMMDD

	// Remove hyphens
	return strings.ReplaceAll(date, "-", "")
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
func (c *Client) DownloadFilingContent(filing Filing, baseDir string) (string, error) {
	// Create the directory structure if it doesn't exist
	// Format: data/filings/TICKER/YYYY-MM-DD_TYPE_ACCESSION.htm

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

// DownloadAllFilings downloads all filings for a company to the specified directory
func (c *Client) DownloadAllFilings(ticker string, filingTypes []string, startDate, endDate, baseDir string) ([]string, error) {
	// Get CIK for the ticker
	cik, err := c.GetCIKByTicker(ticker)
	if err != nil {
		return nil, fmt.Errorf("error getting CIK for %s: %w", ticker, err)
	}

	// Get filings
	filings, err := c.GetFilingsByCIK(cik, filingTypes, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error getting filings: %w", err)
	}

	// Create company directory
	companyDir := filepath.Join(baseDir, ticker)
	if err := os.MkdirAll(companyDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating directory for %s: %w", ticker, err)
	}

	// Download each filing
	downloadedFiles := []string{}
	for _, filing := range filings {
		filePath, err := c.DownloadFilingContent(filing, companyDir)
		if err != nil {
			fmt.Printf("Warning: Error downloading filing %s: %v\n", filing.AccessionNumber, err)
			continue
		}

		if filePath != "" {
			downloadedFiles = append(downloadedFiles, filePath)
		}
	}

	return downloadedFiles, nil
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

// AnalyzeDownloadedFilings analyzes all downloaded filings for Bitcoin transactions
func AnalyzeDownloadedFilings(ticker, baseDir string) (*CompanyTransactions, error) {
	// Get list of downloaded filings
	filingPaths, err := GetDownloadedFilings(ticker, baseDir)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d filings for %s\n", len(filingPaths), ticker)

	// Create a document parser
	client := NewClient("")
	parser := NewDocumentParser(client)

	// Create result object
	result := &CompanyTransactions{
		Company:     ticker,
		LastUpdated: time.Now(),
	}

	// Get CIK (try to extract from filenames)
	if len(filingPaths) > 0 {
		filename := filepath.Base(filingPaths[0])
		// Extract accession number
		parts := strings.Split(filename, "_")
		if len(parts) >= 3 {
			accessionParts := strings.Split(parts[2], ".")
			if len(accessionParts) >= 1 {
				accession := accessionParts[0]
				// Extract CIK from accession
				cikRegex := regexp.MustCompile(`(\d{10})-\d{2}-\d{6}`)
				match := cikRegex.FindStringSubmatch(accession)
				if len(match) >= 2 {
					result.CIK = match[1]
				} else {
					// Try to get CIK from ticker
					cik, _ := client.GetCIKByTicker(ticker)
					result.CIK = cik
				}
			}
		}
	}

	// Process each filing
	for _, filePath := range filingPaths {
		// Extract filing information from filename
		filename := filepath.Base(filePath)
		parts := strings.Split(filename, "_")

		if len(parts) < 3 {
			fmt.Printf("Warning: Invalid filename format for %s\n", filename)
			continue // Invalid filename format
		}

		filingDate, err := time.Parse("2006-01-02", parts[0])
		if err != nil {
			fmt.Printf("Warning: Invalid date format in %s\n", filename)
			continue // Invalid date format
		}

		filingType := parts[1]
		accessionNumber := strings.TrimSuffix(parts[2], ".htm")

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Warning: Error reading file %s: %v\n", filePath, err)
			continue
		}

		// Create mock filing for parser
		filing := Filing{
			AccessionNumber: accessionNumber,
			FilingType:      filingType,
			FilingDate:      filingDate,
			ReportDate:      filingDate, // Use filing date as report date
			URL:             filePath,
			DocumentURL:     filePath,
		}

		// Parse for transactions
		var txns []BitcoinTransaction
		if strings.HasSuffix(filePath, ".htm") || strings.HasSuffix(filePath, ".html") {
			txns, err = parser.ParseHTMLDocument(content, filing)
			if err != nil {
				fmt.Printf("Warning: Error parsing HTML file %s: %v\n", filePath, err)
				continue
			}
		} else {
			txns, err = parser.ParseTextDocument(content, filing)
			if err != nil {
				fmt.Printf("Warning: Error parsing text file %s: %v\n", filePath, err)
				continue
			}
		}

		// Debug: Log transactions found
		if len(txns) > 0 {
			fmt.Printf("Found %d Bitcoin transactions in %s\n", len(txns), filename)
			for i, tx := range txns {
				fmt.Printf("  Transaction %d: %.2f BTC at $%.2f per BTC (Total: $%.2f)\n",
					i+1, tx.BTCPurchased, tx.AvgPriceUSD, tx.USDSpent)
			}
		} else {
			fmt.Printf("No Bitcoin transactions found in %s\n", filename)
		}

		// Add transactions to result
		result.Transactions = append(result.Transactions, txns...)
	}

	return result, nil
}
