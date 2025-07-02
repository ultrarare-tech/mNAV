package parser

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ultrarare-tech/mNAV/pkg/interpretation/grok"
	"github.com/ultrarare-tech/mNAV/pkg/shared/models"
)

// EnhancedParser combines regex-based parsing with Grok AI for maximum accuracy
type EnhancedParser struct {
	grokClient *grok.Client
	verbose    bool
}

// NewEnhancedParser creates a new enhanced parser with optional Grok integration
func NewEnhancedParser(grokClient *grok.Client, verbose bool) *EnhancedParser {
	return &EnhancedParser{
		grokClient: grokClient,
		verbose:    verbose,
	}
}

// BitcoinParagraph represents a paragraph that mentions Bitcoin with numerical values
type BitcoinParagraph struct {
	Text           string
	Numbers        []string
	Context        string
	ParagraphIndex int
}

// ParseFiling processes a filing using the two-stage approach
func (p *EnhancedParser) ParseFiling(content, filingType string, filingPath string) (*models.FilingParseResult, error) {
	if p.verbose {
		log.Printf("Starting two-stage parsing for %s filing (%d chars)", filingType, len(content))
	}

	// Parse filing metadata from the file path
	filing := p.parseFilingMetadata(filingPath, filingType)

	result := &models.FilingParseResult{
		Filing:   filing,
		ParsedAt: time.Now(),
	}

	startTime := time.Now()

	// Stage 1: Use regex to identify Bitcoin-related paragraphs with numerical values
	bitcoinParagraphs := p.extractBitcoinParagraphs(content)

	if p.verbose {
		log.Printf("Stage 1: Found %d Bitcoin-related paragraphs with numerical values", len(bitcoinParagraphs))
	}

	var parsingMethod string

	if len(bitcoinParagraphs) == 0 {
		if p.verbose {
			log.Printf("No Bitcoin-related paragraphs found, skipping Grok analysis")
		}
		parsingMethod = "Enhanced Parser (No Bitcoin content found)"
	} else {
		// Stage 2: Send identified paragraphs to Grok for interpretation
		transactions, err := p.interpretParagraphsWithGrok(bitcoinParagraphs, filing)
		if err != nil {
			if p.verbose {
				log.Printf("Grok interpretation failed: %v", err)
			}
			// Fall back to regex-only parsing
			transactions = p.fallbackRegexParsing(bitcoinParagraphs, filing)
			parsingMethod = "Enhanced Parser (Regex fallback - Grok failed)"
		} else {
			parsingMethod = "Enhanced Parser (Grok AI + Regex)"
		}

		// Populate filing metadata in transactions
		for i := range transactions {
			transactions[i].FilingType = filing.FilingType
			transactions[i].FilingURL = filing.URL
		}

		result.BitcoinTransactions = transactions
	}

	// Extract shares information
	result.SharesOutstanding = p.extractSharesFromParagraphs(bitcoinParagraphs, filing)

	// Set processing metadata
	result.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	result.ParsingMethod = parsingMethod

	if p.verbose {
		sharesCount := 0
		if result.SharesOutstanding != nil {
			sharesCount = 1
		}
		log.Printf("Parsing complete: found %d transactions, %d shares entries (method: %s)",
			len(result.BitcoinTransactions), sharesCount, parsingMethod)
	}

	return result, nil
}

// parseFilingMetadata extracts filing metadata from the file path
func (p *EnhancedParser) parseFilingMetadata(filePath, filingType string) models.Filing {
	fileName := filepath.Base(filePath)

	// Expected format: YYYY-MM-DD_FORM-TYPE_ACCESSION-NUMBER.htm
	parts := strings.Split(strings.TrimSuffix(fileName, ".htm"), "_")

	filing := models.Filing{
		FilingType:  filingType,
		DocumentURL: filePath,
	}

	if len(parts) >= 3 {
		// Parse date
		if date, err := time.Parse("2006-01-02", parts[0]); err == nil {
			filing.FilingDate = date
			filing.ReportDate = date
		}

		// Parse accession number
		filing.AccessionNumber = parts[2]

		// Construct SEC URL
		filing.URL = fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/1050446/%s/%s",
			strings.ReplaceAll(parts[2], "-", ""), fileName)
	}

	return filing
}

