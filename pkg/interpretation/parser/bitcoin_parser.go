package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ultrarare-tech/mNAV/pkg/shared/models"
)

// ParseBitcoinTransactions parses content and extracts Bitcoin transactions
func ParseBitcoinTransactions(content []byte, filing models.Filing) ([]models.BitcoinTransaction, error) {
	// Determine if it's HTML or text content
	if strings.Contains(string(content[:min(1000, len(content))]), "<html") ||
		strings.Contains(string(content[:min(1000, len(content))]), "<HTML") {
		return parseHTMLDocument(content, filing)
	}
	return parseTextDocument(content, filing)
}

// parseHTMLDocument parses an HTML document for Bitcoin transactions
func parseHTMLDocument(body []byte, filing models.Filing) ([]models.BitcoinTransaction, error) {
	transactions := []models.BitcoinTransaction{}
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
			if len(paragraph) > 50 && containsBitcoinKeywords(paragraph) {
				bitcoinParagraphs = append(bitcoinParagraphs, paragraph)
			}
		}
	}

	// Extract transactions from the found paragraphs
	for _, paragraph := range bitcoinParagraphs {
		extracted := extractTransactionsFromText(paragraph, filing)
		for _, tx := range extracted {
			// Create a unique key for deduplication
			key := fmt.Sprintf("%.2f-%.2f-%s", tx.BTCPurchased, tx.USDSpent, tx.Date.Format("2006-01-02"))
			if !seenTransactions[key] && isValidTransaction(tx) {
				seenTransactions[key] = true
				transactions = append(transactions, tx)
			}
		}
	}

	return transactions, nil
}

// parseTextDocument parses a text document for Bitcoin transactions
func parseTextDocument(body []byte, filing models.Filing) ([]models.BitcoinTransaction, error) {
	text := string(body)

	// Split into paragraphs and process each one
	paragraphs := strings.Split(text, "\n")
	var transactions []models.BitcoinTransaction
	seenTransactions := make(map[string]bool)

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if len(paragraph) > 50 && containsBitcoinKeywords(paragraph) {
			extracted := extractTransactionsFromText(paragraph, filing)
			for _, tx := range extracted {
				key := fmt.Sprintf("%.2f-%.2f-%s", tx.BTCPurchased, tx.USDSpent, tx.Date.Format("2006-01-02"))
				if !seenTransactions[key] && isValidTransaction(tx) {
					seenTransactions[key] = true
					transactions = append(transactions, tx)
				}
			}
		}
	}

	return transactions, nil
}

// containsBitcoinKeywords checks if text contains Bitcoin-related keywords
func containsBitcoinKeywords(text string) bool {
	lowerText := strings.ToLower(text)

	// Primary Bitcoin keywords
	bitcoinKeywords := []string{
		"bitcoin", "btc", "cryptocurrency", "digital asset",
		"purchase", "acquired", "investment", "treasury",
	}

	// Must contain at least one Bitcoin-related term
	hasBitcoinTerm := false
	for _, keyword := range bitcoinKeywords[:4] { // First 4 are Bitcoin-specific
		if strings.Contains(lowerText, keyword) {
			hasBitcoinTerm = true
			break
		}
	}

	if !hasBitcoinTerm {
		return false
	}

	// Must also contain purchase/transaction indicators
	actionKeywords := []string{
		"purchase", "acquired", "investment", "bought", "treasury",
		"million", "aggregate", "proceeds",
	}

	for _, keyword := range actionKeywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}

	return false
}

// extractTransactionsFromText extracts Bitcoin transactions from text using regex patterns
func extractTransactionsFromText(text string, filing models.Filing) []models.BitcoinTransaction {
	var transactions []models.BitcoinTransaction

	// Skip financing activities
	if isFinancingActivity(text) {
		return transactions
	}

	// Pattern 1: "purchased X bitcoin for $Y million"
	pattern1 := regexp.MustCompile(`(?i)purchased?\s+(?:approximately\s+)?([0-9,]+(?:\.[0-9]+)?)\s+(?:additional\s+)?bitcoin.*?(?:for|at).*?\$([0-9,]+(?:\.[0-9]+)?)\s*million`)
	if matches := pattern1.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 3 {
				btc := parseNumber(match[1])
				usdMillions := parseNumber(match[2])

				if btc > 0 && usdMillions > 0 {
					tx := models.BitcoinTransaction{
						Date:            filing.FilingDate,
						FilingType:      filing.FilingType,
						FilingURL:       filing.DocumentURL,
						BTCPurchased:    btc,
						USDSpent:        usdMillions * 1000000, // Convert millions to dollars
						AvgPriceUSD:     (usdMillions * 1000000) / btc,
						ExtractedText:   text,
						ConfidenceScore: 0.9,
					}
					transactions = append(transactions, tx)
				}
			}
		}
	}

	// Pattern 2: "acquired Y bitcoin at an average price of $X"
	pattern2 := regexp.MustCompile(`(?i)acquired?\s+(?:approximately\s+)?([0-9,]+(?:\.[0-9]+)?)\s+bitcoin.*?average\s+price.*?\$([0-9,]+(?:\.[0-9]+)?)`)
	if matches := pattern2.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) >= 3 {
				btc := parseNumber(match[1])
				avgPrice := parseNumber(match[2])

				if btc > 0 && avgPrice > 0 {
					tx := models.BitcoinTransaction{
						Date:            filing.FilingDate,
						FilingType:      filing.FilingType,
						FilingURL:       filing.DocumentURL,
						BTCPurchased:    btc,
						USDSpent:        btc * avgPrice,
						AvgPriceUSD:     avgPrice,
						ExtractedText:   text,
						ConfidenceScore: 0.85,
					}
					transactions = append(transactions, tx)
				}
			}
		}
	}

	return transactions
}

// isFinancingActivity checks if the text describes financing activities rather than actual purchases
func isFinancingActivity(text string) bool {
	lowerText := strings.ToLower(text)

	financingTerms := []string{
		"convertible note", "bond", "debenture", "credit facility",
		"loan", "financing", "proceeds from", "debt offering",
		"equity offering", "stock offering", "warrant",
		"intends to", "plans to", "will use", "intended use",
	}

	for _, term := range financingTerms {
		if strings.Contains(lowerText, term) {
			return true
		}
	}

	return false
}

// parseNumber parses a number string, removing commas
func parseNumber(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	num, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return num
}

// isValidTransaction validates a Bitcoin transaction
func isValidTransaction(tx models.BitcoinTransaction) bool {
	return tx.BTCPurchased > 0 &&
		tx.USDSpent > 0 &&
		tx.AvgPriceUSD > 0 &&
		tx.AvgPriceUSD < 1000000 // Sanity check for price
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
