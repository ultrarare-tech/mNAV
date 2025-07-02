package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/shared/models"
)

// Client handles interactions with the Grok API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

// Request represents a request to the Grok API
type Request struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
	Stream   bool      `json:"stream"`
}

// Message represents a message in the Grok conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents the response from Grok API
type Response struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a choice in the Grok response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// BitcoinExtractionResult represents the structured output from Grok for Bitcoin transactions
type BitcoinExtractionResult struct {
	Transactions []GrokBitcoinTransaction `json:"transactions"`
	Analysis     string                   `json:"analysis"`
	Confidence   float64                  `json:"confidence"`
	Reasoning    string                   `json:"reasoning"`
}

// GrokBitcoinTransaction represents a Bitcoin transaction extracted by Grok
type GrokBitcoinTransaction struct {
	BTCAmount       float64 `json:"btc_amount"`
	USDAmount       float64 `json:"usd_amount"`
	PricePerBTC     float64 `json:"price_per_btc"`
	TransactionType string  `json:"transaction_type"` // "purchase", "sale", "holdings_update"
	Date            string  `json:"date"`
	Confidence      float64 `json:"confidence"`
	Reasoning       string  `json:"reasoning"`
	SourceText      string  `json:"source_text"`
}

// SharesExtractionResult represents the structured output from Grok for shares outstanding
type SharesExtractionResult struct {
	SharesData []GrokSharesData `json:"shares_data"`
	Analysis   string           `json:"analysis"`
	Confidence float64          `json:"confidence"`
	Reasoning  string           `json:"reasoning"`
}

// GrokSharesData represents shares outstanding data extracted by Grok
type GrokSharesData struct {
	CommonShares    float64 `json:"common_shares"`
	PreferredShares float64 `json:"preferred_shares"`
	TotalShares     float64 `json:"total_shares"`
	AsOfDate        string  `json:"as_of_date"`
	Source          string  `json:"source"` // "balance_sheet", "cover_page", "notes", etc.
	Confidence      float64 `json:"confidence"`
	Reasoning       string  `json:"reasoning"`
	SourceText      string  `json:"source_text"`
}

// NewClient creates a new Grok API client
func NewClient() *Client {
	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GROK_API_KEY environment variable not set")
	}

	baseURL := os.Getenv("GROK_API_URL")
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1" // Default Grok API endpoint
	}

	model := os.Getenv("GROK_MODEL")
	if model == "" {
		model = "grok-2-1212" // Default model
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for complex filings
		},
	}
}

// IsConfigured returns true if the Grok client is properly configured
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

// ExtractBitcoinTransactions uses Grok to extract Bitcoin transactions from SEC filing text
func (c *Client) ExtractBitcoinTransactions(text string, filing models.Filing) ([]models.BitcoinTransaction, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("Grok API key not configured")
	}

	// Pre-filter content to only Bitcoin-relevant paragraphs to reduce token usage
	filteredText := c.filterBitcoinRelevantContent(text)

	// If no Bitcoin-relevant content found, return empty results
	if filteredText == "" {
		return []models.BitcoinTransaction{}, nil
	}

	// Create the prompt for Bitcoin transaction extraction using filtered content
	prompt := c.createBitcoinExtractionPrompt(filteredText, filing)

	// Make the API request
	response, err := c.makeRequest(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling Grok API: %w", err)
	}

	// Parse the response
	extraction, err := c.parseBitcoinExtractionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("error parsing Grok response: %w", err)
	}

	// Convert to our standard format
	transactions := c.convertBitcoinToStandardFormat(extraction, filing)

	return transactions, nil
}

// ExtractSharesOutstanding uses Grok to extract shares outstanding data from SEC filing text
func (c *Client) ExtractSharesOutstanding(text string, filing models.Filing) (*models.SharesOutstandingRecord, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("Grok API key not configured")
	}

	// Pre-filter content to only shares-relevant sections to reduce token usage
	filteredText := c.filterSharesRelevantContent(text)

	// If no shares-relevant content found, return nil
	if filteredText == "" {
		return nil, nil
	}

	// Create the prompt for shares extraction using filtered content
	prompt := c.createSharesExtractionPrompt(filteredText, filing)

	// Make the API request
	response, err := c.makeRequest(context.Background(), prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling Grok API: %w", err)
	}

	// Parse the response
	extraction, err := c.parseSharesExtractionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("error parsing Grok response: %w", err)
	}

	// Convert to our standard format
	sharesRecord := c.convertSharesToStandardFormat(extraction, filing)

	return sharesRecord, nil
}

