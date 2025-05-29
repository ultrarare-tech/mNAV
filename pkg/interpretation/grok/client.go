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

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
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

	// Create the prompt for Bitcoin transaction extraction
	prompt := c.createBitcoinExtractionPrompt(text, filing)

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

	// Create the prompt for shares extraction
	prompt := c.createSharesExtractionPrompt(text, filing)

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

CRITICAL INSTRUCTIONS:
1. Identify ONLY completed Bitcoin transactions (purchases, sales, or transfers)
2. EXCLUDE financing activities (bond offerings, loan proceeds, convertible notes, ATM offerings)
3. EXCLUDE future intentions or plans ("intends to invest", "will use proceeds", "may purchase")
4. EXCLUDE general holdings statements unless they mention specific new transactions
5. EXCLUDE impairment charges or accounting adjustments
6. Look for specific amounts, dates, and transaction details

For each transaction found, extract:
- BTC amount (number of bitcoins purchased/sold)
- USD amount (total cost/proceeds in dollars)
- Price per BTC (if mentioned or calculable)
- Transaction type (purchase/sale/transfer)
- Confidence level (0.0-1.0 based on clarity of information)
- Brief reasoning for extraction
- Exact source text from the filing

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
      "reasoning": "Brief explanation of why this is a valid transaction",
      "source_text": "Exact relevant excerpt from filing"
    }
  ],
  "analysis": "Overall analysis of Bitcoin-related content in the filing",
  "confidence": 0.0,
  "reasoning": "Overall reasoning for the extraction results"
}

If no Bitcoin transactions are found, return an empty transactions array but still provide analysis.`,
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
