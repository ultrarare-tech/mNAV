package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/shared/models"
)

// SaylorTransaction represents a transaction from SaylorTracker
type SaylorTransaction struct {
	Date        string  `json:"date"`
	BTCAmount   float64 `json:"btc_amount"`
	USDAmount   float64 `json:"usd_amount"`
	AvgPrice    float64 `json:"avg_price"`
	Source      string  `json:"source"`
	Description string  `json:"description"`
}

// FilingAnalysis represents analysis of a specific filing
type FilingAnalysis struct {
	FilingPath     string                    `json:"filing_path"`
	FilingType     string                    `json:"filing_type"`
	FilingDate     string                    `json:"filing_date"`
	ParsedResult   *models.FilingParseResult `json:"parsed_result"`
	MatchedSaylor  []SaylorTransaction       `json:"matched_saylor"`
	Classification string                    `json:"classification"` // "correct", "cumulative", "incorrect"
	Confidence     float64                   `json:"confidence"`
	Reasoning      string                    `json:"reasoning"`
	Paragraphs     []ParagraphAnalysis       `json:"paragraphs"`
}

// ParagraphAnalysis represents analysis of Bitcoin-related paragraphs
type ParagraphAnalysis struct {
	Text           string  `json:"text"`
	Context        string  `json:"context"`
	Classification string  `json:"classification"` // "individual_transaction", "cumulative_total", "pricing_info"
	Confidence     float64 `json:"confidence"`
	Reasoning      string  `json:"reasoning"`
}

// ValidationResult represents the complete validation analysis
type ValidationResult struct {
	GeneratedAt       time.Time           `json:"generated_at"`
	SaylorData        []SaylorTransaction `json:"saylor_data"`
	FilingAnalyses    []FilingAnalysis    `json:"filing_analyses"`
	CorrectFilings    []FilingAnalysis    `json:"correct_filings"`
	CumulativeFilings []FilingAnalysis    `json:"cumulative_filings"`
	Summary           ValidationSummary   `json:"summary"`
}

// ValidationSummary provides summary statistics
type ValidationSummary struct {
	TotalFilings      int     `json:"total_filings"`
	CorrectFilings    int     `json:"correct_filings"`
	CumulativeFilings int     `json:"cumulative_filings"`
	IncorrectFilings  int     `json:"incorrect_filings"`
	MatchRate         float64 `json:"match_rate"`
}

func main() {
	fmt.Printf("🔍 SAYLOR TRACKER VALIDATION ANALYSIS\n")
	fmt.Printf("=====================================\n\n")

	// Load SaylorTracker reference data
	saylorData, err := loadSaylorTrackerData()
	if err != nil {
		log.Fatalf("❌ Error loading SaylorTracker data: %v", err)
	}

	fmt.Printf("📊 Loaded %d SaylorTracker transactions\n", len(saylorData))

	// Analyze all parsed filings
	filingAnalyses, err := analyzeAllFilings(saylorData)
	if err != nil {
		log.Fatalf("❌ Error analyzing filings: %v", err)
	}

	// Classify filings
	correctFilings, cumulativeFilings := classifyFilings(filingAnalyses)

	// Create validation result
	result := ValidationResult{
		GeneratedAt:       time.Now(),
		SaylorData:        saylorData,
		FilingAnalyses:    filingAnalyses,
		CorrectFilings:    correctFilings,
		CumulativeFilings: cumulativeFilings,
		Summary: ValidationSummary{
			TotalFilings:      len(filingAnalyses),
			CorrectFilings:    len(correctFilings),
			CumulativeFilings: len(cumulativeFilings),
			IncorrectFilings:  len(filingAnalyses) - len(correctFilings) - len(cumulativeFilings),
			MatchRate:         float64(len(correctFilings)) / float64(len(filingAnalyses)) * 100,
		},
	}

	// Display results
	displayValidationResults(result)

	// Save results
	if err := saveValidationResults(result, "data/analysis/saylor_validation.json"); err != nil {
		log.Printf("⚠️  Warning: Could not save validation results: %v", err)
	} else {
		fmt.Printf("\n💾 Validation results saved to: data/analysis/saylor_validation.json\n")
	}

	// Generate training data for prompt improvement
	if err := generateTrainingData(result); err != nil {
		log.Printf("⚠️  Warning: Could not generate training data: %v", err)
	} else {
		fmt.Printf("💾 Training data saved to: data/analysis/training_data.json\n")
	}
}