// createBitcoinExtractionPrompt creates a detailed prompt for Bitcoin transaction extraction
func (c *Client) createBitcoinExtractionPrompt(text string, filing models.Filing) string {
	prompt := fmt.Sprintf(`You are an expert financial analyst specializing in SEC filing analysis. Your task is to extract Bitcoin transaction information from the following SEC filing text.

FILING CONTEXT:
- Filing Type: %s
- Filing Date: %s
- Company: Public company filing with SEC
- Accession Number: %s

CRITICAL CLASSIFICATION RULES:

1. INDIVIDUAL TRANSACTIONS (EXTRACT THESE):
   - Announced on a SPECIFIC DATE with direct purchase language
   - Pattern: "On [specific date], [company] purchased/acquired [amount] bitcoins"
   - Examples:
     * "On August 11, 2020, MicroStrategy... has purchased 21,454 bitcoins"
     * "On December 4, 2020, MicroStrategy... had purchased approximately 2,574 bitcoins"
     * "On January 22, 2021, MicroStrategy... purchased approximately 314 bitcoins"
   - Key indicators: "On [date]", "announced that it had purchased", "announced that it purchased"

2. CUMULATIVE TOTALS (DO NOT EXTRACT):
   - Covers a DATE RANGE or PERIOD between two dates
   - Pattern: "during [period/quarter/time range between date1 and date2], purchased [amount] bitcoins"
   - Examples:
     * "during the period between July 1, 2021 and August 23, 2021, purchased 3,907 bitcoins"
     * "during the period between November 29, 2021 and December 8, 2021, purchased 1,434 bitcoins"
     * "During the period between November 1, 2022 and December 21, 2022, acquired 2,395 bitcoins"
   - Key indicators: "during the period between", "the period between X and Y", "during the quarter"

3. ADDITIONAL EXCLUSIONS:
   - Holdings updates ("As of [date], the Company holds X bitcoins")
   - Financing activities (bond offerings, loan proceeds, convertible notes)
   - Future intentions ("intends to invest", "will use proceeds", "may purchase")
   - Impairment charges or accounting adjustments

EXTRACTION RULES:
- ONLY extract transactions that follow the "On [specific date]" pattern
- IGNORE any transaction that mentions a date range or period
- For valid transactions, extract: BTC amount, USD amount, price per BTC, specific date
- Set confidence based on clarity of information (0.9 for clear individual transactions)

FILING TEXT:
%s

Please respond with a JSON object in this exact format:
{
  "transactions": [
    {
      "btc_amount": 0.0,
      "usd_amount": 0.0,
      "price_per_btc": 0.0,
      "transaction_type": "purchase|sale|transfer",
      "date": "YYYY-MM-DD",
      "confidence": 0.0,
      "reasoning": "Brief explanation of why this is a valid INDIVIDUAL transaction (not cumulative)",
      "source_text": "Exact relevant excerpt from filing"
    }
  ],
  "analysis": "Overall analysis of Bitcoin-related content, noting any cumulative totals that were excluded",
  "confidence": 0.0,
  "reasoning": "Overall reasoning for the extraction results, explaining individual vs cumulative classification"
}

If no INDIVIDUAL Bitcoin transactions are found (only cumulative totals), return an empty transactions array but still provide analysis explaining what cumulative data was found and excluded.`,
		filing.FilingType,
		filing.FilingDate.Format("2006-01-02"),
		filing.AccessionNumber,
		text)

	return prompt
}

