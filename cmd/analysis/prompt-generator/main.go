package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type TrainingExample struct {
	Text           string `json:"text"`
	Classification string `json:"classification"`
	Reasoning      string `json:"reasoning"`
}

type ImprovedPrompt struct {
	GeneratedAt        time.Time         `json:"generated_at"`
	IndividualExamples []TrainingExample `json:"individual_examples"`
	CumulativeExamples []TrainingExample `json:"cumulative_examples"`
	PromptTemplate     string            `json:"prompt_template"`
	Instructions       string            `json:"instructions"`
}

func main() {
	fmt.Printf("🔍 IMPROVED GROK PROMPT GENERATOR\n")
	fmt.Printf("=================================\n\n")

	// Create training examples based on our analysis
	individualExamples := []TrainingExample{
		{
			Text:           "On August 11, 2020, MicroStrategy Incorporated (the \"Company,\" \"we,\" or \"us\") issued a press release announcing that the Company has purchased 21,454 bitcoins at an aggregate purchase price of $250.0 million, inclusive of fees and expenses (the \"BTC Investment\").",
			Classification: "individual_transaction",
			Reasoning:      "Specific date 'On August 11, 2020' with direct purchase announcement - this is an individual transaction",
		},
		{
			Text:           "On December 4, 2020, MicroStrategy Incorporated (the \"Company\") announced that it had purchased approximately 2,574 bitcoins for $50.0 million in cash in accordance with its Treasury Reserve Policy, at an average price of approximately $19,427 per bitcoin.",
			Classification: "individual_transaction",
			Reasoning:      "Specific date 'On December 4, 2020' with direct purchase announcement - this is an individual transaction",
		},
		{
			Text:           "On January 22, 2021, MicroStrategy Incorporated (the \"Company\") announced that it had purchased approximately 314 bitcoins for $10.0 million in cash, at an average price of approximately $31,808 per bitcoin, inclusive of fees and expenses.",
			Classification: "individual_transaction",
			Reasoning:      "Specific date 'On January 22, 2021' with direct purchase announcement - this is an individual transaction",
		},
		{
			Text:           "On February 24, 2021, MicroStrategy Incorporated (the \"Company\") announced that it had purchased approximately 19,452 bitcoins for approximately $1.026 billion in cash, at an average price of approximately $52,765 per bitcoin, inclusive of fees and expenses.",
			Classification: "individual_transaction",
			Reasoning:      "Specific date 'On February 24, 2021' with direct purchase announcement - this is an individual transaction",
		},
		{
			Text:           "On March 5, 2021, MicroStrategy Incorporated (the \"Company\") announced that it had purchased approximately 328 bitcoins for $15.0 million in cash, at an average price of approximately $45,732 per bitcoin, inclusive of fees and expenses.",
			Classification: "individual_transaction",
			Reasoning:      "Specific date 'On March 5, 2021' with direct purchase announcement - this is an individual transaction",
		},
	}

	cumulativeExamples := []TrainingExample{
		{
			Text:           "On August 24, 2021, MicroStrategy Incorporated (the \"Company\") announced that during the third quarter of the Company's fiscal year to date (the period between July 1, 2021 and August 23, 2021), the Company purchased approximately 3,907 bitcoins for approximately $177.0 million in cash, at an average price of approximately $45,294 per bitcoin, inclusive of fees and expenses.",
			Classification: "cumulative_total",
			Reasoning:      "Contains date range 'the period between July 1, 2021 and August 23, 2021' - this is a cumulative total over a period",
		},
		{
			Text:           "On September 13, 2021, MicroStrategy Incorporated (the \"Company\") announced that during the third quarter of the Company's fiscal year to date (the period between July 1, 2021 and September 12, 2021), the Company purchased approximately 8,957 bitcoins for approximately $419.9 million in cash, at an average price of approximately $46,875 per bitcoin, inclusive of fees and expenses.",
			Classification: "cumulative_total",
			Reasoning:      "Contains date range 'the period between July 1, 2021 and September 12, 2021' - this is a cumulative total over a period",
		},
		{
			Text:           "On November 29, 2021, MicroStrategy Incorporated (the \"Company\") announced that during the fourth quarter of the Company's fiscal year to date (the period between October 1, 2021 and November 29, 2021), the Company purchased approximately 7,002 bitcoins for approximately $414.4 million in cash, at an average price of approximately $59,187 per bitcoin, inclusive of fees and expenses.",
			Classification: "cumulative_total",
			Reasoning:      "Contains date range 'the period between October 1, 2021 and November 29, 2021' - this is a cumulative total over a period",
		},
		{
			Text:           "On December 9, 2021, MicroStrategy Incorporated (the \"Company\") announced that, during the period between November 29, 2021 and December 8, 2021, the Company purchased approximately 1,434 bitcoins for approximately $82.4 million in cash, at an average price of approximately $57,477 per bitcoin, inclusive of fees and expenses.",
			Classification: "cumulative_total",
			Reasoning:      "Contains date range 'during the period between November 29, 2021 and December 8, 2021' - this is a cumulative total over a period",
		},
		{
			Text:           "During the period between November 1, 2022 and December 21, 2022, MicroStrategy, through its wholly-owned subsidiary MacroStrategy LLC (\"MacroStrategy\"), acquired approximately 2,395 bitcoins for approximately $42.8 million in cash, at an average price of approximately $17,871 per bitcoin, inclusive of fees and expenses.",
			Classification: "cumulative_total",
			Reasoning:      "Starts with 'During the period between' with date range - this is clearly a cumulative total over a period",
		},
	}

	// Generate improved prompt template
	promptTemplate := createImprovedPromptTemplate()

	// Create the improved prompt structure
	improvedPrompt := ImprovedPrompt{
		GeneratedAt:        time.Now(),
		IndividualExamples: individualExamples,
		CumulativeExamples: cumulativeExamples,
		PromptTemplate:     promptTemplate,
		Instructions:       "This improved prompt template should be used to replace the existing Grok Bitcoin extraction prompt. It provides clear examples and rules for distinguishing individual transactions from cumulative totals.",
	}

	// Display the results
	displayPromptAnalysis(improvedPrompt)

	// Save the improved prompt
	if err := saveImprovedPrompt(improvedPrompt, "data/analysis/improved_grok_prompt.json"); err != nil {
		log.Printf("⚠️  Warning: Could not save improved prompt: %v", err)
	} else {
		fmt.Printf("\n💾 Improved prompt saved to: data/analysis/improved_grok_prompt.json\n")
	}

	// Generate the actual prompt code for easy copy-paste
	generatePromptCode(promptTemplate)
}

