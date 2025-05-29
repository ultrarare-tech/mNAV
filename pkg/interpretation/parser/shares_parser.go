package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

// SharesParser handles extraction of shares outstanding data from SEC filings
type SharesParser struct {
	// Patterns for finding shares outstanding information
	patterns map[string]*regexp.Regexp
}

// NewSharesParser creates a new shares parser
func NewSharesParser() *SharesParser {
	return &SharesParser{
		patterns: initializeSharesPatterns(),
	}
}

// initializeSharesPatterns creates regex patterns for finding shares outstanding
func initializeSharesPatterns() map[string]*regexp.Regexp {
	patterns := map[string]*regexp.Regexp{
		// Common patterns in 10-Q/10-K filings
		"common_outstanding": regexp.MustCompile(`(?i)common\s+stock.*?outstanding.*?([0-9,]+(?:\.[0-9]+)?)\s*(?:shares)?`),
		"shares_outstanding": regexp.MustCompile(`(?i)shares\s+outstanding.*?([0-9,]+(?:\.[0-9]+)?)`),
		"outstanding_shares": regexp.MustCompile(`(?i)([0-9,]+(?:\.[0-9]+)?)\s*shares.*?outstanding`),
		"as_of_date":         regexp.MustCompile(`(?i)as\s+of\s+([A-Za-z]+\s+\d{1,2},?\s+\d{4})`),

		// Balance sheet patterns
		"balance_sheet_shares": regexp.MustCompile(`(?i)common\s+stock.*?shares.*?outstanding.*?([0-9,]+)`),
		"weighted_average":     regexp.MustCompile(`(?i)weighted\s+average.*?shares.*?outstanding.*?([0-9,]+)`),

		// Cover page patterns (often in 10-Q/10-K)
		"cover_page": regexp.MustCompile(`(?i)Common\s+Stock.*?Outstanding.*?([0-9,]+)\s*shares`),

		// Table patterns
		"table_shares": regexp.MustCompile(`(?i)<td[^>]*>.*?shares\s+outstanding.*?<td[^>]*>([0-9,]+)`),
	}

	return patterns
}

// ExtractSharesFromFiling extracts shares outstanding data from a filing
func (p *SharesParser) ExtractSharesFromFiling(body []byte, filing models.Filing) (*models.SharesOutstandingRecord, error) {
	// Try HTML parsing first
	if strings.HasSuffix(filing.DocumentURL, ".htm") || strings.HasSuffix(filing.DocumentURL, ".html") {
		record, err := p.extractFromHTML(body, filing)
		if err == nil && record != nil {
			return record, nil
		}
	}

	// Fall back to text parsing
	return p.extractFromText(string(body), filing)
}

// extractFromHTML extracts shares data from HTML documents
func (p *SharesParser) extractFromHTML(body []byte, filing models.Filing) (*models.SharesOutstandingRecord, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	var bestMatch *models.SharesOutstandingRecord
	highestConfidence := 0.0

	// Look for shares in different sections
	sections := []string{
		// Common section identifiers
		"Cover Page",
		"Balance Sheet",
		"Consolidated Balance Sheet",
		"Notes to Financial Statements",
		"Capital Stock",
		"Stockholders' Equity",
		"Equity",
	}

	for _, section := range sections {
		// Find sections by heading
		doc.Find("h1, h2, h3, h4, b, strong").Each(func(i int, s *goquery.Selection) {
			heading := strings.TrimSpace(s.Text())
			if strings.Contains(strings.ToLower(heading), strings.ToLower(section)) {
				// Look for shares data near this heading
				parent := s.Parent()
				if parent.Length() == 0 {
					parent = s
				}

				// Get text from the section
				sectionText := parent.Text()

				// Try to extract shares from this section
				if record := p.extractSharesFromText(sectionText, filing, section); record != nil {
					if record.ConfidenceScore > highestConfidence {
						bestMatch = record
						highestConfidence = record.ConfidenceScore
					}
				}
			}
		})
	}

	// Also look in tables
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		tableText := table.Text()
		if strings.Contains(strings.ToLower(tableText), "shares outstanding") ||
			strings.Contains(strings.ToLower(tableText), "common stock") {
			if record := p.extractSharesFromTable(table, filing); record != nil {
				if record.ConfidenceScore > highestConfidence {
					bestMatch = record
					highestConfidence = record.ConfidenceScore
				}
			}
		}
	})

	if bestMatch != nil {
		return bestMatch, nil
	}

	return nil, fmt.Errorf("no shares outstanding data found in HTML")
}

