package config

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// RebalancingRule represents a single row in the dynamic rebalancing table
type RebalancingRule struct {
	MinThreshold float64 `json:"min_threshold" csv:"Min"`
	MaxThreshold float64 `json:"max_threshold" csv:"Max"`
	TargetRatio  float64 `json:"target_ratio" csv:"Ratio (X:1)"`
}

// RebalancingConfig represents the complete rebalancing configuration
type RebalancingConfig struct {
	Rules   []RebalancingRule `json:"rules"`
	Version string            `json:"version"`
	Source  string            `json:"source"`
}

// LoadRebalancingConfig loads the rebalancing configuration from CSV or JSON
func LoadRebalancingConfig() (*RebalancingConfig, error) {
	csvPath := "configs/rebalancing/rebalancing_table.csv"
	jsonPath := "configs/rebalancing/rebalancing_table.json"

	// Check if JSON exists and is newer than CSV
	csvInfo, csvErr := os.Stat(csvPath)
	jsonInfo, jsonErr := os.Stat(jsonPath)

	if jsonErr == nil && csvErr == nil && jsonInfo.ModTime().After(csvInfo.ModTime()) {
		// JSON is newer, load from JSON
		return loadFromJSON(jsonPath)
	}

	if csvErr == nil {
		// CSV exists, load from CSV and optionally save as JSON
		config, err := loadFromCSV(csvPath)
		if err != nil {
			return nil, err
		}

		// Save as JSON for faster future loading
		if err := saveToJSON(config, jsonPath); err != nil {
			// Log warning but don't fail
			fmt.Printf("⚠️  Warning: Could not save JSON cache: %v\n", err)
		}

		return config, nil
	}

	return nil, fmt.Errorf("no rebalancing configuration found at %s or %s", csvPath, jsonPath)
}

// loadFromCSV loads the rebalancing rules from a CSV file
func loadFromCSV(filePath string) (*RebalancingConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have at least a header and one data row")
	}

	// Validate header
	header := records[0]
	if len(header) != 3 || header[0] != "Min" || header[1] != "Max" || header[2] != "Ratio (X:1)" {
		return nil, fmt.Errorf("CSV header must be: Min,Max,Ratio (X:1)")
	}

	var rules []RebalancingRule

	// Process data rows
	for i, record := range records[1:] {
		if len(record) != 3 {
			return nil, fmt.Errorf("row %d: expected 3 columns, got %d", i+2, len(record))
		}

		min, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Min value '%s': %w", i+2, record[0], err)
		}

		max, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Max value '%s': %w", i+2, record[1], err)
		}

		ratio, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid Ratio value '%s': %w", i+2, record[2], err)
		}

		// Validation
		if min >= max {
			return nil, fmt.Errorf("row %d: Min (%.4f) must be less than Max (%.4f)", i+2, min, max)
		}

		if ratio <= 0 {
			return nil, fmt.Errorf("row %d: Ratio (%.4f) must be positive", i+2, ratio)
		}

		rules = append(rules, RebalancingRule{
			MinThreshold: min,
			MaxThreshold: max,
			TargetRatio:  ratio,
		})
	}

	// Sort rules by ratio (highest first - most MSTR allocation)
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].TargetRatio < rules[j].TargetRatio {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	absPath, _ := filepath.Abs(filePath)

	return &RebalancingConfig{
		Rules:   rules,
		Version: "1.0",
		Source:  absPath,
	}, nil
}

// loadFromJSON loads the rebalancing configuration from a JSON file
func loadFromJSON(filePath string) (*RebalancingConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var config RebalancingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

// saveToJSON saves the rebalancing configuration to a JSON file
func saveToJSON(config *RebalancingConfig, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// ValidateConfig validates the rebalancing configuration
func (config *RebalancingConfig) ValidateConfig() error {
	if len(config.Rules) == 0 {
		return fmt.Errorf("configuration must have at least one rule")
	}

	// Basic validation for each rule
	for i, rule := range config.Rules {
		if rule.MinThreshold >= rule.MaxThreshold {
			return fmt.Errorf("rule %d: Min (%.4f) must be less than Max (%.4f)",
				i+1, rule.MinThreshold, rule.MaxThreshold)
		}

		if rule.TargetRatio <= 0 {
			return fmt.Errorf("rule %d: TargetRatio (%.4f) must be positive",
				i+1, rule.TargetRatio)
		}
	}

	// Note: Overlapping ranges are intentional for threshold-based rebalancing
	// They allow for smooth transitions between ratios

	return nil
}

// GetSummary returns a human-readable summary of the configuration
func (config *RebalancingConfig) GetSummary() string {
	summary := fmt.Sprintf("Rebalancing Configuration (Version %s)\n", config.Version)
	summary += fmt.Sprintf("Source: %s\n", config.Source)
	summary += fmt.Sprintf("Rules: %d\n\n", len(config.Rules))

	summary += "Min    | Max    | Ratio\n"
	summary += "-------|--------|----- \n"

	for _, rule := range config.Rules {
		summary += fmt.Sprintf("%.4f | %.4f | %.0f:1\n",
			rule.MinThreshold, rule.MaxThreshold, rule.TargetRatio)
	}

	return summary
}
