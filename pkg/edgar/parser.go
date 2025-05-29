package edgar

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// BitcoinTransaction represents a Bitcoin transaction found in an SEC filing
type BitcoinTransaction struct {
	Date            time.Time `json:"date"`
	FilingType      string    `json:"filingType"`
	FilingURL       string    `json:"filingUrl"`
	BTCPurchased    float64   `json:"btcPurchased"`
	USDSpent        float64   `json:"usdSpent"`
	AvgPriceUSD     float64   `json:"avgPriceUsd"`
	TotalBTCAfter   float64   `json:"totalBtcAfter,omitempty"`
	ExtractedText   string    `json:"extractedText"`
	ConfidenceScore float64   `json:"confidenceScore"`
}

// CompanyTransactions holds all Bitcoin transactions for a company
type CompanyTransactions struct {
	Company      string               `json:"company"`
	CIK          string               `json:"cik"`
	Transactions []BitcoinTransaction `json:"transactions"`
	LastUpdated  time.Time            `json:"lastUpdated"`
}

// DocumentParser parses SEC EDGAR documents for Bitcoin transactions and shares outstanding
type DocumentParser struct {
	client       *Client
	sharesParser *SharesParser
}

// NewDocumentParser creates a new EDGAR document parser
func NewDocumentParser(client *Client) *DocumentParser {
	return &DocumentParser{
		client:       client,
		sharesParser: NewSharesParser(),
	}
}

// FetchAndParseDocument fetches and parses a single SEC document
func (c *Client) FetchAndParseDocument(filing Filing, storage *CompanyDataStorage, symbol string) (*FilingProcessingResult, error) {
	return c.FetchAndParseDocumentWithParser(filing, storage, symbol, nil)
}

// FetchAndParseDocumentWithParser fetches and parses a single SEC document with optional enhanced parser
func (c *Client) FetchAndParseDocumentWithParser(filing Filing, storage *CompanyDataStorage, symbol string, enhancedParser *EnhancedDocumentParser) (*FilingProcessingResult, error) {
	// Fetch the document content
	content, err := c.FetchDocumentContent(filing.DocumentURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching document: %w", err)
	}

	// Save raw filing content
	rawDoc, err := storage.SaveRawFiling(symbol, filing, content)
	if err != nil {
		return nil, fmt.Errorf("error saving raw filing: %w", err)
	}

	// Parse the content with optional enhanced parser
	extractedData, err := c.parseDocumentContentWithParser(content, filing, enhancedParser)
	if err != nil {
		// Still save the processing result even if parsing failed
		result := &FilingProcessingResult{
			Document:         *rawDoc,
			ProcessingErrors: []string{err.Error()},
			ProcessedAt:      time.Now(),
		}

		// Save processing result
		if saveErr := storage.SaveProcessingResult(symbol, result); saveErr != nil {
			return nil, fmt.Errorf("error saving processing result: %w", saveErr)
		}

		return result, fmt.Errorf("error parsing document: %w", err)
	}

	// Update raw document with processing info
	rawDoc.ProcessedAt = time.Now()
	rawDoc.ProcessingNotes = fmt.Sprintf("Successfully extracted %d BTC transactions and shares data",
		len(extractedData.BTCTransactions))
	if extractedData.SharesOutstanding != nil {
		rawDoc.ProcessingNotes += " with shares outstanding"
	}

	// Create successful processing result
	result := &FilingProcessingResult{
		Document:      *rawDoc,
		ExtractedData: extractedData,
		ProcessedAt:   time.Now(),
	}

	// Save processing result
	if err := storage.SaveProcessingResult(symbol, result); err != nil {
		return nil, fmt.Errorf("error saving processing result: %w", err)
	}

	return result, nil
}

// parseDocumentContent parses document content and extracts financial data
func (c *Client) parseDocumentContent(content []byte, filing Filing) (*ExtractedFinancialData, error) {
	return c.parseDocumentContentWithParser(content, filing, nil)
}