// extractFromText extracts shares data from plain text
func (p *SharesParser) extractFromText(text string, filing models.Filing) (*models.SharesOutstandingRecord, error) {
	// Try each pattern
	for patternName, pattern := range p.patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				sharesStr := strings.ReplaceAll(match[1], ",", "")
				shares, err := strconv.ParseFloat(sharesStr, 64)
				if err != nil {
					continue
				}

				// Sanity check - shares should be a reasonable number
				if shares < 1000 || shares > 100000000000 { // Less than 1k or more than 100B is suspicious
					continue
				}

				// Extract surrounding context
				startIdx := strings.Index(text, match[0])
				contextStart := startIdx - 200
				if contextStart < 0 {
					contextStart = 0
				}
				contextEnd := startIdx + len(match[0]) + 200
				if contextEnd > len(text) {
					contextEnd = len(text)
				}
				context := text[contextStart:contextEnd]

				// Look for date
				asOfDate := p.extractAsOfDate(context)
				if asOfDate.IsZero() {
					asOfDate = filing.FilingDate
				}

				// Calculate confidence score
				confidence := p.calculateConfidence(patternName, context, shares)

				record := &models.SharesOutstandingRecord{
					Date:            asOfDate,
					FilingType:      filing.FilingType,
					FilingURL:       filing.URL,
					AccessionNumber: filing.AccessionNumber,
					CommonShares:    shares,
					TotalShares:     shares, // Will be updated if we find preferred shares
					ExtractedFrom:   "Text Pattern: " + patternName,
					ExtractedText:   strings.TrimSpace(match[0]),
					ConfidenceScore: confidence,
				}

				// Look for preferred shares nearby
				preferredPattern := regexp.MustCompile(`(?i)preferred\s+stock.*?outstanding.*?([0-9,]+)`)
				if prefMatches := preferredPattern.FindStringSubmatch(context); len(prefMatches) >= 2 {
					prefStr := strings.ReplaceAll(prefMatches[1], ",", "")
					if prefShares, err := strconv.ParseFloat(prefStr, 64); err == nil {
						record.PreferredShares = prefShares
						record.TotalShares = record.CommonShares + record.PreferredShares
					}
				}

				return record, nil
			}
		}
	}

	return nil, fmt.Errorf("no shares outstanding data found in text")
}

// extractSharesFromText helper method
func (p *SharesParser) extractSharesFromText(text string, filing models.Filing, section string) *models.SharesOutstandingRecord {
	for patternName, pattern := range p.patterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				sharesStr := strings.ReplaceAll(match[1], ",", "")
				shares, err := strconv.ParseFloat(sharesStr, 64)
				if err != nil || shares < 1000 || shares > 100000000000 {
					continue
				}

				asOfDate := p.extractAsOfDate(text)
				if asOfDate.IsZero() {
					asOfDate = filing.FilingDate
				}

				return &models.SharesOutstandingRecord{
					Date:            asOfDate,
					FilingType:      filing.FilingType,
					FilingURL:       filing.URL,
					AccessionNumber: filing.AccessionNumber,
					CommonShares:    shares,
					TotalShares:     shares,
					ExtractedFrom:   section,
					ExtractedText:   strings.TrimSpace(match[0]),
					ConfidenceScore: p.calculateConfidence(patternName, text, shares),
				}
			}
		}
	}
	return nil
}

