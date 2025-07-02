package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ultrarare-tech/mNAV/pkg/collection/external"
)

func main() {
	// Create output directory if it doesn't exist
	outputDir := "data/mstr"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Initialize SaylorTracker client
	client := external.NewSaylorTrackerClient()

	// Save comprehensive MSTR data to JSON
	jsonFile := filepath.Join(outputDir, "mstr_bitcoin_data.json")
	if err := client.SaveToJSON(jsonFile); err != nil {
		log.Fatalf("Failed to save MSTR data to JSON: %v", err)
	}

	fmt.Printf("\nMSTR Bitcoin data successfully saved to: %s\n", jsonFile)
	fmt.Println("\nThis JSON file contains:")
	fmt.Println("- Complete transaction history from 2020-2025")
	fmt.Println("- Quarterly Bitcoin holdings data")
	fmt.Println("- Historical shares outstanding information")
	fmt.Println("- Calculated totals and averages")
	fmt.Println("\nThis data can now be used by downstream processes like CSV exporters and analysis tools.")
}
