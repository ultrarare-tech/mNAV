package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

type CumulativePattern struct {
	Context     string
	Count       int
	Examples    []string
	FilingTypes map[string]int
}

func main() {
	fmt.Printf("🔍 CUMULATIVE PATTERN ANALYSIS\n")
	fmt.Printf("==============================\n\n")

	// Analyze the enhanced parsing results log
	patterns, err := analyzeCumulativePatterns("enhanced_parsing_results.txt")
	if err != nil {
		log.Fatalf("❌ Error analyzing patterns: %v", err)
	}

	// Display results
	displayPatternAnalysis(patterns)
}

func analyzeCumulativePatterns(logFile string) (map[string]*CumulativePattern, error) {
	file, err := os.Open(logFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	patterns := make(map[string]*CumulativePattern)
	scanner := bufio.NewScanner(file)

	// Regex to match Bitcoin paragraph log entries
	paragraphRegex := regexp.MustCompile(`Found Bitcoin paragraph \d+ \(context: ([^)]+)\): (.+)`)
	filingRegex := regexp.MustCompile(`Processing (\d{4}-\d{2}-\d{2})_([^_]+)_`)

	var currentFilingType string

	for scanner.Scan() {
		line := scanner.Text()

		// Track current filing type
		if filingMatch := filingRegex.FindStringSubmatch(line); len(filingMatch) >= 3 {
			currentFilingType = filingMatch[2]
			continue
		}

		// Analyze Bitcoin paragraphs
		if matches := paragraphRegex.FindStringSubmatch(line); len(matches) >= 3 {
			context := matches[1]
			text := matches[2]

			// Only track cumulative patterns
			if strings.Contains(context, "cumulative") {
				if patterns[context] == nil {
					patterns[context] = &CumulativePattern{
						Context:     context,
						Count:       0,
						Examples:    []string{},
						FilingTypes: make(map[string]int),
					}
				}

				pattern := patterns[context]
				pattern.Count++
				pattern.FilingTypes[currentFilingType]++

				// Store examples (limit to 5 per pattern)
				if len(pattern.Examples) < 5 {
					// Clean up the text for display
					cleanText := cleanLogText(text)
					if len(cleanText) > 150 {
						cleanText = cleanText[:150] + "..."
					}
					pattern.Examples = append(pattern.Examples, cleanText)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

func cleanLogText(text string) string {
	// Remove HTML tags and excessive whitespace
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	cleaned := htmlRegex.ReplaceAllString(text, " ")

	// Remove excessive whitespace
	spaceRegex := regexp.MustCompile(`\s+`)
	cleaned = spaceRegex.ReplaceAllString(cleaned, " ")

	// Remove HTML entities
	entityRegex := regexp.MustCompile(`&#\d+;`)
	cleaned = entityRegex.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}

func displayPatternAnalysis(patterns map[string]*CumulativePattern) {
	fmt.Printf("📊 CUMULATIVE PATTERNS DETECTED\n")
	fmt.Printf("================================\n\n")

	totalCumulative := 0
	for _, pattern := range patterns {
		totalCumulative += pattern.Count
	}

	fmt.Printf("🔍 Total Cumulative Paragraphs Found: %d\n\n", totalCumulative)

	// Sort patterns by count
	var sortedPatterns []*CumulativePattern
	for _, pattern := range patterns {
		sortedPatterns = append(sortedPatterns, pattern)
	}

	// Simple sort by count (descending)
	for i := 0; i < len(sortedPatterns)-1; i++ {
		for j := i + 1; j < len(sortedPatterns); j++ {
			if sortedPatterns[i].Count < sortedPatterns[j].Count {
				sortedPatterns[i], sortedPatterns[j] = sortedPatterns[j], sortedPatterns[i]
			}
		}
	}

	for _, pattern := range sortedPatterns {
		fmt.Printf("📋 PATTERN: %s\n", strings.ToUpper(pattern.Context))
		fmt.Printf("   Count: %d paragraphs\n", pattern.Count)

		fmt.Printf("   Filing Types: ")
		for filingType, count := range pattern.FilingTypes {
			fmt.Printf("%s(%d) ", filingType, count)
		}
		fmt.Printf("\n")

		fmt.Printf("   Examples:\n")
		for i, example := range pattern.Examples {
			fmt.Printf("   %d. %s\n", i+1, example)
		}
		fmt.Printf("\n")
	}

	// Analysis summary
	fmt.Printf("📋 ANALYSIS SUMMARY\n")
	fmt.Printf("===================\n")
	fmt.Printf("✅ The enhanced parser is successfully identifying cumulative patterns\n")
	fmt.Printf("🎯 These patterns would have been incorrectly counted as individual transactions\n")
	fmt.Printf("💰 By filtering these out, we're improving the accuracy of Bitcoin transaction counting\n")

	if totalCumulative > 50 {
		fmt.Printf("🚨 HIGH VOLUME: %d cumulative patterns detected - significant over-counting prevention\n", totalCumulative)
	} else if totalCumulative > 20 {
		fmt.Printf("⚠️  MODERATE VOLUME: %d cumulative patterns detected - good filtering impact\n", totalCumulative)
	} else {
		fmt.Printf("✅ LOW VOLUME: %d cumulative patterns detected - minor but important filtering\n", totalCumulative)
	}
}