// parseDocumentContentWithParser parses document content using a specific parser
func (c *Client) parseDocumentContentWithParser(content []byte, filing Filing, enhancedParser *EnhancedDocumentParser) (*ExtractedFinancialData, error) {
	result := &ExtractedFinancialData{
		Filing:      filing,
		ProcessedAt: time.Now(),
	}

	// Create appropriate parser
	var parser interface {
		ParseHTMLDocument([]byte, Filing) ([]BitcoinTransaction, error)
		ParseTextDocument([]byte, Filing) ([]BitcoinTransaction, error)
	}

	if enhancedParser != nil {
		// Use enhanced parser with Grok capabilities
		parser = &enhancedParserWrapper{enhancedParser}
	} else {
		// Use standard regex parser
		parser = &DocumentParser{client: c}
	}

	// Extract Bitcoin transactions
	var err error
	if strings.HasSuffix(filing.DocumentURL, ".htm") || strings.HasSuffix(filing.DocumentURL, ".html") {
		result.BTCTransactions, err = parser.ParseHTMLDocument(content, filing)
		if err != nil {
			result.ProcessingErrors = append(result.ProcessingErrors, fmt.Sprintf("BTC extraction error: %v", err))
		}
	} else {
		result.BTCTransactions, err = parser.ParseTextDocument(content, filing)
		if err != nil {
			result.ProcessingErrors = append(result.ProcessingErrors, fmt.Sprintf("BTC extraction error: %v", err))
		}
	}

	// Extract shares outstanding (only for 10-Q and 10-K filings)
	if filing.FilingType == "10-Q" || filing.FilingType == "10-K" {
		sharesParser := NewSharesParser()
		result.SharesOutstanding, err = sharesParser.ExtractSharesFromFiling(content, filing)
		if err != nil {
			result.ProcessingErrors = append(result.ProcessingErrors, fmt.Sprintf("Shares extraction error: %v", err))
		}
	}

	return result, nil
}

// enhancedParserWrapper wraps the EnhancedDocumentParser to match the interface
type enhancedParserWrapper struct {
	parser *EnhancedDocumentParser
}

func (w *enhancedParserWrapper) ParseHTMLDocument(content []byte, filing Filing) ([]BitcoinTransaction, error) {
	return w.parser.ParseHTMLDocumentEnhanced(content, filing)
}

func (w *enhancedParserWrapper) ParseTextDocument(content []byte, filing Filing) ([]BitcoinTransaction, error) {
	// For text documents, fall back to regex parser
	return w.parser.regexParser.ParseTextDocument(content, filing)
}

// ParseHTMLDocument parses an HTML document for Bitcoin transactions
func (p *DocumentParser) ParseHTMLDocument(body []byte, filing Filing) ([]BitcoinTransaction, error) {
	transactions := []BitcoinTransaction{}
	seenTransactions := make(map[string]bool) // For deduplication

	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML document: %w", err)
	}

	// First, look for the content in paragraphs
	bitcoinParagraphs := []string{}

	// Look for paragraphs containing Bitcoin-related keywords
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if containsBitcoinKeywords(text) {
			bitcoinParagraphs = append(bitcoinParagraphs, strings.TrimSpace(text))
		}
	})

	// If no paragraphs found with Bitcoin references, look in other elements
	if len(bitcoinParagraphs) == 0 {
		doc.Find("div, td, li, span").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			if containsBitcoinKeywords(text) {
				bitcoinParagraphs = append(bitcoinParagraphs, strings.TrimSpace(text))
			}
		})
	}

	// As a last resort, get the entire body text and look for Bitcoin references
	if len(bitcoinParagraphs) == 0 {
		bodyText := doc.Text()
		paragraphs := strings.Split(bodyText, "\n")

		for _, paragraph := range paragraphs {
			paragraph = strings.TrimSpace(paragraph)
			if paragraph != "" && containsBitcoinKeywords(paragraph) {
				bitcoinParagraphs = append(bitcoinParagraphs, paragraph)
			}
		}
	}

	// Process each Bitcoin-related paragraph
	for _, paragraph := range bitcoinParagraphs {
		// Try different parsing approaches
		txns := extractTransactionsFromText(paragraph, filing)

		// If standard patterns didn't work, try more flexible patterns
		if len(txns) == 0 {
			tx, found := extractBitcoinTransaction(paragraph, filing)
			if found {
				txns = append(txns, tx)
			}
		}

		// Add transactions with deduplication
		for _, tx := range txns {
			// Create a unique key for deduplication based on extracted text
			// Use first 100 characters of extracted text to identify duplicates
			textKey := tx.ExtractedText
			if len(textKey) > 100 {
				textKey = textKey[:100]
			}

			if !seenTransactions[textKey] {
				seenTransactions[textKey] = true
				transactions = append(transactions, tx)
			}
		}
	}

	return transactions, nil
}

