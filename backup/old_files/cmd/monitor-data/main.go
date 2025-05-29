package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jeffreykibler/mNAV/pkg/edgar"
)

func main() {
	var (
		dataDir         = flag.String("data", "data/edgar", "Directory containing company data")
		maxAgeDays      = flag.Int("max-age", 7, "Maximum age in days before data is considered stale")
		format          = flag.String("format", "text", "Output format: text, json")
		alertsOnly      = flag.Bool("alerts-only", false, "Show only alerts")
		recommendations = flag.Bool("recommendations", false, "Show refresh recommendations")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nMonitors SEC EDGAR data health and generates alerts.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Check data health\n")
		fmt.Fprintf(os.Stderr, "  %s\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Show only alerts\n")
		fmt.Fprintf(os.Stderr, "  %s -alerts-only\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Get refresh recommendations\n")
		fmt.Fprintf(os.Stderr, "  %s -recommendations\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Output as JSON\n")
		fmt.Fprintf(os.Stderr, "  %s -format json\n", os.Args[0])
	}

	flag.Parse()

	// Create storage and monitoring service
	storage := edgar.NewCompanyDataStorage(*dataDir)
	monitor := edgar.NewMonitoringService(storage)

	maxAge := time.Duration(*maxAgeDays) * 24 * time.Hour

	if *recommendations {
		// Show refresh recommendations
		recs, err := monitor.GetRefreshRecommendations(maxAge)
		if err != nil {
			log.Fatalf("Error getting recommendations: %v", err)
		}

		if *format == "json" {
			data, _ := json.MarshalIndent(recs, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println("=== Data Refresh Recommendations ===")
			fmt.Println()

			if len(recs) == 0 {
				fmt.Println("No refresh needed - all data is up to date!")
			} else {
				// Group by priority
				high := []edgar.RefreshRecommendation{}
				medium := []edgar.RefreshRecommendation{}
				low := []edgar.RefreshRecommendation{}

				for _, rec := range recs {
					switch rec.Priority {
					case "high":
						high = append(high, rec)
					case "medium":
						medium = append(medium, rec)
					case "low":
						low = append(low, rec)
					}
				}

				// Print high priority
				if len(high) > 0 {
					fmt.Println("HIGH PRIORITY:")
					for _, rec := range high {
						fmt.Printf("  • %s - %s (last updated: %s ago)\n",
							rec.Symbol, rec.Reason, rec.DataAge.Round(time.Hour))
						for _, action := range rec.RecommendedActions {
							fmt.Printf("    → %s\n", action)
						}
						fmt.Println()
					}
				}

				// Print medium priority
				if len(medium) > 0 {
					fmt.Println("MEDIUM PRIORITY:")
					for _, rec := range medium {
						fmt.Printf("  • %s - %s (last updated: %s ago)\n",
							rec.Symbol, rec.Reason, rec.DataAge.Round(time.Hour))
						for _, action := range rec.RecommendedActions {
							fmt.Printf("    → %s\n", action)
						}
						fmt.Println()
					}
				}

				// Print low priority
				if len(low) > 0 {
					fmt.Println("LOW PRIORITY:")
					for _, rec := range low {
						fmt.Printf("  • %s - %s\n", rec.Symbol, rec.Reason)
						for _, action := range rec.RecommendedActions {
							fmt.Printf("    → %s\n", action)
						}
						fmt.Println()
					}
				}
			}
		}
		return
	}

	// Check system health
	health, err := monitor.CheckSystemHealth(maxAge)
	if err != nil {
		log.Fatalf("Error checking system health: %v", err)
	}

	// Generate alerts
	alerts := monitor.GenerateAlerts(health)

	if *alertsOnly {
		// Show only alerts
		if *format == "json" {
			data, _ := json.MarshalIndent(alerts, "", "  ")
			fmt.Println(string(data))
		} else {
			if len(alerts) == 0 {
				fmt.Println("✓ No alerts - system is healthy!")
			} else {
				fmt.Printf("⚠️  %d alerts found:\n\n", len(alerts))

				// Group by level
				critical := []edgar.Alert{}
				errors := []edgar.Alert{}
				warnings := []edgar.Alert{}

				for _, alert := range alerts {
					switch alert.Level {
					case "critical":
						critical = append(critical, alert)
					case "error":
						errors = append(errors, alert)
					case "warning":
						warnings = append(warnings, alert)
					}
				}

				// Print critical alerts
				if len(critical) > 0 {
					fmt.Println("🔴 CRITICAL:")
					for _, alert := range critical {
						if alert.Company != "" {
							fmt.Printf("   [%s] %s: %s\n", alert.Company, alert.Message, alert.Details)
						} else {
							fmt.Printf("   %s: %s\n", alert.Message, alert.Details)
						}
					}
					fmt.Println()
				}

				// Print error alerts
				if len(errors) > 0 {
					fmt.Println("🟠 ERRORS:")
					for _, alert := range errors {
						if alert.Company != "" {
							fmt.Printf("   [%s] %s: %s\n", alert.Company, alert.Message, alert.Details)
						} else {
							fmt.Printf("   %s: %s\n", alert.Message, alert.Details)
						}
					}
					fmt.Println()
				}

				// Print warning alerts
				if len(warnings) > 0 {
					fmt.Println("🟡 WARNINGS:")
					for _, alert := range warnings {
						if alert.Company != "" {
							fmt.Printf("   [%s] %s: %s\n", alert.Company, alert.Message, alert.Details)
						} else {
							fmt.Printf("   %s: %s\n", alert.Message, alert.Details)
						}
					}
				}
			}
		}
		return
	}

	// Show full health report
	if *format == "json" {
		data, _ := json.MarshalIndent(health, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("=== SEC EDGAR Data Health Report ===")
		fmt.Printf("Timestamp: %s\n", health.Timestamp.Format(time.RFC3339))
		fmt.Printf("Total Companies: %d\n", health.TotalCompanies)
		fmt.Printf("Companies with Data: %d\n", health.CompaniesWithData)
		fmt.Printf("Stale Companies: %d\n", health.StaleCompanies)
		fmt.Println()

		// System issues
		if len(health.SystemIssues) > 0 {
			fmt.Println("System Issues:")
			for _, issue := range health.SystemIssues {
				fmt.Printf("  ⚠️  %s\n", issue)
			}
			fmt.Println()
		}

		// Company health details
		fmt.Println("Company Details:")
		for _, company := range health.DataHealth {
			status := "✓"
			if len(company.Issues) > 0 {
				status = "⚠️"
			}
			if !company.SharesDataAvailable && !company.BTCDataAvailable {
				status = "❌"
			}

			fmt.Printf("\n%s %s:\n", status, company.Symbol)
			fmt.Printf("  Last Updated: %s (%s ago)\n",
				company.LastUpdated.Format("2006-01-02 15:04:05"),
				company.DataAge.Round(time.Second))

			if !company.LastFilingDate.IsZero() {
				fmt.Printf("  Last Filing: %s\n", company.LastFilingDate.Format("2006-01-02"))
			}

			fmt.Printf("  Shares Data: %v", company.SharesDataAvailable)
			if company.SharesDataAvailable && company.ConfidenceScore > 0 {
				fmt.Printf(" (confidence: %.2f)", company.ConfidenceScore)
			}
			fmt.Println()

			fmt.Printf("  BTC Data: %v\n", company.BTCDataAvailable)

			if len(company.Issues) > 0 {
				fmt.Println("  Issues:")
				for _, issue := range company.Issues {
					fmt.Printf("    - %s\n", issue)
				}
			}
		}

		// Summary
		fmt.Println("\n=== Summary ===")
		if len(alerts) == 0 {
			fmt.Println("✅ System is healthy - no alerts!")
		} else {
			critical := 0
			errors := 0
			warnings := 0
			for _, alert := range alerts {
				switch alert.Level {
				case "critical":
					critical++
				case "error":
					errors++
				case "warning":
					warnings++
				}
			}

			fmt.Printf("⚠️  %d total alerts: %d critical, %d errors, %d warnings\n",
				len(alerts), critical, errors, warnings)
			fmt.Println("\nRun with -alerts-only to see alert details")
			fmt.Println("Run with -recommendations to see refresh recommendations")
		}
	}
}