// createSharesExtractionPrompt creates a detailed prompt for shares outstanding extraction
func (c *Client) createSharesExtractionPrompt(text string, filing models.Filing) string {
	prompt := fmt.Sprintf(`You are an expert financial analyst specializing in SEC filing analysis. Your task is to extract shares outstanding information from the following SEC filing text.

FILING CONTEXT:
- Filing Type: %s
- Filing Date: %s
- Company: Public company filing with SEC
- Accession Number: %s

INSTRUCTIONS:
1. Look for shares outstanding data in these sections:
   - Cover page information
   - Consolidated balance sheets
   - Notes to financial statements
   - Stockholders' equity section
   - Capital stock disclosures

2. Extract the most recent and reliable shares outstanding numbers
3. Distinguish between common stock and preferred stock if both are present
4. Look for "as of" dates to determine when the share count is effective
5. Prefer balance sheet data over weighted average calculations
6. Ignore treasury shares unless specifically relevant

For shares data found, extract:
- Common shares outstanding (number of shares)
- Preferred shares outstanding (if any)
- Total shares outstanding
- As of date (when the count is effective)
- Source section (where the data was found)
- Confidence level (0.0-1.0)

FILING TEXT:
%s

Please respond with a JSON object in this exact format:
{
  "shares_data": [
    {
      "common_shares": 0.0,
      "preferred_shares": 0.0,
      "total_shares": 0.0,
      "as_of_date": "YYYY-MM-DD",
      "source": "balance_sheet|cover_page|notes|equity_section",
      "confidence": 0.0,
      "reasoning": "Brief explanation of why this data is reliable",
      "source_text": "Exact relevant excerpt from filing"
    }
  ],
  "analysis": "Overall analysis of shares outstanding information in the filing",
  "confidence": 0.0,
  "reasoning": "Overall reasoning for the extraction results"
}

If no shares outstanding data is found, return an empty shares_data array but still provide analysis.`,
		filing.FilingType,
		filing.FilingDate.Format("2006-01-02"),
		filing.AccessionNumber,
		text)

	return prompt
}

// makeRequest makes a request to the Grok API
func (c *Client) makeRequest(ctx context.Context, prompt string) (*Response, error) {
	request := Request{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Model:  c.model,
		Stream: false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}

// parseBitcoinExtractionResponse parses the Grok response for Bitcoin transactions
func (c *Client) parseBitcoinExtractionResponse(response *Response) (*BitcoinExtractionResult, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content

	// Try to extract JSON from the response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}") + 1

	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonContent := content[jsonStart:jsonEnd]

	var extraction BitcoinExtractionResult
	if err := json.Unmarshal([]byte(jsonContent), &extraction); err != nil {
		return nil, fmt.Errorf("error parsing extraction JSON: %w", err)
	}

	return &extraction, nil
}

// parseSharesExtractionResponse parses the Grok response for shares outstanding
func (c *Client) parseSharesExtractionResponse(response *Response) (*SharesExtractionResult, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content

	// Try to extract JSON from the response
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}") + 1

	if jsonStart == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonContent := content[jsonStart:jsonEnd]

	var extraction SharesExtractionResult
	if err := json.Unmarshal([]byte(jsonContent), &extraction); err != nil {
		return nil, fmt.Errorf("error parsing extraction JSON: %w", err)
	}

	return &extraction, nil
}

// convertBitcoinToStandardFormat converts Grok extraction to standard models
func (c *Client) convertBitcoinToStandardFormat(extraction *BitcoinExtractionResult, filing models.Filing) []models.BitcoinTransaction {
	var transactions []models.BitcoinTransaction

	for _, grokTx := range extraction.Transactions {
		// Parse date
		txDate := filing.FilingDate // Default to filing date
		if grokTx.Date != "" {
			if parsed, err := time.Parse("2006-01-02", grokTx.Date); err == nil {
				txDate = parsed
			}
		}

		// Calculate average price if not provided
		avgPrice := grokTx.PricePerBTC
		if avgPrice == 0 && grokTx.BTCAmount > 0 && grokTx.USDAmount > 0 {
			avgPrice = grokTx.USDAmount / grokTx.BTCAmount
		}

		tx := models.BitcoinTransaction{
			Date:            txDate,
			FilingType:      filing.FilingType,
			FilingURL:       filing.DocumentURL,
			BTCPurchased:    grokTx.BTCAmount,
			USDSpent:        grokTx.USDAmount,
			AvgPriceUSD:     avgPrice,
			ExtractedText:   grokTx.SourceText,
			ConfidenceScore: grokTx.Confidence,
		}

		transactions = append(transactions, tx)
	}

	return transactions
}