// extractSharesFromTable extracts shares from HTML tables
func (p *SharesParser) extractSharesFromTable(table *goquery.Selection, filing models.Filing) *models.SharesOutstandingRecord {
	var sharesValue float64
	var foundShares bool
	var extractedText string

	// Look for rows containing shares outstanding
	table.Find("tr").Each(func(i int, row *goquery.Selection) {
		rowText := strings.ToLower(row.Text())
		if strings.Contains(rowText, "shares outstanding") ||
			strings.Contains(rowText, "common stock outstanding") {
			// Look for numbers in the same row
			row.Find("td").Each(func(j int, cell *goquery.Selection) {
				cellText := strings.TrimSpace(cell.Text())
				// Try to parse as number
				cleaned := strings.ReplaceAll(cellText, ",", "")
				cleaned = strings.ReplaceAll(cleaned, "$", "")
				if val, err := strconv.ParseFloat(cleaned, 64); err == nil && val > 1000 {
					sharesValue = val
					foundShares = true
					extractedText = rowText
				}
			})
		}
	})

	if foundShares {
		return &models.SharesOutstandingRecord{
			Date:            filing.FilingDate,
			FilingType:      filing.FilingType,
			FilingURL:       filing.URL,
			AccessionNumber: filing.AccessionNumber,
			CommonShares:    sharesValue,
			TotalShares:     sharesValue,
			ExtractedFrom:   "Table",
			ExtractedText:   extractedText,
			ConfidenceScore: 0.9, // High confidence for table data
		}
	}

	return nil
}

// extractAsOfDate extracts the "as of" date from text
func (p *SharesParser) extractAsOfDate(text string) time.Time {
	// Pattern for dates like "March 31, 2024" or "December 31, 2023"
	datePattern := regexp.MustCompile(`(?i)as\s+of\s+([A-Za-z]+\s+\d{1,2},?\s+\d{4})`)
	matches := datePattern.FindStringSubmatch(text)
	if len(matches) >= 2 {
		// Parse the date
		layouts := []string{
			"January 2, 2006",
			"January 2 2006",
			"Jan 2, 2006",
			"Jan 2 2006",
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, matches[1]); err == nil {
				return t
			}
		}
	}

	// Try ISO date format
	isoPattern := regexp.MustCompile(`(?i)as\s+of\s+(\d{4}-\d{2}-\d{2})`)
	if matches := isoPattern.FindStringSubmatch(text); len(matches) >= 2 {
		if t, err := time.Parse("2006-01-02", matches[1]); err == nil {
			return t
		}
	}

	return time.Time{}
}

// calculateConfidence calculates confidence score for extracted data
func (p *SharesParser) calculateConfidence(patternName, context string, shares float64) float64 {
	confidence := 0.5 // Base confidence

	// Increase confidence for specific patterns
	switch patternName {
	case "common_outstanding", "cover_page":
		confidence += 0.3
	case "balance_sheet_shares":
		confidence += 0.25
	case "table_shares":
		confidence += 0.35
	}

	// Check for context clues
	contextLower := strings.ToLower(context)

	// Positive indicators
	if strings.Contains(contextLower, "as of") {
		confidence += 0.1
	}
	if strings.Contains(contextLower, "balance sheet") {
		confidence += 0.1
	}
	if strings.Contains(contextLower, "common stock") && strings.Contains(contextLower, "outstanding") {
		confidence += 0.1
	}

	// Negative indicators
	if strings.Contains(contextLower, "authorized") {
		confidence -= 0.2 // Authorized shares, not outstanding
	}
	if strings.Contains(contextLower, "reserved") {
		confidence -= 0.2 // Reserved shares, not outstanding
	}
	if strings.Contains(contextLower, "weighted average") {
		confidence -= 0.1 // Weighted average is different from actual outstanding
	}

	// Sanity check on the number
	if shares < 1000000 { // Less than 1M shares is unusual for public companies
		confidence -= 0.1
	}
	if shares > 10000000000 { // More than 10B shares is unusual
		confidence -= 0.1
	}

	// Ensure confidence is between 0 and 1
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}