func createImprovedPromptTemplate() string {
	return `You are an expert financial analyst specializing in SEC filing analysis. Your task is to extract Bitcoin transaction information from the following SEC filing text.

FILING CONTEXT:
- Filing Type: %s
- Filing Date: %s
- Company: Public company filing with SEC
- Accession Number: %s

CRITICAL CLASSIFICATION RULES:

1. INDIVIDUAL TRANSACTIONS (EXTRACT THESE):
   - Announced on a SPECIFIC DATE with direct purchase language
   - Pattern: "On [specific date], [company] purchased/acquired [amount] bitcoins"
   - Examples:
     * "On August 11, 2020, MicroStrategy... has purchased 21,454 bitcoins"
     * "On December 4, 2020, MicroStrategy... had purchased approximately 2,574 bitcoins"
     * "On January 22, 2021, MicroStrategy... purchased approximately 314 bitcoins"
   - Key indicators: "On [date]", "announced that it had purchased", "announced that it purchased"

2. CUMULATIVE TOTALS (DO NOT EXTRACT):
   - Covers a DATE RANGE or PERIOD between two dates
   - Pattern: "during [period/quarter/time range between date1 and date2], purchased [amount] bitcoins"
   - Examples:
     * "during the period between July 1, 2021 and August 23, 2021, purchased 3,907 bitcoins"
     * "during the period between November 29, 2021 and December 8, 2021, purchased 1,434 bitcoins"
     * "During the period between November 1, 2022 and December 21, 2022, acquired 2,395 bitcoins"
   - Key indicators: "during the period between", "the period between X and Y", "during the quarter"

3. ADDITIONAL EXCLUSIONS:
   - Holdings updates ("As of [date], the Company holds X bitcoins")
   - Financing activities (bond offerings, loan proceeds, convertible notes)
   - Future intentions ("intends to invest", "will use proceeds", "may purchase")
   - Impairment charges or accounting adjustments

EXTRACTION RULES:
- ONLY extract transactions that follow the "On [specific date]" pattern
- IGNORE any transaction that mentions a date range or period
- For valid transactions, extract: BTC amount, USD amount, price per BTC, specific date
- Set confidence based on clarity of information (0.9 for clear individual transactions)

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
      "reasoning": "Brief explanation of why this is a valid INDIVIDUAL transaction (not cumulative)",
      "source_text": "Exact relevant excerpt from filing"
    }
  ],
  "analysis": "Overall analysis of Bitcoin-related content, noting any cumulative totals that were excluded",
  "confidence": 0.0,
  "reasoning": "Overall reasoning for the extraction results, explaining individual vs cumulative classification"
}

If no INDIVIDUAL Bitcoin transactions are found (only cumulative totals), return an empty transactions array but still provide analysis explaining what cumulative data was found and excluded.`
}