// convertSharesToStandardFormat converts Grok shares extraction to standard models
func (c *Client) convertSharesToStandardFormat(extraction *SharesExtractionResult, filing models.Filing) *models.SharesOutstandingRecord {
	if len(extraction.SharesData) == 0 {
		return nil
	}

	// Use the highest confidence shares data
	var bestData *GrokSharesData
	highestConfidence := 0.0

	for i := range extraction.SharesData {
		data := &extraction.SharesData[i]
		if data.Confidence > highestConfidence {
			bestData = data
			highestConfidence = data.Confidence
		}
	}

	if bestData == nil {
		return nil
	}

	// Parse as of date
	asOfDate := filing.FilingDate // Default to filing date
	if bestData.AsOfDate != "" {
		if parsed, err := time.Parse("2006-01-02", bestData.AsOfDate); err == nil {
			asOfDate = parsed
		}
	}

	record := &models.SharesOutstandingRecord{
		Date:            asOfDate,
		FilingType:      filing.FilingType,
		FilingURL:       filing.URL,
		AccessionNumber: filing.AccessionNumber,
		CommonShares:    bestData.CommonShares,
		PreferredShares: bestData.PreferredShares,
		TotalShares:     bestData.TotalShares,
		ExtractedFrom:   "Grok AI: " + bestData.Source,
		ExtractedText:   bestData.SourceText,
		ConfidenceScore: bestData.Confidence,
		Notes:           bestData.Reasoning,
	}

	return record
}

// filterBitcoinRelevantContent extracts only paragraphs that contain Bitcoin-related keywords
func (c *Client) filterBitcoinRelevantContent(text string) string {
	// Bitcoin-related keywords to look for
	bitcoinKeywords := []string{
		"bitcoin", "btc", "cryptocurrency", "crypto", "digital asset", "digital currency",
		"blockchain", "satoshi", "mining", "wallet", "private key", "public key",
	}

	// Split text into paragraphs (by double newlines or single newlines)
	paragraphs := strings.Split(text, "\n")

	var relevantParagraphs []string
	var currentParagraph strings.Builder

	for _, line := range paragraphs {
		line = strings.TrimSpace(line)

		// If empty line, check if current paragraph is relevant
		if line == "" {
			if currentParagraph.Len() > 0 {
				paragraph := currentParagraph.String()
				if c.containsBitcoinKeywords(paragraph, bitcoinKeywords) && len(paragraph) > 50 {
					relevantParagraphs = append(relevantParagraphs, paragraph)
				}
				currentParagraph.Reset()
			}
			continue
		}

		// Add line to current paragraph
		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(line)
	}

	// Check final paragraph
	if currentParagraph.Len() > 0 {
		paragraph := currentParagraph.String()
		if c.containsBitcoinKeywords(paragraph, bitcoinKeywords) && len(paragraph) > 50 {
			relevantParagraphs = append(relevantParagraphs, paragraph)
		}
	}

	// Also look for table rows or list items that might contain Bitcoin info
	tableRows := c.extractBitcoinTableContent(text, bitcoinKeywords)
	relevantParagraphs = append(relevantParagraphs, tableRows...)

	// Join relevant paragraphs with clear separators
	if len(relevantParagraphs) == 0 {
		return ""
	}

	return strings.Join(relevantParagraphs, "\n\n---\n\n")
}