// extractBitcoinParagraphs identifies paragraphs that mention Bitcoin with numerical values
func (p *EnhancedParser) extractBitcoinParagraphs(content string) []BitcoinParagraph {
	var bitcoinParagraphs []BitcoinParagraph

	// Split content into paragraphs (by double newlines or HTML paragraph tags)
	paragraphSeparators := regexp.MustCompile(`\n\s*\n|</?p[^>]*>|<br\s*/?>\s*<br\s*/?>`)
	paragraphs := paragraphSeparators.Split(content, -1)

	// Bitcoin-related keywords (case-insensitive)
	bitcoinKeywords := regexp.MustCompile(`(?i)\b(bitcoin|btc|digital\s+asset|cryptocurrency|crypto\s+asset|digital\s+currency)\b`)

	// Numerical patterns that might indicate amounts, prices, or values
	numericalPatterns := []*regexp.Regexp{
		// Large numbers (potential BTC amounts or USD values)
		regexp.MustCompile(`\b\d{1,3}(?:,\d{3})*(?:\.\d+)?\b`),
		// Dollar amounts
		regexp.MustCompile(`\$\s*\d+(?:,\d{3})*(?:\.\d+)?(?:\s*(?:million|billion|thousand|M|B|K))?\b`),
		// Percentages
		regexp.MustCompile(`\d+(?:\.\d+)?%`),
		// Decimal numbers
		regexp.MustCompile(`\b\d+\.\d+\b`),
	}

	// Transaction-related keywords that suggest purchases, sales, or holdings
	transactionKeywords := regexp.MustCompile(`(?i)\b(purchase|purchased|buy|bought|acquire|acquired|acquisition|sell|sold|sale|hold|holding|holdings|invest|investment|treasury|reserve|asset)\b`)

	// Date range patterns that indicate cumulative totals (should be excluded)
	// Updated based on SaylorTracker validation analysis
	dateRangePatterns := []*regexp.Regexp{
		// "During the period between X and Y" patterns (most common in MSTR filings)
		regexp.MustCompile(`(?i)\bduring\s+the\s+period\s+between\s+[^,]+\s+and\s+[^,]+\b`),
		// "Since X through Y" patterns
		regexp.MustCompile(`(?i)\bsince\s+[^,]+\s+through\s+[^,]+\b`),
		// "From X to Y" patterns
		regexp.MustCompile(`(?i)\bfrom\s+[^,]+\s+to\s+[^,]+\b`),
		// "Between X and Y" patterns (when not preceded by "period")
		regexp.MustCompile(`(?i)\bbetween\s+[^,]+\s+and\s+[^,]+\b`),
		// "During the quarter" patterns
		regexp.MustCompile(`(?i)\bduring\s+the\s+(quarter|year|month)\b`),
		// "For the three months" patterns
		regexp.MustCompile(`(?i)\bfor\s+the\s+(three|six|nine|twelve)\s+months?\b`),
		// "In the quarter ended" patterns
		regexp.MustCompile(`(?i)\bin\s+the\s+quarter\s+ended\b`),
		// "Over the period" patterns
		regexp.MustCompile(`(?i)\bover\s+the\s+(period|quarter|year)\b`),
		// "For the period" patterns
		regexp.MustCompile(`(?i)\bfor\s+the\s+period\b`),
		// "Throughout the" patterns
		regexp.MustCompile(`(?i)\bthroughout\s+the\s+(period|quarter|year|month)\b`),
	}

	// Cumulative keywords that suggest totals rather than individual transactions
	cumulativeKeywords := regexp.MustCompile(`(?i)\b(total|aggregate|cumulative|combined|overall|sum|collectively)\b`)

	for i, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if len(paragraph) < 50 { // Skip very short paragraphs
			continue
		}

		// Check if paragraph mentions Bitcoin
		if !bitcoinKeywords.MatchString(paragraph) {
			continue
		}

		// Extract all numerical values from the paragraph
		var numbers []string
		for _, pattern := range numericalPatterns {
			matches := pattern.FindAllString(paragraph, -1)
			numbers = append(numbers, matches...)
		}

		// Only include paragraphs with numerical values
		if len(numbers) == 0 {
			continue
		}

		// Determine context based on surrounding keywords
		context := "unknown"

		// Check for date ranges first (these indicate cumulative totals)
		hasDateRange := false
		for _, pattern := range dateRangePatterns {
			if pattern.MatchString(paragraph) {
				hasDateRange = true
				break
			}
		}

		if hasDateRange {
			context = "cumulative_range"
		} else if transactionKeywords.MatchString(paragraph) {
			context = "transaction"
		}

		// Additional context clues
		if cumulativeKeywords.MatchString(paragraph) {
			if context == "transaction" {
				context = "cumulative_total"
			} else if context == "unknown" {
				context = "cumulative"
			}
		}

		if regexp.MustCompile(`(?i)\b(per\s+share|average|price)\b`).MatchString(paragraph) {
			context = "pricing"
		}

		bitcoinParagraphs = append(bitcoinParagraphs, BitcoinParagraph{
			Text:           paragraph,
			Numbers:        numbers,
			Context:        context,
			ParagraphIndex: i,
		})

		if p.verbose {
			log.Printf("Found Bitcoin paragraph %d (context: %s): %.100s...", i, context, paragraph)
		}
	}

	return bitcoinParagraphs
}