func loadSaylorTrackerData() ([]SaylorTransaction, error) {
	// For now, create reference data based on known SaylorTracker information
	// In a real implementation, this would load from SaylorTracker API or CSV
	saylorData := []SaylorTransaction{
		{Date: "2020-08-11", BTCAmount: 21454, USDAmount: 250000000, AvgPrice: 11652, Source: "SaylorTracker", Description: "Initial Bitcoin purchase"},
		{Date: "2020-09-14", BTCAmount: 16796, USDAmount: 175000000, AvgPrice: 10419, Source: "SaylorTracker", Description: "Second major purchase"},
		{Date: "2020-12-04", BTCAmount: 2574, USDAmount: 50000000, AvgPrice: 19427, Source: "SaylorTracker", Description: "December purchase"},
		{Date: "2020-12-21", BTCAmount: 29646, USDAmount: 650000000, AvgPrice: 21925, Source: "SaylorTracker", Description: "Large December purchase"},
		{Date: "2021-01-22", BTCAmount: 314, USDAmount: 10000000, AvgPrice: 31808, Source: "SaylorTracker", Description: "January 2021 purchase"},
		{Date: "2021-02-24", BTCAmount: 19452, USDAmount: 1026000000, AvgPrice: 52765, Source: "SaylorTracker", Description: "February 2021 purchase"},
		{Date: "2021-03-05", BTCAmount: 328, USDAmount: 15000000, AvgPrice: 45732, Source: "SaylorTracker", Description: "March 2021 purchase"},
		{Date: "2021-03-12", BTCAmount: 262, USDAmount: 15000000, AvgPrice: 57146, Source: "SaylorTracker", Description: "March 2021 purchase"},
		// Add more known transactions from SaylorTracker...
	}

	return saylorData, nil
}

func analyzeAllFilings(saylorData []SaylorTransaction) ([]FilingAnalysis, error) {
	var analyses []FilingAnalysis

	// Walk through all parsed files
	err := filepath.Walk("data/parsed", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, "_parsed.json") {
			return nil
		}

		// Load parsed result
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var result models.FilingParseResult
		if err := json.Unmarshal(data, &result); err != nil {
			return err
		}

		// Analyze this filing
		analysis := analyzeFilingAgainstSaylor(path, &result, saylorData)
		analyses = append(analyses, analysis)

		return nil
	})

	return analyses, err
}

func analyzeFilingAgainstSaylor(filePath string, result *models.FilingParseResult, saylorData []SaylorTransaction) FilingAnalysis {
	analysis := FilingAnalysis{
		FilingPath:   filePath,
		FilingType:   result.Filing.FilingType,
		FilingDate:   result.Filing.FilingDate.Format("2006-01-02"),
		ParsedResult: result,
	}

	// Find matching SaylorTracker transactions
	var matchedSaylor []SaylorTransaction
	for _, tx := range result.BitcoinTransactions {
		for _, saylor := range saylorData {
			if isTransactionMatch(tx, saylor) {
				matchedSaylor = append(matchedSaylor, saylor)
			}
		}
	}

	analysis.MatchedSaylor = matchedSaylor

	// Classify the filing
	if len(matchedSaylor) > 0 && len(result.BitcoinTransactions) > 0 {
		analysis.Classification = "correct"
		analysis.Confidence = 0.9
		analysis.Reasoning = fmt.Sprintf("Found %d transactions matching SaylorTracker data", len(matchedSaylor))
	} else if len(result.BitcoinTransactions) > 0 {
		// Check if this looks like cumulative data
		if containsCumulativeLanguage(result) {
			analysis.Classification = "cumulative"
			analysis.Confidence = 0.8
			analysis.Reasoning = "Contains transactions but no SaylorTracker matches, likely cumulative totals"
		} else {
			analysis.Classification = "incorrect"
			analysis.Confidence = 0.7
			analysis.Reasoning = "Contains transactions but no SaylorTracker matches, classification unclear"
		}
	} else {
		analysis.Classification = "no_data"
		analysis.Confidence = 1.0
		analysis.Reasoning = "No Bitcoin transactions found in filing"
	}

	// Analyze paragraphs (would need to re-parse the original filing to get paragraph details)
	analysis.Paragraphs = analyzeParagraphs(result)

	return analysis
}

func isTransactionMatch(tx models.BitcoinTransaction, saylor SaylorTransaction) bool {
	// Parse saylor date
	saylorDate, err := time.Parse("2006-01-02", saylor.Date)
	if err != nil {
		return false
	}

	// Check date match (within 7 days)
	daysDiff := tx.Date.Sub(saylorDate).Hours() / 24
	if daysDiff < 0 {
		daysDiff = -daysDiff
	}
	if daysDiff > 7 {
		return false
	}

	// Check BTC amount match (within 5%)
	btcDiff := (tx.BTCPurchased - saylor.BTCAmount) / saylor.BTCAmount
	if btcDiff < 0 {
		btcDiff = -btcDiff
	}
	if btcDiff > 0.05 {
		return false
	}

	return true
}

func containsCumulativeLanguage(result *models.FilingParseResult) bool {
	for _, tx := range result.BitcoinTransactions {
		text := strings.ToLower(tx.ExtractedText)
		if strings.Contains(text, "during the period") ||
			strings.Contains(text, "between") && strings.Contains(text, "and") ||
			strings.Contains(text, "aggregate") ||
			strings.Contains(text, "total") ||
			strings.Contains(text, "cumulative") {
			return true
		}
	}
	return false
}