// ParseTextDocument parses a text document for Bitcoin transactions
func (p *DocumentParser) ParseTextDocument(body []byte, filing Filing) ([]BitcoinTransaction, error) {
	text := string(body)

	// Split the document into paragraphs
	paragraphs := strings.Split(text, "\n\n")

	transactions := []BitcoinTransaction{}

	// Process each paragraph
	for _, paragraph := range paragraphs {
		if containsBitcoinKeywords(paragraph) {
			txns := extractTransactionsFromText(paragraph, filing)
			transactions = append(transactions, txns...)
		}
	}

	return transactions, nil
}

// containsBitcoinKeywords checks if text contains Bitcoin-related keywords
func containsBitcoinKeywords(text string) bool {
	text = strings.ToLower(text)

	// Primary Bitcoin keywords - must have at least one
	bitcoinKeywords := []string{
		"bitcoin", "btc", "digital asset", "cryptocurrency",
	}

	hasBitcoinKeyword := false
	for _, keyword := range bitcoinKeywords {
		if strings.Contains(text, keyword) {
			hasBitcoinKeyword = true
			break
		}
	}

	if !hasBitcoinKeyword {
		return false
	}

	// Exclude financing/bond/loan activities that mention Bitcoin but aren't purchases
	excludeKeywords := []string{
		"convertible senior notes", "convertible notes", "bond offering",
		"private offering", "note offering", "debt offering",
		"loan proceeds", "net proceeds", "financing", "borrowing",
		"credit facility", "line of credit", "term loan",
		"purchase agreement", "underwriting", "initial purchaser",
		"qualified institutional buyers", "rule 144a",
		"securities act", "aggregate principal amount",
	}

	for _, exclude := range excludeKeywords {
		if strings.Contains(text, exclude) {
			// Only exclude if this is clearly about financing and not actual Bitcoin purchases
			if !strings.Contains(text, "purchased") && !strings.Contains(text, "acquired") && !strings.Contains(text, "bought") {
				return false
			}
		}
	}

	// Exclude intent/future statements that aren't actual transactions
	intentKeywords := []string{
		"intends to", "plans to", "will invest", "may invest", "expects to",
		"anticipates", "pending", "for the purpose of", "to purchase",
		"to acquire", "to invest", "will use", "may use",
	}

	for _, intent := range intentKeywords {
		if strings.Contains(text, intent) {
			// This is about future intent, not completed transactions
			return false
		}
	}

	// Must also contain action keywords for actual completed transactions
	actionKeywords := []string{
		"purchased", "acquired", "bought", "completed", "announced that it had",
	}

	for _, keyword := range actionKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

// extractTransactionsFromText extracts Bitcoin transactions from text
func extractTransactionsFromText(text string, filing Filing) []BitcoinTransaction {
	transactions := []BitcoinTransaction{}

	// Skip if this is clearly about financing/bonds/loans and not actual Bitcoin purchases
	if isFinancingActivity(text) {
		return transactions
	}

	// Enhanced regex patterns to match different transaction descriptions
	patterns := []*regexp.Regexp{
		// Pattern 1: "purchased X bitcoins at an average price of $Y per bitcoin for a total of $Z"
		regexp.MustCompile(`(?i)(?:purchased|acquired|bought)\s+(?:approximately\s+)?([0-9,]+(?:\.[0-9]+)?)\s+(?:additional\s+)?bitcoins?.*?average\s+price\s+of\s+(?:approximately\s+)?\$([0-9,.]+).*?(?:total|aggregate)\s+(?:of|purchase\s+price)\s+(?:of\s+)?\$([0-9,.]+)(?:\s+million|\s+billion)?`),

		// Pattern 2: "acquired X bitcoins for $Y million at an average price of $Z"
		regexp.MustCompile(`(?i)(?:acquired|purchased|bought)\s+(?:approximately\s+)?([0-9,]+(?:\.[0-9]+)?)\s+(?:additional\s+)?bitcoins?\s+for\s+\$([0-9,.]+)(?:\s+million|\s+billion)?\s+.*?average\s+price\s+of\s+(?:approximately\s+)?\$([0-9,]+)`),

		// Pattern 3: "purchased X bitcoins for $Y million" (without explicit price)
		regexp.MustCompile(`(?i)(?:purchased|acquired|bought)\s+(?:approximately\s+)?([0-9,]+(?:\.[0-9]+)?)\s+(?:additional\s+)?bitcoins?\s+for\s+\$([0-9,.]+)(?:\s+million|\s+billion)?(?:\s+in\s+cash)?`),

		// Pattern 4: "invested $X to purchase Y bitcoins"
		regexp.MustCompile(`(?i)invested\s+\$([0-9,.]+)(?:\s+million|\s+billion)?\s+.*?(?:purchase|acquire|buy)\s+(?:approximately\s+)?([0-9,]+(?:\.[0-9]+)?)\s+bitcoins?`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) > 0 {
			transaction := BitcoinTransaction{
				FilingType:    filing.FilingType,
				FilingURL:     filing.URL,
				Date:          filing.ReportDate,
				ExtractedText: text,
			}

			// Parse based on pattern matched
			if pattern == patterns[0] && len(matches) >= 4 {
				// Pattern 1: purchased X bitcoins at average price Y for total Z
				btc := parseNumber(matches[1])
				price := parseNumber(matches[2])
				total := parseNumber(matches[3])

				// Handle millions/billions
				if strings.Contains(strings.ToLower(text), "million") {
					total *= 1_000_000
				} else if strings.Contains(strings.ToLower(text), "billion") {
					total *= 1_000_000_000
				}

				transaction.BTCPurchased = btc
				transaction.AvgPriceUSD = price
				transaction.USDSpent = total
				transaction.ConfidenceScore = 0.95 // Very high confidence for complete match

			} else if pattern == patterns[1] && len(matches) >= 4 {
				// Pattern 2: acquired X bitcoins for $Y million at average price of $Z
				btc := parseNumber(matches[1])
				total := parseNumber(matches[2])
				price := parseNumber(matches[3])

				// Handle millions/billions
				if strings.Contains(strings.ToLower(text), "million") {
					total *= 1_000_000
				} else if strings.Contains(strings.ToLower(text), "billion") {
					total *= 1_000_000_000
				}

				transaction.BTCPurchased = btc
				transaction.AvgPriceUSD = price
				transaction.USDSpent = total
				transaction.ConfidenceScore = 0.95

			} else if pattern == patterns[2] && len(matches) >= 3 {
				// Pattern 3: purchased X bitcoins for $Y million
				btc := parseNumber(matches[1])
				total := parseNumber(matches[2])

				// Handle millions/billions
				if strings.Contains(strings.ToLower(text), "million") {
					total *= 1_000_000
				} else if strings.Contains(strings.ToLower(text), "billion") {
					total *= 1_000_000_000
				}

				transaction.BTCPurchased = btc
				transaction.USDSpent = total
				if btc > 0 {
					transaction.AvgPriceUSD = total / btc
				}
				transaction.ConfidenceScore = 0.85 // High confidence but missing explicit price

			} else if pattern == patterns[3] && len(matches) >= 3 {
				// Pattern 4: invested $X to purchase Y bitcoins
				total := parseNumber(matches[1])
				btc := parseNumber(matches[2])

				// Handle millions/billions
				if strings.Contains(strings.ToLower(text), "million") {
					total *= 1_000_000
				} else if strings.Contains(strings.ToLower(text), "billion") {
					total *= 1_000_000_000
				}

				transaction.BTCPurchased = btc
				transaction.USDSpent = total
				if btc > 0 {
					transaction.AvgPriceUSD = total / btc
				}
				transaction.ConfidenceScore = 0.85
			}

			// Validate the transaction makes sense
			if isValidTransaction(transaction) {
				transactions = append(transactions, transaction)
			}
		}
	}

	return transactions
}

// isFinancingActivity checks if the text is about financing/bonds/loans rather than Bitcoin purchases
func isFinancingActivity(text string) bool {
	text = strings.ToLower(text)

	// Strong indicators this is about financing, not Bitcoin purchases
	financingIndicators := []string{
		"convertible senior notes", "convertible notes", "private offering", "bond offering",
		"note offering", "debt offering", "net proceeds", "aggregate principal amount",
		"initial purchaser", "qualified institutional buyers", "rule 144a", "securities act",
		"purchase agreement", "underwriting", "discounts and commissions",
		"offering expenses", "resale to qualified", "pursuant to rule",
	}

	// Count financing indicators
	financingCount := 0
	for _, indicator := range financingIndicators {
		if strings.Contains(text, indicator) {
			financingCount++
		}
	}

	// If we have multiple financing indicators, this is likely about financing
	if financingCount >= 2 {
		return true
	}

	// Single strong indicators that almost always mean financing
	strongFinancingIndicators := []string{
		"convertible senior notes", "private offering", "qualified institutional buyers",
		"rule 144a", "securities act", "net proceeds", "aggregate principal amount",
	}

	for _, indicator := range strongFinancingIndicators {
		if strings.Contains(text, indicator) {
			// Only allow if it explicitly mentions Bitcoin purchases in the same sentence
			sentences := strings.Split(text, ".")
			for _, sentence := range sentences {
				if strings.Contains(sentence, indicator) {
					// Check if this sentence also mentions Bitcoin purchases
					if !strings.Contains(sentence, "purchased") && !strings.Contains(sentence, "acquired") {
						return true
					}
					if !strings.Contains(sentence, "bitcoin") && !strings.Contains(sentence, "btc") {
						return true
					}
				}
			}
		}
	}

	return false
}

// parseNumber parses a number string, removing commas
func parseNumber(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	num, _ := strconv.ParseFloat(s, 64)
	return num
}

// isValidTransaction validates that a transaction has reasonable values
func isValidTransaction(tx BitcoinTransaction) bool {
	// Must have positive BTC amount
	if tx.BTCPurchased <= 0 {
		return false
	}

	// Must have positive USD amount
	if tx.USDSpent <= 0 {
		return false
	}

	// Price per Bitcoin should be reasonable (between $100 and $1,000,000)
	if tx.AvgPriceUSD > 0 && (tx.AvgPriceUSD < 100 || tx.AvgPriceUSD > 1_000_000) {
		return false
	}

	// Calculate implied price if not provided
	if tx.AvgPriceUSD == 0 && tx.BTCPurchased > 0 && tx.USDSpent > 0 {
		impliedPrice := tx.USDSpent / tx.BTCPurchased
		if impliedPrice < 100 || impliedPrice > 1_000_000 {
			return false
		}
	}

	// BTC amount should be reasonable (not more than 100,000 in a single transaction)
	if tx.BTCPurchased > 100_000 {
		return false
	}

	return true
}

// extractBitcoinTransaction extracts a Bitcoin transaction using more flexible patterns
func extractBitcoinTransaction(paragraph string, filing Filing) (BitcoinTransaction, bool) {
	// Default transaction
	tx := BitcoinTransaction{
		Date:          filing.FilingDate,
		FilingType:    filing.FilingType,
		FilingURL:     filing.URL,
		ExtractedText: paragraph,
	}

	// First check for explicit purchase patterns - These are the most reliable indicators
	purchasePatterns := []*regexp.Regexp{
		// "acquired X bitcoins for $Y"
		regexp.MustCompile(`(?i)(?:acquired|purchased).*?([0-9,]+(?:\.[0-9]+)?)\s+bitcoins?.*?\$([0-9,.]+)(?:\s+million|\s+billion)?`),
		// "acquired/purchased additional X bitcoins for $Y million/billion at an average price of approximately $Z"
		regexp.MustCompile(`(?i)(?:acquired|purchased).*?([0-9,]+(?:\.[0-9]+)?)\s+bitcoins?\s+for\s+\$([0-9,.]+)(?:\s+million|\s+billion)?.*?average\s+price\s+of\s+(?:approximately)?\s+\$([0-9,]+)`),
	}

	// Try explicit purchase patterns first
	for _, pattern := range purchasePatterns {
		matches := pattern.FindStringSubmatch(paragraph)
		if len(matches) >= 3 {
			// We found a clear purchase statement
			btcStr := strings.ReplaceAll(matches[1], ",", "")
			btc, err := strconv.ParseFloat(btcStr, 64)
			if err == nil {
				tx.BTCPurchased = btc
			}

			usdStr := strings.ReplaceAll(matches[2], ",", "")
			usd, err := strconv.ParseFloat(usdStr, 64)
			if err == nil {
				// Check if amount is in millions/billions
				if strings.Contains(paragraph, "million") {
					usd *= 1000000
				} else if strings.Contains(paragraph, "billion") {
					usd *= 1000000000
				}
				tx.USDSpent = usd
			}

			// If we have a price component (3rd match group)
			if len(matches) >= 4 {
				priceStr := strings.ReplaceAll(matches[3], ",", "")
				price, err := strconv.ParseFloat(priceStr, 64)
				if err == nil {
					tx.AvgPriceUSD = price
				}
			}

			// Calculate price if missing
			if tx.BTCPurchased > 0 && tx.USDSpent > 0 && tx.AvgPriceUSD == 0 {
				tx.AvgPriceUSD = tx.USDSpent / tx.BTCPurchased
			}

			tx.ConfidenceScore = 0.9 // High confidence for explicit match
			return tx, true
		}
	}

	// Check if this is a purchase statement or a holdings statement
	isPurchase := containsAny(paragraph, []string{"purchased", "acquired", "buying", "buy", "bought"})
	isHoldings := containsAny(paragraph, []string{"held", "holds", "holding", "as of"})

	// Common patterns in holdings statements
	isHoldingsStatement := isHoldings ||
		containsAny(paragraph, []string{
			"as of", "held a total", "total of", "holdings increased to",
			"aggregate purchase", "average purchase price", "were acquired",
			"aggregate", "total", "approximately"})

	// Extract Bitcoin amount - look for numbers near bitcoin keyword
	var btcRegex *regexp.Regexp
	if isPurchase && !isHoldingsStatement {
		btcRegex = regexp.MustCompile(`(?i)(?:acquired|purchased|bought)?\s*(?:approximately|about)?\s*([0-9,]+(?:\.[0-9]+)?)\s*(?:additional)?\s*(?:bitcoins?|BTC)`)
	} else {
		btcRegex = regexp.MustCompile(`(?i)(?:held|holds|holding)?\s*(?:approximately|about|a total of)?\s*([0-9,]+(?:\.[0-9]+)?)\s*(?:bitcoins?|BTC)`)
	}

	btcMatches := btcRegex.FindStringSubmatch(paragraph)

	if len(btcMatches) >= 2 {
		// Remove commas
		btcStr := strings.ReplaceAll(btcMatches[1], ",", "")
		btc, err := strconv.ParseFloat(btcStr, 64)
		if err == nil {
			tx.BTCPurchased = btc
		}
	}

	// Extract USD amount - look for dollar amounts
	// Try different patterns for USD amounts
	usdPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$([0-9,]+(?:\.[0-9]+)?)\s*(?:million|billion|M|B)?`),
		regexp.MustCompile(`([0-9,]+(?:\.[0-9]+)?)\s*(?:million|billion|M|B)?\s*(?:dollars|USD)`),
	}

	for _, pattern := range usdPatterns {
		usdMatches := pattern.FindStringSubmatch(paragraph)
		if len(usdMatches) >= 2 {
			// Remove commas
			usdStr := strings.ReplaceAll(usdMatches[1], ",", "")
			usd, err := strconv.ParseFloat(usdStr, 64)
			if err == nil {
				// Adjust for millions/billions
				if strings.Contains(usdMatches[0], "million") || strings.Contains(usdMatches[0], "M") {
					usd *= 1000000
				} else if strings.Contains(usdMatches[0], "billion") || strings.Contains(usdMatches[0], "B") {
					usd *= 1000000000
				}

				tx.USDSpent = usd
				break
			}
		}
	}

	// Extract price per Bitcoin
	priceRegex := regexp.MustCompile(`(?i)(?:average|avg\.?|approximate)?\s*(?:price|cost)\s*(?:of|per)?\s*(?:approximately|about)?\s*\$([0-9,]+(?:\.[0-9]+)?)`)
	priceMatches := priceRegex.FindStringSubmatch(paragraph)

	if len(priceMatches) >= 2 {
		// Remove commas
		priceStr := strings.ReplaceAll(priceMatches[1], ",", "")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			tx.AvgPriceUSD = price
		}
	}

	// Determine if this is a holdings statement vs. a purchase
	// If we see explicit "held" or "as of" language, or if the BTC amount is very large, treat as holdings
	if isHoldingsStatement || (tx.BTCPurchased > 100000 && !isPurchase) { // MicroStrategy's holdings in 2023 were ~140k BTC
		// Mark this as a holdings update by setting a special flag
		tx.TotalBTCAfter = tx.BTCPurchased
		// Only count as a purchase if it explicitly mentions a purchase
		tx.BTCPurchased = 0
	}

	// Calculate missing values if possible
	if tx.BTCPurchased > 0 && tx.USDSpent > 0 && tx.AvgPriceUSD == 0 {
		tx.AvgPriceUSD = tx.USDSpent / tx.BTCPurchased
	} else if tx.BTCPurchased > 0 && tx.AvgPriceUSD > 0 && tx.USDSpent == 0 {
		tx.USDSpent = tx.BTCPurchased * tx.AvgPriceUSD
	}

	// Set confidence score based on available data
	if tx.BTCPurchased > 0 && tx.USDSpent > 0 && tx.AvgPriceUSD > 0 {
		tx.ConfidenceScore = 0.9 // High confidence if all three values are present
	} else if tx.BTCPurchased > 0 && (tx.USDSpent > 0 || tx.AvgPriceUSD > 0) {
		tx.ConfidenceScore = 0.7 // Medium confidence if two values are present
	} else if tx.BTCPurchased > 0 || tx.USDSpent > 0 || tx.AvgPriceUSD > 0 || tx.TotalBTCAfter > 0 {
		tx.ConfidenceScore = 0.5 // Low confidence if only one value is present
	} else {
		tx.ConfidenceScore = 0.1 // Very low confidence if no values are present
		return tx, false         // Skip this transaction
	}

	return tx, true
}

// containsAny returns true if the text contains any of the keywords
func containsAny(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// ProcessCompanyFilings processes all filings for a company to extract Bitcoin transactions and shares outstanding
func (p *DocumentParser) ProcessCompanyFilings(ticker string, filingTypes []string, startDate, endDate string) (*CompanyFinancialData, error) {
	// Get company filings
	filings, err := p.client.GetCompanyFilings(ticker, filingTypes, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error getting filings: %w", err)
	}

	cik, _ := p.client.GetCIKByTicker(ticker)

	result := &CompanyFinancialData{
		Symbol:      ticker,
		CompanyName: ticker, // Will be updated with proper name later
		CIK:         cik,
		LastUpdated: time.Now(),
	}

	// Process each filing
	for _, filing := range filings {
		extractedData, err := p.client.FetchAndParseDocument(filing, nil, ticker)
		if err != nil {
			fmt.Printf("Warning: Error processing filing %s: %v\n", filing.AccessionNumber, err)
			continue
		}

		// Add Bitcoin transactions
		if extractedData.ExtractedData != nil {
			result.BTCTransactions = append(result.BTCTransactions, extractedData.ExtractedData.BTCTransactions...)

			// Add shares outstanding record if found
			if extractedData.ExtractedData.SharesOutstanding != nil {
				result.SharesHistory = append(result.SharesHistory, *extractedData.ExtractedData.SharesOutstanding)
			}
		}

		// Update last filing date
		if filing.FilingDate.After(result.LastFilingDate) {
			result.LastFilingDate = filing.FilingDate
		}
	}

	result.LastProcessedDate = time.Now()

	return result, nil
}