// filterSharesRelevantContent extracts only sections that contain shares outstanding information
func (c *Client) filterSharesRelevantContent(text string) string {
	// Shares-related keywords and section headers
	sharesKeywords := []string{
		"shares outstanding", "common stock", "preferred stock", "stockholders", "equity",
		"balance sheet", "consolidated balance", "capital stock", "share count",
		"weighted average", "basic shares", "diluted shares", "treasury shares",
	}

	sectionHeaders := []string{
		"balance sheet", "consolidated balance sheet", "stockholders equity", "shareholders equity",
		"capital stock", "common stock", "preferred stock", "cover page", "equity section",
		"notes to", "note ", "financial statements", "consolidated statements",
	}

	// Split text into paragraphs
	paragraphs := strings.Split(text, "\n")

	var relevantParagraphs []string
	var currentParagraph strings.Builder
	inRelevantSection := false

	for _, line := range paragraphs {
		line = strings.TrimSpace(line)

		// Check if this line is a section header
		lowerLine := strings.ToLower(line)
		for _, header := range sectionHeaders {
			if strings.Contains(lowerLine, header) {
				inRelevantSection = true
				break
			}
		}

		// If empty line, check if current paragraph is relevant
		if line == "" {
			if currentParagraph.Len() > 0 {
				paragraph := currentParagraph.String()
				if (inRelevantSection || c.containsBitcoinKeywords(paragraph, sharesKeywords)) && len(paragraph) > 30 {
					relevantParagraphs = append(relevantParagraphs, paragraph)
				}
				currentParagraph.Reset()
			}
			inRelevantSection = false // Reset section flag on paragraph break
			continue
		}

		// Add line to current paragraph
		if currentParagraph.Len() > 0 {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(line)
	}

	// Check final paragraph
	if currentParagraph.Len() > 0 {
		paragraph := currentParagraph.String()
		if (inRelevantSection || c.containsBitcoinKeywords(paragraph, sharesKeywords)) && len(paragraph) > 30 {
			relevantParagraphs = append(relevantParagraphs, paragraph)
		}
	}

	// Also extract table content that might contain shares data
	tableRows := c.extractSharesTableContent(text, sharesKeywords)
	relevantParagraphs = append(relevantParagraphs, tableRows...)

	// Join relevant paragraphs
	if len(relevantParagraphs) == 0 {
		return ""
	}

	return strings.Join(relevantParagraphs, "\n\n---\n\n")
}

// containsBitcoinKeywords checks if text contains any of the specified keywords
func (c *Client) containsBitcoinKeywords(text string, keywords []string) bool {
	lowerText := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

// extractBitcoinTableContent looks for table rows or structured data containing Bitcoin keywords
func (c *Client) extractBitcoinTableContent(text string, keywords []string) []string {
	var tableContent []string

	// Look for HTML table rows
	if strings.Contains(text, "<tr>") || strings.Contains(text, "<td>") {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if (strings.Contains(line, "<tr>") || strings.Contains(line, "<td>")) &&
				c.containsBitcoinKeywords(line, keywords) {
				// Clean up HTML tags for better readability
				cleaned := strings.ReplaceAll(line, "<tr>", "")
				cleaned = strings.ReplaceAll(cleaned, "</tr>", "")
				cleaned = strings.ReplaceAll(cleaned, "<td>", " | ")
				cleaned = strings.ReplaceAll(cleaned, "</td>", "")
				cleaned = strings.TrimSpace(cleaned)
				if len(cleaned) > 20 {
					tableContent = append(tableContent, cleaned)
				}
			}
		}
	}

	// Look for structured data patterns (like financial statements)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines with numbers and Bitcoin keywords
		if c.containsBitcoinKeywords(line, keywords) &&
			(strings.Contains(line, "$") || strings.Contains(line, "million") || strings.Contains(line, "billion")) &&
			len(line) > 30 && len(line) < 500 {
			tableContent = append(tableContent, line)
		}
	}

	return tableContent
}

// extractSharesTableContent looks for table rows or structured data containing shares information
func (c *Client) extractSharesTableContent(text string, keywords []string) []string {
	var tableContent []string

	// Look for HTML table rows
	if strings.Contains(text, "<tr>") || strings.Contains(text, "<td>") {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if (strings.Contains(line, "<tr>") || strings.Contains(line, "<td>")) &&
				c.containsBitcoinKeywords(line, keywords) {
				// Clean up HTML tags
				cleaned := strings.ReplaceAll(line, "<tr>", "")
				cleaned = strings.ReplaceAll(cleaned, "</tr>", "")
				cleaned = strings.ReplaceAll(cleaned, "<td>", " | ")
				cleaned = strings.ReplaceAll(cleaned, "</td>", "")
				cleaned = strings.TrimSpace(cleaned)
				if len(cleaned) > 15 {
					tableContent = append(tableContent, cleaned)
				}
			}
		}
	}

	// Look for balance sheet or financial statement lines
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines with share counts
		if c.containsBitcoinKeywords(line, keywords) &&
			(strings.Contains(line, ",") || strings.Contains(line, "shares")) &&
			len(line) > 20 && len(line) < 300 {
			tableContent = append(tableContent, line)
		}
	}

	return tableContent
}