func analyzeParagraphs(result *models.FilingParseResult) []ParagraphAnalysis {
	var paragraphs []ParagraphAnalysis

	for _, tx := range result.BitcoinTransactions {
		paragraph := ParagraphAnalysis{
			Text: tx.ExtractedText,
		}

		// Classify based on content
		text := strings.ToLower(tx.ExtractedText)
		if strings.Contains(text, "during the period") || strings.Contains(text, "between") && strings.Contains(text, "and") {
			paragraph.Classification = "cumulative_total"
			paragraph.Confidence = 0.9
			paragraph.Reasoning = "Contains date range language indicating cumulative total"
		} else if strings.Contains(text, "on ") && (strings.Contains(text, "purchased") || strings.Contains(text, "acquired")) {
			paragraph.Classification = "individual_transaction"
			paragraph.Confidence = 0.8
			paragraph.Reasoning = "Contains specific date and transaction language"
		} else {
			paragraph.Classification = "unclear"
			paragraph.Confidence = 0.5
			paragraph.Reasoning = "Classification unclear from text analysis"
		}

		paragraphs = append(paragraphs, paragraph)
	}

	return paragraphs
}

func classifyFilings(analyses []FilingAnalysis) ([]FilingAnalysis, []FilingAnalysis) {
	var correctFilings, cumulativeFilings []FilingAnalysis

	for _, analysis := range analyses {
		if analysis.Classification == "correct" {
			correctFilings = append(correctFilings, analysis)
		} else if analysis.Classification == "cumulative" {
			cumulativeFilings = append(cumulativeFilings, analysis)
		}
	}

	return correctFilings, cumulativeFilings
}

func displayValidationResults(result ValidationResult) {
	fmt.Printf("📊 VALIDATION RESULTS\n")
	fmt.Printf("=====================\n\n")

	fmt.Printf("📋 SUMMARY:\n")
	fmt.Printf("   Total Filings Analyzed: %d\n", result.Summary.TotalFilings)
	fmt.Printf("   Correct Filings: %d\n", result.Summary.CorrectFilings)
	fmt.Printf("   Cumulative Filings: %d\n", result.Summary.CumulativeFilings)
	fmt.Printf("   Incorrect/Unclear: %d\n", result.Summary.IncorrectFilings)
	fmt.Printf("   Match Rate: %.1f%%\n\n", result.Summary.MatchRate)

	fmt.Printf("✅ CORRECT FILINGS (match SaylorTracker):\n")
	for i, filing := range result.CorrectFilings {
		if i >= 10 { // Limit display
			fmt.Printf("   ... and %d more\n", len(result.CorrectFilings)-10)
			break
		}
		fmt.Printf("   %s (%s): %d transactions, %d matches\n",
			filepath.Base(filing.FilingPath), filing.FilingType,
			len(filing.ParsedResult.BitcoinTransactions), len(filing.MatchedSaylor))
	}

	fmt.Printf("\n⚠️  CUMULATIVE FILINGS (no SaylorTracker matches):\n")
	for i, filing := range result.CumulativeFilings {
		if i >= 10 { // Limit display
			fmt.Printf("   ... and %d more\n", len(result.CumulativeFilings)-10)
			break
		}
		fmt.Printf("   %s (%s): %d transactions\n",
			filepath.Base(filing.FilingPath), filing.FilingType,
			len(filing.ParsedResult.BitcoinTransactions))
	}
}

func saveValidationResults(result ValidationResult, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func generateTrainingData(result ValidationResult) error {
	trainingData := struct {
		CorrectExamples    []string  `json:"correct_examples"`
		CumulativeExamples []string  `json:"cumulative_examples"`
		GeneratedAt        time.Time `json:"generated_at"`
		Instructions       string    `json:"instructions"`
	}{
		GeneratedAt:  time.Now(),
		Instructions: "Use these examples to train a better classification prompt for distinguishing individual Bitcoin transactions from cumulative totals",
	}

	// Extract examples from correct filings
	for _, filing := range result.CorrectFilings {
		for _, paragraph := range filing.Paragraphs {
			if paragraph.Classification == "individual_transaction" {
				trainingData.CorrectExamples = append(trainingData.CorrectExamples, paragraph.Text)
			}
		}
	}

	// Extract examples from cumulative filings
	for _, filing := range result.CumulativeFilings {
		for _, paragraph := range filing.Paragraphs {
			if paragraph.Classification == "cumulative_total" {
				trainingData.CumulativeExamples = append(trainingData.CumulativeExamples, paragraph.Text)
			}
		}
	}

	// Limit examples to avoid overwhelming the prompt
	if len(trainingData.CorrectExamples) > 20 {
		trainingData.CorrectExamples = trainingData.CorrectExamples[:20]
	}
	if len(trainingData.CumulativeExamples) > 20 {
		trainingData.CumulativeExamples = trainingData.CumulativeExamples[:20]
	}

	data, err := json.MarshalIndent(trainingData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("data/analysis/training_data.json", data, 0644)
}
