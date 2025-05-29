package parser

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/interpretation/grok"
	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

// EnhancedParser combines regex-based parsing with Grok AI for maximum accuracy
type EnhancedParser struct {
	sharesParser *SharesParser
	grokClient   *grok.Client
	useGrok      bool
	verbose      bool
}

// NewEnhancedParser creates a new enhanced parser with optional Grok integration
func NewEnhancedParser(useGrok bool, verbose bool) *EnhancedParser {
	var grokClient *grok.Client
	if useGrok {
		grokClient = grok.NewClient()
		if !grokClient.IsConfigured() {
			log.Println("Warning: Grok API key not configured, falling back to regex-only mode")
			useGrok = false
		}
	}

	return &EnhancedParser{
		sharesParser: NewSharesParser(),
		grokClient:   grokClient,
		useGrok:      useGrok,
		verbose:      verbose,
	}
}

// ParseBitcoinTransactions extracts Bitcoin transactions using hybrid regex + Grok approach
func (p *EnhancedParser) ParseBitcoinTransactions(content io.Reader, filing models.Filing) ([]models.BitcoinTransaction, error) {
	// Read content
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	// First, try regex-based parsing (fast)
	regexTransactions, err := ParseBitcoinTransactions(contentBytes, filing)
	if err != nil {
		if p.verbose {
			log.Printf("Regex parsing error: %v", err)
		}
		regexTransactions = []models.BitcoinTransaction{} // Continue with empty results
	}

	if p.verbose {
		log.Printf("Regex parser found %d Bitcoin transactions", len(regexTransactions))
	}

	// If regex found transactions or Grok is disabled, return regex results
	if len(regexTransactions) > 0 || !p.useGrok {
		return regexTransactions, nil
	}

	// If regex found nothing and Grok is enabled, try Grok analysis
	if p.verbose {
		log.Printf("No transactions found by regex, trying Grok AI analysis...")
	}

	startTime := time.Now()
	grokTransactions, err := p.grokClient.ExtractBitcoinTransactions(string(contentBytes), filing)
	grokDuration := time.Since(startTime)

	if err != nil {
		if p.verbose {
			log.Printf("Grok analysis failed (%v), falling back to regex results: %v", grokDuration, err)
		}
		return regexTransactions, nil // Fall back to regex results (even if empty)
	}

	if p.verbose {
		log.Printf("Grok analysis completed in %v, found %d transactions", grokDuration, len(grokTransactions))
	}

	// Return Grok results if available
	return grokTransactions, nil
}

// ParseSharesOutstanding extracts shares outstanding using hybrid regex + Grok approach
func (p *EnhancedParser) ParseSharesOutstanding(content io.Reader, filing models.Filing) (*models.SharesOutstandingRecord, error) {
	// Read content
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	// First, try regex-based parsing (fast)
	regexShares, err := p.sharesParser.ExtractSharesFromFiling(contentBytes, filing)
	if err != nil {
		if p.verbose {
			log.Printf("Regex shares parsing error: %v", err)
		}
		regexShares = nil // Continue with empty results
	}

	if p.verbose {
		if regexShares != nil {
			log.Printf("Regex parser found shares data: %.0f common shares", regexShares.CommonShares)
		} else {
			log.Printf("Regex parser found no shares data")
		}
	}

	// If regex found shares or Grok is disabled, return regex results
	if regexShares != nil || !p.useGrok {
		return regexShares, nil
	}

	// If regex found nothing and Grok is enabled, try Grok analysis
	if p.verbose {
		log.Printf("No shares data found by regex, trying Grok AI analysis...")
	}

	startTime := time.Now()
	grokShares, err := p.grokClient.ExtractSharesOutstanding(string(contentBytes), filing)
	grokDuration := time.Since(startTime)

	if err != nil {
		if p.verbose {
			log.Printf("Grok shares analysis failed (%v), falling back to regex results: %v", grokDuration, err)
		}
		return regexShares, nil // Fall back to regex results (even if nil)
	}

	if p.verbose {
		if grokShares != nil {
			log.Printf("Grok shares analysis completed in %v, found %.0f common shares", grokDuration, grokShares.CommonShares)
		} else {
			log.Printf("Grok shares analysis completed in %v, found no shares data", grokDuration)
		}
	}

	// Return Grok results if available
	return grokShares, nil
}

// ParseFiling performs comprehensive parsing of a filing for both Bitcoin transactions and shares
func (p *EnhancedParser) ParseFiling(content io.Reader, filing models.Filing) (*models.FilingParseResult, error) {
	// Read content once
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	result := &models.FilingParseResult{
		Filing:              filing,
		BitcoinTransactions: []models.BitcoinTransaction{},
		SharesOutstanding:   nil,
		ParsedAt:            time.Now(),
		ParsingMethod:       "Enhanced Parser",
		ProcessingTimeMs:    0,
		Errors:              []string{},
	}

	startTime := time.Now()

	// Parse Bitcoin transactions
	bitcoinTransactions, err := p.ParseBitcoinTransactions(strings.NewReader(string(contentBytes)), filing)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Bitcoin parsing error: %v", err))
	} else {
		result.BitcoinTransactions = bitcoinTransactions
	}

	// Parse shares outstanding
	sharesRecord, err := p.ParseSharesOutstanding(strings.NewReader(string(contentBytes)), filing)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Shares parsing error: %v", err))
	} else {
		result.SharesOutstanding = sharesRecord
	}

	result.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())

	// Update parsing method based on what was used
	if p.useGrok && p.grokClient.IsConfigured() {
		result.ParsingMethod = "Enhanced Parser (Regex + Grok AI)"
	} else {
		result.ParsingMethod = "Enhanced Parser (Regex only)"
	}

	return result, nil
}

// GetStats returns statistics about the enhanced parser configuration
func (p *EnhancedParser) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"grok_enabled":    p.useGrok,
		"grok_configured": false,
		"verbose":         p.verbose,
		"parser_type":     "enhanced",
	}

	if p.grokClient != nil {
		stats["grok_configured"] = p.grokClient.IsConfigured()
	}

	return stats
}
