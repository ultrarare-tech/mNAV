package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ChartData represents the data structure for the chart
type ChartData struct {
	Symbol    string    `json:"symbol"`
	Title     string    `json:"title"`
	Generated time.Time `json:"generated"`
	Labels    []string  `json:"labels"`
	Datasets  []Dataset `json:"datasets"`
}

// Dataset represents a data series for the chart
type Dataset struct {
	Label       string    `json:"label"`
	Data        []float64 `json:"data"`
	BorderColor string    `json:"borderColor"`
	YAxisID     string    `json:"yAxisID"`
	Fill        bool      `json:"fill"`
}

// HTML template for the chart
const chartTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Symbol}} mNAV Chart</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            text-align: center;
            color: #333;
        }
        .chart-container {
            position: relative;
            height: 600px;
            margin-top: 20px;
        }
        .info {
            text-align: center;
            color: #666;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        <div class="chart-container">
            <canvas id="mnavChart"></canvas>
        </div>
        <div class="info">
            Generated: {{.Generated.Format "2006-01-02 15:04:05"}}
        </div>
    </div>

    <script>
        const ctx = document.getElementById('mnavChart').getContext('2d');
        const chartData = {
            labels: {{.Labels}},
            datasets: {{.DatasetsJSON}}
        };

        new Chart(ctx, {
            type: 'line',
            data: chartData,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false,
                },
                plugins: {
                    title: {
                        display: false
                    },
                    legend: {
                        position: 'top',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                let label = context.dataset.label || '';
                                if (label) {
                                    label += ': ';
                                }
                                if (context.parsed.y !== null) {
                                    if (label.includes('Price')) {
                                        label += '$' + context.parsed.y.toFixed(2);
                                    } else if (label.includes('Premium')) {
                                        label += context.parsed.y.toFixed(1) + '%';
                                    } else {
                                        label += context.parsed.y.toFixed(2);
                                    }
                                }
                                return label;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Date'
                        }
                    },
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: 'mNAV'
                        }
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: 'Premium (%)'
                        },
                        grid: {
                            drawOnChartArea: false,
                        },
                    }
                }
            }
        });
    </script>
</body>
</html>`

func main() {
	var (
		input     = flag.String("input", "", "Path to historical mNAV JSON file")
		outputDir = flag.String("output", "data/charts", "Output directory for chart files")
		format    = flag.String("format", "html", "Output format: html, json, csv")
	)
	flag.Parse()

	fmt.Printf("📊 mNAV CHART GENERATOR\n")
	fmt.Printf("======================\n\n")

	if *input == "" {
		// Try to find the most recent mNAV file
		pattern := "data/analysis/mnav/*_mnav_historical_*.json"
		files, err := filepath.Glob(pattern)
		if err != nil || len(files) == 0 {
			log.Fatalf("❌ No input file specified and no historical mNAV files found")
		}
		*input = files[len(files)-1]
		fmt.Printf("📂 Using most recent file: %s\n", *input)
	}

	// Load historical mNAV data
	data, err := loadMNAVData(*input)
	if err != nil {
		log.Fatalf("❌ Error loading mNAV data: %v", err)
	}

	fmt.Printf("✅ Loaded %d data points for %s\n", len(data.DataPoints), data.Symbol)

	// Generate chart based on format
	switch *format {
	case "html":
		if err := generateHTMLChart(data, *outputDir); err != nil {
			log.Fatalf("❌ Error generating HTML chart: %v", err)
		}
	case "json":
		if err := generateJSONChart(data, *outputDir); err != nil {
			log.Fatalf("❌ Error generating JSON chart: %v", err)
		}
	case "csv":
		if err := generateCSVChart(data, *outputDir); err != nil {
			log.Fatalf("❌ Error generating CSV: %v", err)
		}
	default:
		log.Fatalf("❌ Unknown format: %s", *format)
	}

	fmt.Printf("\n✅ Chart generation complete!\n")
}

// HistoricalMNAVData structures (matching the mnav-historical output)
type HistoricalMNAVData struct {
	Symbol     string                 `json:"symbol"`
	StartDate  string                 `json:"start_date"`
	EndDate    string                 `json:"end_date"`
	DataPoints []HistoricalMNAVPoint  `json:"data_points"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type HistoricalMNAVPoint struct {
	Date              string  `json:"date"`
	StockPrice        float64 `json:"stock_price"`
	BitcoinPrice      float64 `json:"bitcoin_price"`
	BitcoinHoldings   float64 `json:"bitcoin_holdings"`
	SharesOutstanding float64 `json:"shares_outstanding"`
	MarketCap         float64 `json:"market_cap"`
	BitcoinValue      float64 `json:"bitcoin_value"`
	MNAV              float64 `json:"mnav"`
	MNAVPerShare      float64 `json:"mnav_per_share"`
	Premium           float64 `json:"premium_percentage"`
}