// interpretParagraphsWithGrok sends the identified paragraphs to Grok for interpretation
func (p *EnhancedParser) interpretParagraphsWithGrok(paragraphs []BitcoinParagraph, filing models.Filing) ([]models.BitcoinTransaction, error) {
	if p.grokClient == nil {
		return nil, fmt.Errorf("Grok client not available")
	}

	// Combine paragraphs into a focused prompt
	var combinedText strings.Builder
	combinedText.WriteString("BITCOIN TRANSACTION ANALYSIS\n")
	combinedText.WriteString("Filing Type: " + filing.FilingType + "\n\n")
	combinedText.WriteString("The following paragraphs from an SEC filing mention Bitcoin with numerical values. ")
	combinedText.WriteString("Please analyze each paragraph and extract any Bitcoin transactions (purchases, sales, or holdings).\n\n")

	for i, para := range paragraphs {
		combinedText.WriteString(fmt.Sprintf("PARAGRAPH %d (Context: %s):\n", i+1, para.Context))
		combinedText.WriteString(para.Text)
		combinedText.WriteString("\n\n")
	}

	combinedText.WriteString("INSTRUCTIONS:\n")
	combinedText.WriteString("1. Identify any Bitcoin purchases, sales, or holdings mentioned\n")
	combinedText.WriteString("2. Extract: date, BTC amount, USD amount, average price per BTC\n")
	combinedText.WriteString("3. Distinguish between individual transactions and cumulative totals\n")
	combinedText.WriteString("4. If a paragraph describes cumulative holdings, mark it as 'cumulative'\n")
	combinedText.WriteString("5. Only extract clear, specific transaction data\n")

	if p.verbose {
		log.Printf("Sending %d paragraphs to Grok for interpretation (%d chars)",
			len(paragraphs), combinedText.Len())
	}

	transactions, err := p.grokClient.ExtractBitcoinTransactions(combinedText.String(), filing)
	if err != nil {
		return nil, fmt.Errorf("Grok extraction failed: %w", err)
	}

	if p.verbose {
		log.Printf("Grok returned %d transactions", len(transactions))
	}

	return transactions, nil
}

