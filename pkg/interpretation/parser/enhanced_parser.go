package parser

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/interpretation/grok"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
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
func (p *EnhancedParser) ParseFiling(content, filingType string) (*models.FilingParseResult, error) {
	if p.verbose {
		log.Printf("Starting two-stage parsing for %s filing (%d chars)", filingType, len(content))
	}

	// Create a basic filing object for the result
	filing := models.Filing{
		FilingType: filingType,
	}

	result := &models.FilingParseResult{
		Filing:   filing,
		ParsedAt: time.Now(),
	}

	// Stage 1: Use regex to identify Bitcoin-related paragraphs with numerical values
	bitcoinParagraphs := p.extractBitcoinParagraphs(content)

	if p.verbose {
		log.Printf("Stage 1: Found %d Bitcoin-related paragraphs with numerical values", len(bitcoinParagraphs))
	}

	if len(bitcoinParagraphs) == 0 {
		if p.verbose {
			log.Printf("No Bitcoin-related paragraphs found, skipping Grok analysis")
		}
		return result, nil
	}

	// Stage 2: Send identified paragraphs to Grok for interpretation
	transactions, err := p.interpretParagraphsWithGrok(bitcoinParagraphs, filing)
	if err != nil {
		if p.verbose {
			log.Printf("Grok interpretation failed: %v", err)
		}
		// Fall back to regex-only parsing
		transactions = p.fallbackRegexParsing(bitcoinParagraphs)
	}

	result.BitcoinTransactions = transactions
	result.SharesOutstanding = p.extractSharesFromParagraphs(bitcoinParagraphs)

	if p.verbose {
		sharesCount := 0
		if result.SharesOutstanding != nil {
			sharesCount = 1
		}
		log.Printf("Parsing complete: found %d transactions, %d shares entries",
			len(result.BitcoinTransactions), sharesCount)
	}

	return result, nil
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
		if transactionKeywords.MatchString(paragraph) {
			context = "transaction"
		}

		// Additional context clues
		if regexp.MustCompile(`(?i)\b(total|aggregate|cumulative)\b`).MatchString(paragraph) {
			context = "cumulative"
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
			log.Printf("Found Bitcoin paragraph %d: %d numbers, context: %s", i, len(numbers), context)
			log.Printf("Preview: %.100s...", paragraph)
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
func (p *EnhancedParser) fallbackRegexParsing(paragraphs []BitcoinParagraph) []models.BitcoinTransaction {
	var transactions []models.BitcoinTransaction

	// Basic regex patterns for transaction extraction
	purchasePattern := regexp.MustCompile(`(?i)(?:purchased|acquired|bought)\s+(?:approximately\s+)?(\d+(?:,\d{3})*(?:\.\d+)?)\s+(?:bitcoin|btc)`)
	amountPattern := regexp.MustCompile(`\$\s*(\d+(?:,\d{3})*(?:\.\d+)?)\s*(?:million|billion|M|B)?`)
	datePattern := regexp.MustCompile(`(?:january|february|march|april|may|june|july|august|september|october|november|december|\d{1,2}[/-]\d{1,2}[/-]\d{2,4}|\d{4}-\d{2}-\d{2})`)

	for _, para := range paragraphs {
		// Skip paragraphs that seem to describe cumulative totals
		if strings.Contains(strings.ToLower(para.Text), "total") ||
			strings.Contains(strings.ToLower(para.Text), "aggregate") ||
			strings.Contains(strings.ToLower(para.Text), "cumulative") {
			continue
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
func (p *EnhancedParser) extractSharesFromParagraphs(paragraphs []BitcoinParagraph) *models.SharesOutstandingRecord {
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