func loadMNAVData(filepath string) (*HistoricalMNAVData, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var mnavData HistoricalMNAVData
	if err := json.Unmarshal(data, &mnavData); err != nil {
		return nil, err
	}

	return &mnavData, nil
}

func generateHTMLChart(data *HistoricalMNAVData, outputDir string) error {
	// Prepare chart data
	labels := make([]string, len(data.DataPoints))
	mnavValues := make([]float64, len(data.DataPoints))
	premiumValues := make([]float64, len(data.DataPoints))

	for i, dp := range data.DataPoints {
		labels[i] = dp.Date
		mnavValues[i] = dp.MNAV
		premiumValues[i] = dp.Premium
	}

	chartData := ChartData{
		Symbol:    data.Symbol,
		Title:     fmt.Sprintf("%s mNAV History (%s to %s)", data.Symbol, data.StartDate, data.EndDate),
		Generated: time.Now(),
		Labels:    labels,
		Datasets: []Dataset{
			{
				Label:       "mNAV",
				Data:        mnavValues,
				BorderColor: "rgb(75, 192, 192)",
				YAxisID:     "y",
				Fill:        false,
			},
			{
				Label:       "Premium %",
				Data:        premiumValues,
				BorderColor: "rgb(255, 99, 132)",
				YAxisID:     "y1",
				Fill:        false,
			},
		},
	}

	// Convert datasets to JSON for the template
	datasetsJSON, err := json.Marshal(chartData.Datasets)
	if err != nil {
		return err
	}

	// Create template
	tmpl, err := template.New("chart").Parse(chartTemplate)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Create output file
	filename := fmt.Sprintf("%s_mnav_chart_%s.html", data.Symbol, time.Now().Format("2006-01-02"))
	filepath := filepath.Join(outputDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Execute template with additional data
	templateData := struct {
		ChartData
		DatasetsJSON template.JS
	}{
		ChartData:    chartData,
		DatasetsJSON: template.JS(datasetsJSON),
	}

	if err := tmpl.Execute(file, templateData); err != nil {
		return err
	}

	fmt.Printf("💾 HTML chart saved to: %s\n", filepath)
	return nil
}

func generateJSONChart(data *HistoricalMNAVData, outputDir string) error {
	// Create a simplified structure for charting libraries
	chartData := make(map[string]interface{})

	dates := make([]string, len(data.DataPoints))
	mnavs := make([]float64, len(data.DataPoints))
	premiums := make([]float64, len(data.DataPoints))
	stockPrices := make([]float64, len(data.DataPoints))
	btcPrices := make([]float64, len(data.DataPoints))

	for i, dp := range data.DataPoints {
		dates[i] = dp.Date
		mnavs[i] = dp.MNAV
		premiums[i] = dp.Premium
		stockPrices[i] = dp.StockPrice
		btcPrices[i] = dp.BitcoinPrice
	}

	chartData["symbol"] = data.Symbol
	chartData["dates"] = dates
	chartData["mnav"] = mnavs
	chartData["premium_percentage"] = premiums
	chartData["stock_price"] = stockPrices
	chartData["bitcoin_price"] = btcPrices
	chartData["metadata"] = map[string]interface{}{
		"start_date": data.StartDate,
		"end_date":   data.EndDate,
		"generated":  time.Now(),
	}

	// Save to file
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_mnav_chart_data_%s.json", data.Symbol, time.Now().Format("2006-01-02"))
	filepath := filepath.Join(outputDir, filename)

	jsonData, err := json.MarshalIndent(chartData, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return err
	}

	fmt.Printf("💾 JSON chart data saved to: %s\n", filepath)
	return nil
}

func generateCSVChart(data *HistoricalMNAVData, outputDir string) error {
	// Create CSV content
	csv := "Date,Stock Price,Bitcoin Price,Bitcoin Holdings,Shares Outstanding,Market Cap,Bitcoin Value,mNAV,mNAV Per Share,Premium %\n"

	for _, dp := range data.DataPoints {
		csv += fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.0f,%.2f,%.2f,%.4f,%.2f,%.2f\n",
			dp.Date,
			dp.StockPrice,
			dp.BitcoinPrice,
			dp.BitcoinHoldings,
			dp.SharesOutstanding,
			dp.MarketCap,
			dp.BitcoinValue,
			dp.MNAV,
			dp.MNAVPerShare,
			dp.Premium,
		)
	}

	// Save to file
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s_mnav_data_%s.csv", data.Symbol, time.Now().Format("2006-01-02"))
	filepath := filepath.Join(outputDir, filename)

	if err := os.WriteFile(filepath, []byte(csv), 0644); err != nil {
		return err
	}

	fmt.Printf("💾 CSV data saved to: %s\n", filepath)
	return nil
}