// fallbackRegexParsing provides basic regex parsing when Grok is unavailable
func (p *EnhancedParser) fallbackRegexParsing(paragraphs []BitcoinParagraph, filing models.Filing) []models.BitcoinTransaction {
	var transactions []models.BitcoinTransaction

	// Basic regex patterns for transaction extraction
	purchasePattern := regexp.MustCompile(`(?i)(?:purchased|acquired|bought)\s+(?:approximately\s+)?(\d+(?:,\d{3})*(?:\.\d+)?)\s+(?:bitcoin|btc)`)
	amountPattern := regexp.MustCompile(`\$\s*(\d+(?:,\d{3})*(?:\.\d+)?)\s*(?:million|billion|M|B)?`)
	datePattern := regexp.MustCompile(`(?:january|february|march|april|may|june|july|august|september|october|november|december|\d{1,2}[/-]\d{1,2}[/-]\d{2,4}|\d{4}-\d{2}-\d{2})`)

	for _, para := range paragraphs {
		// Skip paragraphs that have been classified as cumulative totals or date ranges
		if para.Context == "cumulative_range" || para.Context == "cumulative_total" || para.Context == "cumulative" {
			if p.verbose {
				log.Printf("Skipping cumulative paragraph: %s", para.Context)
			}
			continue
		}

		// Additional checks for cumulative language based on SaylorTracker analysis
		lowerText := strings.ToLower(para.Text)
		if strings.Contains(lowerText, "during the period between") ||
			strings.Contains(lowerText, "the period between") ||
			strings.Contains(lowerText, "during the quarter") ||
			strings.Contains(lowerText, "for the quarter") ||
			strings.Contains(lowerText, "for the period") {
			if p.verbose {
				log.Printf("Skipping paragraph with cumulative language: during/period patterns")
			}
			continue
		}

		// However, allow "On [date]" patterns even if they contain some cumulative keywords
		if !strings.Contains(lowerText, "on ") || !regexp.MustCompile(`(?i)\bon\s+\w+\s+\d+,?\s+\d{4}`).MatchString(para.Text) {
			// Only apply additional cumulative checks if it's not an "On [date]" pattern
			if strings.Contains(lowerText, "aggregate") ||
				strings.Contains(lowerText, "cumulative") ||
				strings.Contains(lowerText, "since") && strings.Contains(lowerText, "through") ||
				strings.Contains(lowerText, "from") && strings.Contains(lowerText, "to") {
				if p.verbose {
					log.Printf("Skipping paragraph with cumulative language: aggregate/cumulative terms")
				}
				continue
			}
		}

		// Look for purchase patterns
		btcMatches := purchasePattern.FindStringSubmatch(para.Text)
		if len(btcMatches) < 2 {
			continue
		}

		btcAmountStr := strings.ReplaceAll(btcMatches[1], ",", "")
		btcAmount, err := strconv.ParseFloat(btcAmountStr, 64)
		if err != nil {
			continue
		}

		// Look for USD amount
		var usdAmount float64
		amountMatches := amountPattern.FindStringSubmatch(para.Text)
		if len(amountMatches) >= 2 {
			usdAmountStr := strings.ReplaceAll(amountMatches[1], ",", "")
			usdAmount, _ = strconv.ParseFloat(usdAmountStr, 64)

			// Handle million/billion multipliers
			if strings.Contains(strings.ToLower(amountMatches[0]), "million") ||
				strings.Contains(strings.ToLower(amountMatches[0]), "m") {
				usdAmount *= 1000000
			} else if strings.Contains(strings.ToLower(amountMatches[0]), "billion") ||
				strings.Contains(strings.ToLower(amountMatches[0]), "b") {
				usdAmount *= 1000000000
			}
		}

		// Calculate average price
		var avgPrice float64
		if btcAmount > 0 && usdAmount > 0 {
			avgPrice = usdAmount / btcAmount
		}

		// Parse date (basic extraction)
		var transactionDate time.Time
		dateMatches := datePattern.FindString(para.Text)
		if dateMatches != "" {
			// Try to parse the date - this is a simplified version
			transactionDate = time.Now() // Fallback to current time
		} else {
			transactionDate = time.Now() // Fallback to current time
		}

		transaction := models.BitcoinTransaction{
			Date:            transactionDate,
			FilingType:      filing.FilingType,
			FilingURL:       filing.URL,
			BTCPurchased:    btcAmount,
			USDSpent:        usdAmount,
			AvgPriceUSD:     avgPrice,
			ExtractedText:   para.Text,
			ConfidenceScore: 0.7, // Medium confidence for regex extraction
		}

		transactions = append(transactions, transaction)

		if p.verbose {
			log.Printf("Regex extracted: %s - %.0f BTC for $%.0f", dateMatches, btcAmount, usdAmount)
		}
	}

	return transactions
}

// extractSharesFromParagraphs looks for shares outstanding information in the paragraphs
func (p *EnhancedParser) extractSharesFromParagraphs(paragraphs []BitcoinParagraph, filing models.Filing) *models.SharesOutstandingRecord {
	sharesPattern := regexp.MustCompile(`(?i)(\d+(?:,\d{3})*(?:\.\d+)?)\s+(?:million\s+)?(?:shares?\s+)?(?:outstanding|issued|common\s+stock)`)

	for _, para := range paragraphs {
		if !strings.Contains(strings.ToLower(para.Text), "share") {
			continue
		}

		matches := sharesPattern.FindStringSubmatch(para.Text)
		if len(matches) >= 2 {
			sharesStr := strings.ReplaceAll(matches[1], ",", "")
			sharesCount, err := strconv.ParseFloat(sharesStr, 64)
			if err != nil {
				continue
			}

			// Handle million multiplier
			if strings.Contains(strings.ToLower(matches[0]), "million") {
				sharesCount *= 1000000
			}

			record := &models.SharesOutstandingRecord{
				Date:            time.Now(), // Would need better date extraction
				FilingType:      filing.FilingType,
				FilingURL:       filing.URL,
				AccessionNumber: filing.AccessionNumber,
				CommonShares:    sharesCount,
				TotalShares:     sharesCount,
				ExtractedText:   para.Text,
				ConfidenceScore: 0.7, // Medium confidence for regex extraction
			}

			if p.verbose {
				log.Printf("Extracted shares: %.0f", sharesCount)
			}

			return record
		}
	}

	return nil
}

// GetStats returns statistics about the enhanced parser configuration
func (p *EnhancedParser) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"grok_enabled":    p.grokClient != nil,
		"grok_configured": false,
		"verbose":         p.verbose,
		"parser_type":     "enhanced",
	}

	if p.grokClient != nil {
		stats["grok_configured"] = p.grokClient.IsConfigured()
	}

	return stats
}
