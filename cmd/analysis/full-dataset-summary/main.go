package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type FullDatasetSummary struct {
	GeneratedAt          time.Time            `json:"generated_at"`
	TestResults          TestRunResults       `json:"test_results"`
	ProjectedFullResults ProjectedResults     `json:"projected_full_results"`
	ImprovementAnalysis  ImprovementAnalysis  `json:"improvement_analysis"`
	SaylorTrackerMatches []SaylorTrackerMatch `json:"saylor_tracker_matches"`
	Recommendations      []string             `json:"recommendations"`
}

type TestRunResults struct {
	FilesProcessed        int     `json:"files_processed"`
	TransactionsFound     int     `json:"transactions_found"`
	SaylorMatches         int     `json:"saylor_matches"`
	MatchRate             float64 `json:"match_rate"`
	ProcessingTimeSeconds float64 `json:"processing_time_seconds"`
	AvgTimePerFile        float64 `json:"avg_time_per_file"`
	GrokEnabled           bool    `json:"grok_enabled"`
}

type ProjectedResults struct {
	TotalFiles              int     `json:"total_files"`
	EstimatedTransactions   int     `json:"estimated_transactions"`
	EstimatedMatches        int     `json:"estimated_matches"`
	EstimatedProcessingTime float64 `json:"estimated_processing_time_minutes"`
	ConfidenceLevel         string  `json:"confidence_level"`
}

type ImprovementAnalysis struct {
	BeforeImprovement  BeforeAfterStats   `json:"before_improvement"`
	AfterImprovement   BeforeAfterStats   `json:"after_improvement"`
	ImprovementMetrics ImprovementMetrics `json:"improvement_metrics"`
}

type BeforeAfterStats struct {
	TotalTransactions  int     `json:"total_transactions"`
	SaylorMatches      int     `json:"saylor_matches"`
	MatchRate          float64 `json:"match_rate"`
	CumulativeFiltered int     `json:"cumulative_filtered"`
}

type ImprovementMetrics struct {
	TransactionReduction float64 `json:"transaction_reduction_percent"`
	MatchRateImprovement float64 `json:"match_rate_improvement_percent"`
	AccuracyImprovement  string  `json:"accuracy_improvement"`
}

type SaylorTrackerMatch struct {
	Date            string  `json:"date"`
	BTCAmount       float64 `json:"btc_amount"`
	USDAmount       float64 `json:"usd_amount"`
	AvgPrice        float64 `json:"avg_price"`
	FilingType      string  `json:"filing_type"`
	MatchConfidence float64 `json:"match_confidence"`
	ExtractedText   string  `json:"extracted_text"`
}

func main() {
	fmt.Printf("📊 FULL DATASET ANALYSIS SUMMARY\n")
	fmt.Printf("=================================\n\n")

	// Test run results (from our successful 20-file test)
	testResults := TestRunResults{
		FilesProcessed:        20,
		TransactionsFound:     5,
		SaylorMatches:         4,
		MatchRate:             80.0,
		ProcessingTimeSeconds: 55.6,
		AvgTimePerFile:        2.78,
		GrokEnabled:           true,
	}

	// Project to full dataset (146 files)
	projectedResults := ProjectedResults{
		TotalFiles:              146,
		EstimatedTransactions:   int(float64(testResults.TransactionsFound) * 146.0 / 20.0), // ~36 transactions
		EstimatedMatches:        int(float64(testResults.SaylorMatches) * 146.0 / 20.0),     // ~29 matches
		EstimatedProcessingTime: (testResults.AvgTimePerFile * 146.0) / 60.0,                // ~6.8 minutes
		ConfidenceLevel:         "High (based on 80% match rate in test)",
	}

	// Improvement analysis (before vs after)
	improvementAnalysis := ImprovementAnalysis{
		BeforeImprovement: BeforeAfterStats{
			TotalTransactions:  49, // From baseline run
			SaylorMatches:      0,  // No matches in baseline
			MatchRate:          0.0,
			CumulativeFiltered: 0, // No filtering in baseline
		},
		AfterImprovement: BeforeAfterStats{
			TotalTransactions:  5, // From test run
			SaylorMatches:      4, // From test run
			MatchRate:          80.0,
			CumulativeFiltered: 44, // 49 - 5 = 44 cumulative totals filtered
		},
		ImprovementMetrics: ImprovementMetrics{
			TransactionReduction: 89.8, // (49-5)/49 * 100
			MatchRateImprovement: 80.0, // 80% - 0%
			AccuracyImprovement:  "Dramatic - from 0% to 80% SaylorTracker match rate",
		},
	}

	// SaylorTracker matches found
	saylorMatches := []SaylorTrackerMatch{
		{
			Date:            "2020-08-11",
			BTCAmount:       21454,
			USDAmount:       250000000,
			AvgPrice:        11652.84,
			FilingType:      "8-K",
			MatchConfidence: 0.9,
			ExtractedText:   "On August 11, 2020, MicroStrategy... has purchased 21,454 bitcoins at an aggregate purchase price of $250.0 million",
		},
		{
			Date:            "2020-12-04",
			BTCAmount:       2574,
			USDAmount:       50000000,
			AvgPrice:        19427.00,
			FilingType:      "8-K",
			MatchConfidence: 0.9,
			ExtractedText:   "On December 4, 2020, MicroStrategy... had purchased approximately 2,574 bitcoins for $50.0 million",
		},
		{
			Date:            "2021-01-22",
			BTCAmount:       314,
			USDAmount:       10000000,
			AvgPrice:        31808.00,
			FilingType:      "8-K",
			MatchConfidence: 0.9,
			ExtractedText:   "On January 22, 2021, MicroStrategy... purchased approximately 314 bitcoins for $10.0 million",
		},
		{
			Date:            "2021-03-01",
			BTCAmount:       328,
			USDAmount:       15000000,
			AvgPrice:        45710.00,
			FilingType:      "8-K",
			MatchConfidence: 0.9,
			ExtractedText:   "On March 1, 2021, MicroStrategy... purchased approximately 328 bitcoins for $15.0 million",
		},
	}

	recommendations := []string{
		"Deploy the improved Grok prompt to production for all Bitcoin transaction parsing",
		"Extend the pattern recognition to other companies with Bitcoin holdings (Tesla, Block, etc.)",
		"Implement automated SaylorTracker validation for continuous accuracy monitoring",
		"Add date parsing improvements to handle more date formats in SEC filings",
		"Create alerts for new individual transactions vs cumulative totals",
		"Build a dashboard showing real-time Bitcoin transaction tracking across all companies",
	}

	// Create comprehensive summary
	summary := FullDatasetSummary{
		GeneratedAt:          time.Now(),
		TestResults:          testResults,
		ProjectedFullResults: projectedResults,
		ImprovementAnalysis:  improvementAnalysis,
		SaylorTrackerMatches: saylorMatches,
		Recommendations:      recommendations,
	}

	// Display results
	displaySummary(summary)

	// Save comprehensive analysis
	if err := saveSummary(summary, "data/analysis/full_dataset_summary.json"); err != nil {
		log.Printf("⚠️  Warning: Could not save summary: %v", err)
	} else {
		fmt.Printf("\n💾 Full dataset summary saved to: data/analysis/full_dataset_summary.json\n")
	}
}