func displayPromptAnalysis(prompt ImprovedPrompt) {
	fmt.Printf("📊 IMPROVED GROK PROMPT ANALYSIS\n")
	fmt.Printf("================================\n\n")

	fmt.Printf("✅ INDIVIDUAL TRANSACTION EXAMPLES (%d):\n", len(prompt.IndividualExamples))
	for i, example := range prompt.IndividualExamples {
		fmt.Printf("   %d. %s\n", i+1, truncateText(example.Text, 100))
		fmt.Printf("      → %s\n\n", example.Reasoning)
	}

	fmt.Printf("⚠️  CUMULATIVE TOTAL EXAMPLES (%d):\n", len(prompt.CumulativeExamples))
	for i, example := range prompt.CumulativeExamples {
		fmt.Printf("   %d. %s\n", i+1, truncateText(example.Text, 100))
		fmt.Printf("      → %s\n\n", example.Reasoning)
	}

	fmt.Printf("🔍 KEY PATTERNS IDENTIFIED:\n")
	fmt.Printf("   Individual: 'On [specific date], [company] purchased/acquired [amount] bitcoins'\n")
	fmt.Printf("   Cumulative: 'during the period between [date1] and [date2], purchased [amount] bitcoins'\n\n")

	fmt.Printf("🎯 CLASSIFICATION STRATEGY:\n")
	fmt.Printf("   ✅ Extract: Specific date announcements (On August 11, 2020...)\n")
	fmt.Printf("   ❌ Exclude: Date range periods (during the period between...)\n")
	fmt.Printf("   ❌ Exclude: Holdings updates (As of [date], holds X bitcoins)\n\n")
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func saveImprovedPrompt(prompt ImprovedPrompt, filePath string) error {
	data, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func generatePromptCode(template string) {
	fmt.Printf("📝 PROMPT CODE FOR IMPLEMENTATION:\n")
	fmt.Printf("==================================\n\n")
	fmt.Printf("Replace the createBitcoinExtractionPrompt function in pkg/interpretation/grok/client.go with:\n\n")

	// Escape the template for Go code
	escaped := strings.ReplaceAll(template, "`", "` + \"`\" + `")

	fmt.Printf("func (c *Client) createBitcoinExtractionPrompt(text string, filing models.Filing) string {\n")
	fmt.Printf("	prompt := fmt.Sprintf(`%s`,\n", escaped)
	fmt.Printf("		filing.FilingType,\n")
	fmt.Printf("		filing.FilingDate.Format(\"2006-01-02\"),\n")
	fmt.Printf("		filing.AccessionNumber,\n")
	fmt.Printf("		text)\n\n")
	fmt.Printf("	return prompt\n")
	fmt.Printf("}\n\n")

	fmt.Printf("💾 This code has been saved to: data/analysis/improved_grok_prompt.json\n")
}
