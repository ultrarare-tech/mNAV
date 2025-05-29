package edgar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GrokClient handles interactions with the Grok API
type GrokClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// GrokRequest represents a request to the Grok API
type GrokRequest struct {
	Messages []GrokMessage `json:"messages"`
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
}

// GrokMessage represents a message in the Grok conversation
type GrokMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GrokResponse represents the response from Grok API
type GrokResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []GrokChoice `json:"choices"`
	Usage   GrokUsage    `json:"usage"`
}

// GrokChoice represents a choice in the Grok response
type GrokChoice struct {
	Index        int         `json:"index"`
	Message      GrokMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// GrokUsage represents token usage information
type GrokUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// BitcoinTransactionExtraction represents the structured output from Grok
type BitcoinTransactionExtraction struct {
	Transactions []GrokBitcoinTransaction `json:"transactions"`
	Analysis     string                   `json:"analysis"`
	Confidence   float64                  `json:"confidence"`
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

// NewGrokClient creates a new Grok API client
func NewGrokClient() *GrokClient {
	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: GROK_API_KEY environment variable not set")
	}

	return &GrokClient{
		apiKey:  apiKey,
		baseURL: "https://api.x.ai/v1", // Grok API endpoint
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ExtractBitcoinTransactions uses Grok to extract Bitcoin transactions from SEC filing text
func (g *GrokClient) ExtractBitcoinTransactions(text string, filing Filing) ([]BitcoinTransaction, error) {
	if g.apiKey == "" {
		return nil, fmt.Errorf("Grok API key not configured")
	}

	// Create the prompt for Grok
	prompt := g.createExtractionPrompt(text, filing)

	// Make the API request
	response, err := g.makeRequest(prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling Grok API: %w", err)
	}

	// Parse the response
	extraction, err := g.parseExtractionResponse(response)
	if err != nil {
		return nil, fmt.Errorf("error parsing Grok response: %w", err)
	}

	// Convert to our standard format
	transactions := g.convertToStandardFormat(extraction, filing)

	return transactions, nil
}

// createExtractionPrompt creates a detailed prompt for Bitcoin transaction extraction
func (g *GrokClient) createExtractionPrompt(text string, filing Filing) string {
	prompt := fmt.Sprintf(`You are an expert financial analyst specializing in SEC filing analysis. Your task is to extract Bitcoin transaction information from the following SEC filing text.

FILING CONTEXT:
- Filing Type: %s
- Filing Date: %s
- Company: Public company filing with SEC

INSTRUCTIONS:
1. Identify ONLY completed Bitcoin transactions (purchases, sales, or transfers)
2. EXCLUDE financing activities (bond offerings, loan proceeds, convertible notes)
3. EXCLUDE future intentions or plans ("intends to invest", "will use proceeds")
4. EXCLUDE general holdings statements unless they mention specific new transactions

For each transaction found, extract:
- BTC amount (number of bitcoins)
- USD amount (total cost/proceeds)
- Price per BTC (if mentioned)
- Transaction type (purchase/sale/transfer)
- Confidence level (0.0-1.0)
- Brief reasoning for extraction

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
      "reasoning": "Brief explanation",
      "source_text": "Relevant excerpt from filing"
    }
  ],
  "analysis": "Overall analysis of the filing",
  "confidence": 0.0
}

If no Bitcoin transactions are found, return an empty transactions array.`,
		filing.FilingType,
		filing.FilingDate.Format("2006-01-02"),
		text)

	return prompt
}

// makeRequest makes a request to the Grok API
func (g *GrokClient) makeRequest(prompt string) (*GrokResponse, error) {
	// Get model from environment variable, with fallback to default
	model := os.Getenv("GROK_MODEL")
	if model == "" {
		model = "grok-2-1212" // Default fallback
	}

	request := GrokRequest{
		Messages: []GrokMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Model:  model,
		Stream: false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", g.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var grokResponse GrokResponse
	if err := json.NewDecoder(resp.Body).Decode(&grokResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &grokResponse, nil
}

// parseExtractionResponse parses the Grok API response to extract Bitcoin transaction data
func (g *GrokClient) parseExtractionResponse(response *GrokResponse) (*BitcoinTransactionExtraction, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in Grok response")
	}

	content := response.Choices[0].Message.Content

	// Try to extract JSON from the response
	// Grok might wrap the JSON in markdown code blocks
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in Grok response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var extraction BitcoinTransactionExtraction
	if err := json.Unmarshal([]byte(jsonStr), &extraction); err != nil {
		return nil, fmt.Errorf("error parsing JSON from Grok response: %w", err)
	}

	return &extraction, nil
}

// convertToStandardFormat converts Grok extraction to our standard BitcoinTransaction format
func (g *GrokClient) convertToStandardFormat(extraction *BitcoinTransactionExtraction, filing Filing) []BitcoinTransaction {
	var transactions []BitcoinTransaction

	for _, grokTx := range extraction.Transactions {
		// Parse date
		date := filing.FilingDate // Default to filing date
		if grokTx.Date != "" {
			if parsedDate, err := time.Parse("2006-01-02", grokTx.Date); err == nil {
				date = parsedDate
			}
		}

		// Only include purchases for now (can be extended for sales/transfers)
		if grokTx.TransactionType == "purchase" && grokTx.BTCAmount > 0 && grokTx.USDAmount > 0 {
			tx := BitcoinTransaction{
				Date:            date,
				FilingType:      filing.FilingType,
				FilingURL:       filing.URL,
				BTCPurchased:    grokTx.BTCAmount,
				USDSpent:        grokTx.USDAmount,
				AvgPriceUSD:     grokTx.PricePerBTC,
				ExtractedText:   grokTx.SourceText,
				ConfidenceScore: grokTx.Confidence,
			}

			// Calculate price if not provided
			if tx.AvgPriceUSD == 0 && tx.BTCPurchased > 0 && tx.USDSpent > 0 {
				tx.AvgPriceUSD = tx.USDSpent / tx.BTCPurchased
			}

			transactions = append(transactions, tx)
		}
	}

	return transactions
}

// EnhancedDocumentParser combines regex and Grok parsing for maximum accuracy
type EnhancedDocumentParser struct {
	regexParser *DocumentParser
	grokClient  *GrokClient
	useGrok     bool
}

// NewEnhancedDocumentParser creates a new enhanced parser with both regex and Grok capabilities
func NewEnhancedDocumentParser(client *Client) *EnhancedDocumentParser {
	return &EnhancedDocumentParser{
		regexParser: NewDocumentParser(client),
		grokClient:  NewGrokClient(),
		useGrok:     os.Getenv("GROK_API_KEY") != "",
	}
}

// ParseHTMLDocumentEnhanced parses HTML documents using both regex and Grok for maximum accuracy
func (p *EnhancedDocumentParser) ParseHTMLDocumentEnhanced(body []byte, filing Filing) ([]BitcoinTransaction, error) {
	var allTransactions []BitcoinTransaction
	seenTransactions := make(map[string]bool)

	// First, try regex parsing (fast, reliable for known patterns)
	regexTransactions, err := p.regexParser.ParseHTMLDocument(body, filing)
	if err != nil {
		fmt.Printf("Warning: Regex parsing failed: %v\n", err)
	} else {
		for _, tx := range regexTransactions {
			key := fmt.Sprintf("%.2f_%.2f", tx.BTCPurchased, tx.USDSpent)
			if !seenTransactions[key] {
				seenTransactions[key] = true
				allTransactions = append(allTransactions, tx)
			}
		}
	}

	// If Grok is available and we found few/no transactions, try Grok parsing
	if p.useGrok && len(allTransactions) == 0 {
		fmt.Println("Using Grok API for enhanced extraction...")

		// Convert HTML to text for Grok
		text := string(body)
		// Simple HTML tag removal for Grok processing
		text = strings.ReplaceAll(text, "<", " <")
		text = strings.ReplaceAll(text, ">", "> ")

		grokTransactions, err := p.grokClient.ExtractBitcoinTransactions(text, filing)
		if err != nil {
			fmt.Printf("Warning: Grok parsing failed: %v\n", err)
		} else {
			for _, tx := range grokTransactions {
				key := fmt.Sprintf("%.2f_%.2f", tx.BTCPurchased, tx.USDSpent)
				if !seenTransactions[key] {
					seenTransactions[key] = true
					// Mark as Grok-extracted for identification
					tx.ExtractedText = "[GROK] " + tx.ExtractedText
					allTransactions = append(allTransactions, tx)
				}
			}
		}
	}

	return allTransactions, nil
}