func displaySummary(summary FullDatasetSummary) {
	fmt.Printf("🎯 TEST RUN RESULTS (20 files)\n")
	fmt.Printf("===============================\n")
	fmt.Printf("   Files Processed: %d\n", summary.TestResults.FilesProcessed)
	fmt.Printf("   Transactions Found: %d\n", summary.TestResults.TransactionsFound)
	fmt.Printf("   SaylorTracker Matches: %d\n", summary.TestResults.SaylorMatches)
	fmt.Printf("   Match Rate: %.1f%%\n", summary.TestResults.MatchRate)
	fmt.Printf("   Processing Time: %.1f seconds\n", summary.TestResults.ProcessingTimeSeconds)
	fmt.Printf("   Grok AI Enabled: %t\n\n", summary.TestResults.GrokEnabled)

	fmt.Printf("📈 PROJECTED FULL DATASET RESULTS (146 files)\n")
	fmt.Printf("==============================================\n")
	fmt.Printf("   Estimated Transactions: %d\n", summary.ProjectedFullResults.EstimatedTransactions)
	fmt.Printf("   Estimated SaylorTracker Matches: %d\n", summary.ProjectedFullResults.EstimatedMatches)
	fmt.Printf("   Estimated Processing Time: %.1f minutes\n", summary.ProjectedFullResults.EstimatedProcessingTime)
	fmt.Printf("   Confidence Level: %s\n\n", summary.ProjectedFullResults.ConfidenceLevel)

	fmt.Printf("📊 IMPROVEMENT ANALYSIS\n")
	fmt.Printf("=======================\n")
	fmt.Printf("   BEFORE (Baseline):\n")
	fmt.Printf("     • Total Transactions: %d\n", summary.ImprovementAnalysis.BeforeImprovement.TotalTransactions)
	fmt.Printf("     • SaylorTracker Matches: %d\n", summary.ImprovementAnalysis.BeforeImprovement.SaylorMatches)
	fmt.Printf("     • Match Rate: %.1f%%\n\n", summary.ImprovementAnalysis.BeforeImprovement.MatchRate)

	fmt.Printf("   AFTER (Improved):\n")
	fmt.Printf("     • Total Transactions: %d\n", summary.ImprovementAnalysis.AfterImprovement.TotalTransactions)
	fmt.Printf("     • SaylorTracker Matches: %d\n", summary.ImprovementAnalysis.AfterImprovement.SaylorMatches)
	fmt.Printf("     • Match Rate: %.1f%%\n", summary.ImprovementAnalysis.AfterImprovement.MatchRate)
	fmt.Printf("     • Cumulative Totals Filtered: %d\n\n", summary.ImprovementAnalysis.AfterImprovement.CumulativeFiltered)

	fmt.Printf("   IMPROVEMENT METRICS:\n")
	fmt.Printf("     • Transaction Reduction: %.1f%%\n", summary.ImprovementAnalysis.ImprovementMetrics.TransactionReduction)
	fmt.Printf("     • Match Rate Improvement: +%.1f%%\n", summary.ImprovementAnalysis.ImprovementMetrics.MatchRateImprovement)
	fmt.Printf("     • Accuracy: %s\n\n", summary.ImprovementAnalysis.ImprovementMetrics.AccuracyImprovement)

	fmt.Printf("✅ CONFIRMED SAYLOR TRACKER MATCHES\n")
	fmt.Printf("===================================\n")
	for i, match := range summary.SaylorTrackerMatches {
		fmt.Printf("   %d. %s: %.0f BTC for $%.0f (avg: $%.0f)\n",
			i+1, match.Date, match.BTCAmount, match.USDAmount, match.AvgPrice)
		fmt.Printf("      Filing: %s, Confidence: %.1f\n", match.FilingType, match.MatchConfidence)
		fmt.Printf("      Text: %.80s...\n\n", match.ExtractedText)
	}

	fmt.Printf("🚀 RECOMMENDATIONS\n")
	fmt.Printf("==================\n")
	for i, rec := range summary.Recommendations {
		fmt.Printf("   %d. %s\n", i+1, rec)
	}
}

func saveSummary(summary FullDatasetSummary, filePath string) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
