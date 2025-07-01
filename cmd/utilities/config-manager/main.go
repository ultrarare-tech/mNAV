package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jeffreykibler/mNAV/pkg/config"
)

func main() {
	var (
		validate = flag.Bool("validate", false, "Validate the rebalancing configuration")
		convert  = flag.Bool("convert", false, "Convert CSV to JSON")
		summary  = flag.Bool("summary", false, "Show configuration summary")
		force    = flag.Bool("force", false, "Force regeneration of JSON from CSV")
	)
	flag.Parse()

	if *validate {
		validateConfig()
		return
	}

	if *convert {
		convertConfig(*force)
		return
	}

	if *summary {
		showSummary()
		return
	}

	// Default action
	showSummary()
}

func validateConfig() {
	fmt.Printf("🔍 VALIDATING REBALANCING CONFIGURATION\n")
	fmt.Printf("======================================\n\n")

	config, err := config.LoadRebalancingConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	if err := config.ValidateConfig(); err != nil {
		log.Fatalf("❌ Configuration validation failed: %v", err)
	}

	fmt.Printf("✅ Configuration is valid!\n")
	fmt.Printf("   Source: %s\n", config.Source)
	fmt.Printf("   Rules: %d\n", len(config.Rules))
	fmt.Printf("   Version: %s\n\n", config.Version)

	// Show ranges for verification
	fmt.Printf("📊 Ratio Ranges:\n")
	for _, rule := range config.Rules {
		fmt.Printf("   %.0f:1 → [%.4f - %.4f]\n",
			rule.TargetRatio, rule.MinThreshold, rule.MaxThreshold)
	}
}

func convertConfig(force bool) {
	fmt.Printf("🔄 CONVERTING CONFIGURATION\n")
	fmt.Printf("===========================\n\n")

	csvPath := "configs/rebalancing/rebalancing_table.csv"
	jsonPath := "configs/rebalancing/rebalancing_table.json"

	if !force {
		// Check if JSON already exists and is newer
		csvInfo, csvErr := os.Stat(csvPath)
		jsonInfo, jsonErr := os.Stat(jsonPath)

		if jsonErr == nil && csvErr == nil && jsonInfo.ModTime().After(csvInfo.ModTime()) {
			fmt.Printf("ℹ️  JSON file is already up to date\n")
			fmt.Printf("   Use -force to regenerate anyway\n")
			return
		}
	}

	// Load from CSV
	configObj, err := config.LoadRebalancingConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load CSV: %v", err)
	}

	fmt.Printf("✅ Successfully converted CSV to JSON\n")
	fmt.Printf("   CSV: %s\n", csvPath)
	fmt.Printf("   JSON: %s\n", jsonPath)
	fmt.Printf("   Rules: %d\n", len(configObj.Rules))
}

func showSummary() {
	fmt.Printf("📋 REBALANCING CONFIGURATION SUMMARY\n")
	fmt.Printf("===================================\n\n")

	configObj, err := config.LoadRebalancingConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	fmt.Printf("%s\n", configObj.GetSummary())

	fmt.Printf("📝 Usage Instructions:\n")
	fmt.Printf("   • Edit: configs/rebalancing/rebalancing_table.csv\n")
	fmt.Printf("   • Format: Min,Max,Ratio (X:1)\n")
	fmt.Printf("   • Auto-conversion: JSON is generated automatically\n")
	fmt.Printf("   • Validation: Run './bin/config-manager -validate'\n\n")

	fmt.Printf("🔧 Management Commands:\n")
	fmt.Printf("   ./bin/config-manager -summary    # Show this summary\n")
	fmt.Printf("   ./bin/config-manager -validate   # Validate configuration\n")
	fmt.Printf("   ./bin/config-manager -convert    # Force CSV to JSON conversion\n")
}
