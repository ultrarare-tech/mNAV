package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ScriptOutput represents the parsed output from the update-mnav script
type ScriptOutput struct {
	Timestamp       string        `json:"timestamp"`
	BitcoinPrice    string        `json:"bitcoin_price"`
	MSTRPrice       string        `json:"mstr_price"`
	FBTCPrice       string        `json:"fbtc_price"`
	BitcoinHoldings string        `json:"bitcoin_holdings"`
	BitcoinValue    string        `json:"bitcoin_value"`
	MNAV            string        `json:"mnav"`
	Premium         string        `json:"premium"`
	Ratio           string        `json:"ratio"`
	MarketCap       string        `json:"market_cap"`
	MarketTrend     MarketTrend   `json:"market_trend"`
	Portfolio       PortfolioData `json:"portfolio"`
	DataSources     []DataSource  `json:"data_sources"`
	FilesUpdated    []string      `json:"files_updated"`
	RawOutput       string        `json:"raw_output"`
	Success         bool          `json:"success"`
	Error           string        `json:"error,omitempty"`
}

// MarketTrend represents market trend analysis
type MarketTrend struct {
	PreviousMNAV     string `json:"previous_mnav"`
	PreviousPremium  string `json:"previous_premium"`
	CurrentMNAV      string `json:"current_mnav"`
	CurrentPremium   string `json:"current_premium"`
	TrendDirection   string `json:"trend_direction"`
	TrendDescription string `json:"trend_description"`
}

// PortfolioData represents portfolio analysis
type PortfolioData struct {
	Holdings        []PortfolioHolding `json:"holdings"`
	NetValue        string             `json:"net_value"`
	NetBitcoinValue string             `json:"net_bitcoin_value"`
	BitcoinExposure string             `json:"bitcoin_exposure"`
	CurrentRatio    string             `json:"current_ratio"`
	TargetRatio     string             `json:"target_ratio"`
	IsBalanced      bool               `json:"is_balanced"`
	Recommendation  string             `json:"recommendation"`
}

// PortfolioHolding represents individual portfolio holdings
type PortfolioHolding struct {
	Symbol string `json:"symbol"`
	Shares string `json:"shares"`
	Price  string `json:"price"`
	Value  string `json:"value"`
}

// DataSource represents a data source with freshness info
type DataSource struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Price       string `json:"price,omitempty"`
	LastUpdated string `json:"last_updated"`
	DataFile    string `json:"data_file,omitempty"`
	Method      string `json:"method,omitempty"`
	Holdings    string `json:"holdings,omitempty"`
}

// WebServer handles HTTP requests for the mNAV web interface
type WebServer struct {
	workspaceRoot string
}

// NewWebServer creates a new web server instance
func NewWebServer() *WebServer {
	// Get the current working directory (should be project root when run from there)
	workspaceRoot, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not determine workspace root:", err)
	}

	// Verify we're in the right directory by checking for the update script
	scriptPath := filepath.Join(workspaceRoot, "sh", "update-mnav")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Fatal("Could not find update-mnav script. Please run mnav-web from the project root directory.")
	}

	return &WebServer{
		workspaceRoot: workspaceRoot,
	}
}

// serveHTML serves the main HTML interface
func (ws *WebServer) serveHTML(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>mNAV Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        
        .header {
            background: linear-gradient(135deg, #2c3e50 0%, #3498db 100%);
            color: white;
            padding: 30px;
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            font-weight: 300;
        }
        
        .header p {
            opacity: 0.9;
            font-size: 1.1em;
        }
        
        .update-section {
            padding: 30px;
            text-align: center;
            border-bottom: 1px solid #eee;
        }
        
        .update-btn {
            background: linear-gradient(135deg, #27ae60 0%, #2ecc71 100%);
            color: white;
            border: none;
            padding: 15px 40px;
            font-size: 1.1em;
            border-radius: 50px;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 5px 15px rgba(46, 204, 113, 0.3);
        }
        
        .update-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 25px rgba(46, 204, 113, 0.4);
        }
        
        .update-btn:disabled {
            background: #bdc3c7;
            cursor: not-allowed;
            transform: none;
            box-shadow: none;
        }
        
        .loading {
            display: none;
            margin-top: 20px;
        }
        
        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #3498db;
            border-radius: 50%;
            width: 30px;
            height: 30px;
            animation: spin 1s linear infinite;
            margin: 0 auto;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .content {
            padding: 30px;
        }
        
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .metric-card {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            text-align: center;
            border-left: 4px solid #3498db;
        }
        
        .metric-card.premium {
            border-left-color: #e74c3c;
        }
        
        .metric-card.bitcoin {
            border-left-color: #f39c12;
        }
        
        .metric-card.portfolio {
            border-left-color: #27ae60;
        }
        
        .metric-value {
            font-size: 2em;
            font-weight: bold;
            color: #2c3e50;
            margin-bottom: 5px;
        }
        
        .metric-label {
            color: #7f8c8d;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        
        .section {
            margin-bottom: 30px;
        }
        
        .section-title {
            font-size: 1.5em;
            color: #2c3e50;
            margin-bottom: 15px;
            padding-bottom: 10px;
            border-bottom: 2px solid #ecf0f1;
        }
        
        .data-table {
            width: 100%;
            border-collapse: collapse;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 3px 10px rgba(0,0,0,0.1);
        }
        
        .data-table th {
            background: #34495e;
            color: white;
            padding: 15px;
            text-align: left;
        }
        
        .data-table td {
            padding: 12px 15px;
            border-bottom: 1px solid #ecf0f1;
        }
        
        .data-table tr:last-child td {
            border-bottom: none;
        }
        
        .data-table tr:nth-child(even) {
            background: #f8f9fa;
        }
        
        .timestamp {
            text-align: center;
            color: #7f8c8d;
            font-style: italic;
            margin-top: 20px;
        }
        
        .error {
            background: #e74c3c;
            color: white;
            padding: 15px;
            border-radius: 8px;
            margin: 20px 0;
        }
        
        .raw-output {
            background: #2c3e50;
            color: #ecf0f1;
            padding: 20px;
            border-radius: 8px;
            font-family: 'Courier New', monospace;
            white-space: pre-wrap;
            font-size: 0.9em;
            max-height: 400px;
            overflow-y: auto;
        }
        
        .toggle-raw {
            background: #34495e;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 mNAV Dashboard</h1>
            <p>MicroStrategy Net Asset Value Tracking & Portfolio Analysis</p>
        </div>
        
        <div class="update-section">
            <button id="updateBtn" class="update-btn" onclick="runUpdate()">
                🔄 Update Data
            </button>
            <div id="loading" class="loading">
                <div class="spinner"></div>
                <p>Fetching latest market data...</p>
            </div>
        </div>
        
        <div class="content">
            <div id="error-container"></div>
            <div id="data-container">
                <p style="text-align: center; color: #7f8c8d; font-style: italic;">
                    Click "Update Data" to fetch the latest mNAV information
                </p>
            </div>
        </div>
    </div>

    <script>
        let currentData = null;
        
        async function runUpdate() {
            const updateBtn = document.getElementById('updateBtn');
            const loading = document.getElementById('loading');
            const errorContainer = document.getElementById('error-container');
            const dataContainer = document.getElementById('data-container');
            
            // Show loading state
            updateBtn.disabled = true;
            updateBtn.textContent = 'Updating...';
            loading.style.display = 'block';
            errorContainer.innerHTML = '';
            
            try {
                const response = await fetch('/api/update', {
                    method: 'POST',
                });
                
                const data = await response.json();
                
                if (data.success) {
                    currentData = data;
                    renderData(data);
                } else {
                    showError(data.error || 'Update failed');
                }
                
            } catch (error) {
                showError('Network error: ' + error.message);
            } finally {
                // Hide loading state
                updateBtn.disabled = false;
                updateBtn.textContent = '🔄 Update Data';
                loading.style.display = 'none';
            }
        }
        
        function showError(message) {
            const errorContainer = document.getElementById('error-container');
            errorContainer.innerHTML = '<div class="error">❌ ' + message + '</div>';
        }
        
        function renderData(data) {
            const container = document.getElementById('data-container');
            
            let html = '';
            
            // Key Metrics Grid
            html += '<div class="metrics-grid">';
            
            if (data.bitcoin_price) {
                html += '<div class="metric-card bitcoin">';
                html += '<div class="metric-value">$' + data.bitcoin_price + '</div>';
                html += '<div class="metric-label">Bitcoin Price</div>';
                html += '</div>';
            }
            
            if (data.mstr_price) {
                html += '<div class="metric-card">';
                html += '<div class="metric-value">$' + data.mstr_price + '</div>';
                html += '<div class="metric-label">MSTR Stock Price</div>';
                html += '</div>';
            }
            
            if (data.fbtc_price) {
                html += '<div class="metric-card">';
                html += '<div class="metric-value">$' + data.fbtc_price + '</div>';
                html += '<div class="metric-label">FBTC Price</div>';
                html += '</div>';
            }
            
            if (data.market_cap) {
                html += '<div class="metric-card">';
                html += '<div class="metric-value">$' + data.market_cap + '</div>';
                html += '<div class="metric-label">Market Cap</div>';
                html += '</div>';
            }
            
            if (data.premium) {
                html += '<div class="metric-card premium">';
                html += '<div class="metric-value">' + data.premium + '</div>';
                html += '<div class="metric-label">Premium to NAV</div>';
                html += '</div>';
            }
            
            if (data.ratio) {
                html += '<div class="metric-card">';
                html += '<div class="metric-value">' + data.ratio + '</div>';
                html += '<div class="metric-label">mNAV Ratio</div>';
                html += '</div>';
            }
            
            html += '</div>';
            
            // Bitcoin Holdings & Analysis Section
            if (data.bitcoin_holdings || data.bitcoin_value || data.mnav) {
                html += '<div class="section">';
                html += '<h2 class="section-title">🪙 Bitcoin Holdings & Analysis</h2>';
                html += '<table class="data-table">';
                html += '<thead><tr><th>Metric</th><th>Value</th></tr></thead>';
                html += '<tbody>';
                
                if (data.bitcoin_holdings) {
                    html += '<tr><td>Total BTC Holdings</td><td>' + Number(data.bitcoin_holdings).toLocaleString() + ' BTC</td></tr>';
                }
                if (data.bitcoin_value) {
                    html += '<tr><td>Bitcoin Value</td><td>$' + data.bitcoin_value + '</td></tr>';
                }
                if (data.mnav) {
                    html += '<tr><td>Current mNAV</td><td>' + data.mnav + '</td></tr>';
                }
                if (data.ratio) {
                    html += '<tr><td>mNAV Ratio</td><td>' + data.ratio + '</td></tr>';
                }
                if (data.premium) {
                    html += '<tr><td>Premium to NAV</td><td>' + data.premium + '</td></tr>';
                }
                
                html += '</tbody></table>';
                html += '</div>';
            }
            
            // Market Trend Analysis
            if (data.market_trend && (data.market_trend.previous_mnav || data.market_trend.current_mnav)) {
                html += '<div class="section">';
                html += '<h2 class="section-title">📈 Market Trend Analysis</h2>';
                html += '<table class="data-table">';
                html += '<thead><tr><th>Period</th><th>mNAV</th><th>Premium</th><th>Trend</th></tr></thead>';
                html += '<tbody>';
                
                if (data.market_trend.previous_mnav) {
                    html += '<tr><td>Previous</td><td>' + data.market_trend.previous_mnav + '</td><td>' + 
                           (data.market_trend.previous_premium || 'N/A') + '</td><td>-</td></tr>';
                }
                if (data.market_trend.current_mnav) {
                    let trendIcon = '';
                    if (data.market_trend.trend_direction === 'Up') trendIcon = '📈';
                    else if (data.market_trend.trend_direction === 'Down') trendIcon = '📉';
                    else trendIcon = '➡️';
                    
                    html += '<tr><td>Current</td><td>' + data.market_trend.current_mnav + '</td><td>' + 
                           (data.market_trend.current_premium || 'N/A') + '</td><td>' + trendIcon + ' ' + 
                           (data.market_trend.trend_direction || 'Stable') + '</td></tr>';
                }
                
                if (data.market_trend.trend_description) {
                    html += '<tr><td colspan="4" style="background: #f8f9fa; font-style: italic;">' + 
                           data.market_trend.trend_description + '</td></tr>';
                }
                
                html += '</tbody></table>';
                html += '</div>';
            }
            
            // Portfolio Analysis
            if (data.portfolio && (data.portfolio.holdings || data.portfolio.net_value)) {
                html += '<div class="section">';
                html += '<h2 class="section-title">💼 Portfolio Analysis</h2>';
                
                // Portfolio Holdings
                if (data.portfolio.holdings && data.portfolio.holdings.length > 0) {
                    html += '<h3 style="margin-bottom: 10px;">Current Holdings</h3>';
                    html += '<table class="data-table">';
                    html += '<thead><tr><th>Symbol</th><th>Shares</th><th>Price</th><th>Value</th></tr></thead>';
                    html += '<tbody>';
                    
                    data.portfolio.holdings.forEach(holding => {
                        html += '<tr>';
                        html += '<td><strong>' + holding.symbol + '</strong></td>';
                        html += '<td>' + (holding.shares ? Number(holding.shares).toLocaleString() : 'N/A') + '</td>';
                        html += '<td>' + (holding.price ? '$' + holding.price : 'N/A') + '</td>';
                        html += '<td>' + (holding.value ? '$' + Number(holding.value).toLocaleString() : 'N/A') + '</td>';
                        html += '</tr>';
                    });
                    
                    html += '</tbody></table>';
                }
                
                // Portfolio Metrics
                html += '<h3 style="margin: 20px 0 10px 0;">Portfolio Metrics</h3>';
                html += '<table class="data-table">';
                html += '<thead><tr><th>Metric</th><th>Value</th></tr></thead>';
                html += '<tbody>';
                
                if (data.portfolio.net_value) {
                    html += '<tr><td>Net Portfolio Value</td><td>$' + Number(data.portfolio.net_value).toLocaleString() + '</td></tr>';
                }
                if (data.portfolio.net_bitcoin_value) {
                    html += '<tr><td>Net Bitcoin Value</td><td>' + data.portfolio.net_bitcoin_value + ' BTC</td></tr>';
                }
                if (data.portfolio.bitcoin_exposure) {
                    html += '<tr><td>Bitcoin Exposure</td><td>' + data.portfolio.bitcoin_exposure + ' BTC</td></tr>';
                }
                if (data.portfolio.current_ratio) {
                    html += '<tr><td>Current FBTC:MSTR Ratio</td><td>' + data.portfolio.current_ratio + '</td></tr>';
                }
                if (data.portfolio.target_ratio) {
                    html += '<tr><td>Target FBTC:MSTR Ratio</td><td>' + data.portfolio.target_ratio + '</td></tr>';
                }
                
                let balanceStatus = data.portfolio.is_balanced ? '✅ Balanced' : '⚠️ Needs Rebalancing';
                html += '<tr><td>Balance Status</td><td>' + balanceStatus + '</td></tr>';
                
                if (data.portfolio.recommendation) {
                    html += '<tr><td>Recommendation</td><td style="font-style: italic;">' + data.portfolio.recommendation + '</td></tr>';
                }
                
                html += '</tbody></table>';
                html += '</div>';
            }
            
            // Data Sources Section
            if (data.data_sources && data.data_sources.length > 0) {
                html += '<div class="section">';
                html += '<h2 class="section-title">📡 Data Sources & Freshness</h2>';
                html += '<table class="data-table">';
                html += '<thead><tr><th>Source</th><th>Provider</th><th>Price/Data</th><th>Last Updated</th><th>Method</th></tr></thead>';
                html += '<tbody>';
                
                data.data_sources.forEach(source => {
                    html += '<tr>';
                    html += '<td><strong>' + (source.name || 'Unknown') + '</strong></td>';
                    html += '<td>' + (source.source || 'N/A') + '</td>';
                    
                    let priceData = '';
                    if (source.price) priceData = '$' + source.price;
                    else if (source.holdings) priceData = source.holdings;
                    else priceData = 'N/A';
                    html += '<td>' + priceData + '</td>';
                    
                    html += '<td>' + (source.last_updated || 'N/A') + '</td>';
                    html += '<td>' + (source.method || 'N/A') + '</td>';
                    html += '</tr>';
                });
                
                html += '</tbody></table>';
                html += '</div>';
            }
            
            // Files Updated Section
            if (data.files_updated && data.files_updated.length > 0) {
                html += '<div class="section">';
                html += '<h2 class="section-title">📁 Files Updated</h2>';
                html += '<ul style="list-style-type: none; padding: 0;">';
                
                data.files_updated.forEach(file => {
                    html += '<li style="padding: 8px; background: #f8f9fa; margin: 5px 0; border-radius: 5px;">';
                    html += '<code>' + file + '</code>';
                    html += '</li>';
                });
                
                html += '</ul>';
                html += '</div>';
            }
            
            // Raw Output Section (collapsible)
            if (data.raw_output) {
                html += '<div class="section">';
                html += '<h2 class="section-title">📝 Raw Output</h2>';
                html += '<button class="toggle-raw" onclick="toggleRawOutput()">Show/Hide Raw Output</button>';
                html += '<div id="raw-output" class="raw-output" style="display: none;">';
                html += data.raw_output;
                html += '</div>';
                html += '</div>';
            }
            
            // Timestamp
            if (data.timestamp) {
                html += '<div class="timestamp">Last updated: ' + data.timestamp + '</div>';
            }
            
            container.innerHTML = html;
        }
        
        function toggleRawOutput() {
            const rawOutput = document.getElementById('raw-output');
            rawOutput.style.display = rawOutput.style.display === 'none' ? 'block' : 'none';
        }
        
        // Auto-refresh every 5 minutes
        setInterval(() => {
            if (currentData) {
                runUpdate();
            }
        }, 5 * 60 * 1000);
    </script>
</body>
</html>`

	t, err := template.New("index").Parse(tmpl)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

// handleUpdate runs the update-mnav script and returns parsed results
func (ws *WebServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Change to workspace root directory
	originalDir, err := os.Getwd()
	if err != nil {
		ws.sendError(w, "Could not get current directory: "+err.Error())
		return
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(ws.workspaceRoot)
	if err != nil {
		ws.sendError(w, "Could not change to workspace directory: "+err.Error())
		return
	}

	// Run the update script
	cmd := exec.Command("./sh/update-mnav")
	output, err := cmd.CombinedOutput()

	result := ScriptOutput{
		Timestamp: time.Now().Format("2006-01-02 15:04:05 MST"),
		RawOutput: string(output),
		Success:   err == nil,
	}

	if err != nil {
		result.Error = err.Error()
	} else {
		// Parse the output to extract key metrics
		ws.parseScriptOutput(&result, string(output))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseScriptOutput extracts comprehensive data from the script output
func (ws *WebServer) parseScriptOutput(result *ScriptOutput, output string) {
	lines := strings.Split(output, "\n")
	currentSection := ""

	// Initialize structures
	result.DataSources = []DataSource{}
	result.FilesUpdated = []string{}
	result.Portfolio.Holdings = []PortfolioHolding{}

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Track sections
		if strings.Contains(line, "🪙 Bitcoin Price Data:") {
			currentSection = "bitcoin-data"
		} else if strings.Contains(line, "📈 MSTR Stock Data:") {
			currentSection = "mstr-data"
		} else if strings.Contains(line, "📊 FBTC Price Data:") {
			currentSection = "fbtc-data"
		} else if strings.Contains(line, "🪙 MSTR Bitcoin Holdings:") {
			currentSection = "holdings-data"
		} else if strings.Contains(line, "📊 Portfolio Rebalancing Analysis:") {
			currentSection = "portfolio"
		} else if strings.Contains(line, "📊 Market Trend:") {
			currentSection = "market-trend"
		} else if strings.Contains(line, "📁 Files Updated:") {
			currentSection = "files"
		}

		// Extract basic metrics
		if strings.Contains(line, "💰 Current Price: $") {
			parts := strings.Split(line, "$")
			if len(parts) > 1 {
				price := strings.TrimSpace(parts[1])
				switch currentSection {
				case "bitcoin-data":
					result.BitcoinPrice = price
				case "mstr-data":
					result.MSTRPrice = price
				case "fbtc-data":
					result.FBTCPrice = price
				}
			}
		}

		// Extract from Latest MSTR Analysis section
		if strings.Contains(line, "Stock Price: $") {
			parts := strings.Split(line, "$")
			if len(parts) > 1 {
				result.MSTRPrice = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "Market Cap: $") {
			parts := strings.Split(line, "$")
			if len(parts) > 1 {
				result.MarketCap = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "Bitcoin Holdings:") && strings.Contains(line, "BTC") {
			parts := strings.Fields(line)
			for j, part := range parts {
				if strings.Contains(part, "BTC") && j > 0 {
					result.BitcoinHoldings = strings.Replace(parts[j-1], ",", "", -1)
					break
				}
			}
		}

		if strings.Contains(line, "Bitcoin Value: $") {
			parts := strings.Split(line, "$")
			if len(parts) > 1 {
				value := strings.TrimSpace(parts[1])
				value = strings.Replace(value, " billion", "B", 1)
				value = strings.Replace(value, " million", "M", 1)
				result.BitcoinValue = value
			}
		}

		if strings.Contains(line, "mNAV Ratio:") {
			parts := strings.Split(line, "mNAV Ratio:")
			if len(parts) > 1 {
				result.Ratio = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "Premium:") && strings.Contains(line, "%") {
			parts := strings.Split(line, "Premium:")
			if len(parts) > 1 {
				result.Premium = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "📊 Current mNAV") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				result.MNAV = strings.TrimSpace(parts[1])
			}
		}

		// Extract Market Trend data
		if currentSection == "market-trend" {
			if strings.Contains(line, "Previous mNAV:") {
				parts := strings.Split(line, "Previous mNAV:")
				if len(parts) > 1 {
					text := strings.TrimSpace(parts[1])
					// Extract mNAV and premium from "1.49 (48.7% premium)"
					if strings.Contains(text, "(") {
						mnavPart := strings.Split(text, "(")[0]
						premiumPart := strings.Split(text, "(")[1]
						premiumPart = strings.Replace(premiumPart, "premium)", "", 1)
						result.MarketTrend.PreviousMNAV = strings.TrimSpace(mnavPart)
						result.MarketTrend.PreviousPremium = strings.TrimSpace(premiumPart)
					}
				}
			}
			if strings.Contains(line, "Current mNAV:") {
				parts := strings.Split(line, "Current mNAV:")
				if len(parts) > 1 {
					text := strings.TrimSpace(parts[1])
					if strings.Contains(text, "(") {
						mnavPart := strings.Split(text, "(")[0]
						premiumPart := strings.Split(text, "(")[1]
						premiumPart = strings.Replace(premiumPart, "premium)", "", 1)
						result.MarketTrend.CurrentMNAV = strings.TrimSpace(mnavPart)
						result.MarketTrend.CurrentPremium = strings.TrimSpace(premiumPart)
					}
				}
			}
			if strings.Contains(line, "Trend:") {
				parts := strings.Split(line, "Trend:")
				if len(parts) > 1 {
					trendText := strings.TrimSpace(parts[1])
					result.MarketTrend.TrendDescription = trendText
					if strings.Contains(trendText, "increasing") {
						result.MarketTrend.TrendDirection = "Up"
					} else if strings.Contains(trendText, "decreasing") {
						result.MarketTrend.TrendDirection = "Down"
					} else {
						result.MarketTrend.TrendDirection = "Stable"
					}
				}
			}
		}

		// Extract Portfolio data
		if currentSection == "portfolio" {
			// Extract holdings
			if strings.Contains(line, "FBTC:") && strings.Contains(line, "shares") {
				holding := ws.parsePortfolioLine(line, "FBTC")
				if holding.Symbol != "" {
					result.Portfolio.Holdings = append(result.Portfolio.Holdings, holding)
				}
			}
			if strings.Contains(line, "MSTR:") && strings.Contains(line, "shares") {
				holding := ws.parsePortfolioLine(line, "MSTR")
				if holding.Symbol != "" {
					result.Portfolio.Holdings = append(result.Portfolio.Holdings, holding)
				}
			}

			// Extract portfolio metrics
			if strings.Contains(line, "Net Value: $") {
				parts := strings.Split(line, "$")
				if len(parts) > 1 {
					result.Portfolio.NetValue = strings.Split(strings.TrimSpace(parts[1]), " ")[0]
				}
			}
			if strings.Contains(line, "Net Bitcoin Value:") && strings.Contains(line, "BTC") {
				parts := strings.Fields(line)
				for j, part := range parts {
					if strings.Contains(part, "BTC") && j > 0 {
						result.Portfolio.NetBitcoinValue = parts[j-1]
						break
					}
				}
			}
			if strings.Contains(line, "Bitcoin Exposure:") && strings.Contains(line, "BTC") {
				parts := strings.Fields(line)
				for j, part := range parts {
					if strings.Contains(part, "BTC") && j > 0 {
						result.Portfolio.BitcoinExposure = parts[j-1]
						break
					}
				}
			}
			if strings.Contains(line, "Current Portfolio FBTC:MSTR Ratio:") {
				parts := strings.Split(line, "Ratio:")
				if len(parts) > 1 {
					result.Portfolio.CurrentRatio = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "Target FBTC:MSTR Ratio:") {
				parts := strings.Split(line, "Ratio:")
				if len(parts) > 1 {
					result.Portfolio.TargetRatio = strings.TrimSpace(parts[1])
				}
			}
			if strings.Contains(line, "✅ Portfolio is well balanced") {
				result.Portfolio.IsBalanced = true
				result.Portfolio.Recommendation = "Portfolio is well balanced"
			}
			if strings.Contains(line, "💡 No rebalancing needed") {
				result.Portfolio.Recommendation = "No rebalancing needed - continue monitoring"
			}
		}

		// Extract Data Sources
		if currentSection == "bitcoin-data" || currentSection == "mstr-data" || currentSection == "fbtc-data" || currentSection == "holdings-data" {
			dataSource := ws.parseDataSource(lines, i, currentSection)
			if dataSource.Name != "" {
				result.DataSources = append(result.DataSources, dataSource)
			}
		}

		// Extract Files Updated
		if currentSection == "files" {
			if strings.Contains(line, "CSV:") || strings.Contains(line, "Chart:") || strings.Contains(line, "Bitcoin Data:") {
				result.FilesUpdated = append(result.FilesUpdated, strings.TrimSpace(line))
			}
		}
	}
}

// parsePortfolioLine extracts portfolio holding data from a line
func (ws *WebServer) parsePortfolioLine(line, symbol string) PortfolioHolding {
	// Example: "FBTC: 833.65 shares @ $95.45 = $79571.89"
	holding := PortfolioHolding{Symbol: symbol}

	parts := strings.Split(line, "shares")
	if len(parts) >= 2 {
		// Extract shares
		beforeShares := strings.TrimSpace(parts[0])
		sharesParts := strings.Fields(beforeShares)
		if len(sharesParts) > 0 {
			holding.Shares = sharesParts[len(sharesParts)-1]
		}

		// Extract price and value
		afterShares := parts[1]
		if strings.Contains(afterShares, "@") && strings.Contains(afterShares, "=") {
			pricePart := strings.Split(afterShares, "@")[1]
			pricePart = strings.Split(pricePart, "=")[0]
			pricePart = strings.Replace(pricePart, "$", "", -1)
			holding.Price = strings.TrimSpace(pricePart)

			valuePart := strings.Split(afterShares, "=")[1]
			valuePart = strings.Replace(valuePart, "$", "", -1)
			holding.Value = strings.TrimSpace(valuePart)
		}
	}

	return holding
}

// parseDataSource extracts data source information from lines
func (ws *WebServer) parseDataSource(lines []string, currentIndex int, sectionType string) DataSource {
	dataSource := DataSource{}

	// Look for source info in the next few lines
	for i := currentIndex; i < len(lines) && i < currentIndex+8; i++ {
		line := strings.TrimSpace(lines[i])

		if strings.Contains(line, "📊 Source:") {
			parts := strings.Split(line, "Source:")
			if len(parts) > 1 {
				dataSource.Source = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "💰 Current Price: $") {
			parts := strings.Split(line, "$")
			if len(parts) > 1 {
				dataSource.Price = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "📅 Last Updated:") || strings.Contains(line, "📅 Fetched:") {
			parts := strings.Split(line, ":")
			if len(parts) > 2 {
				dataSource.LastUpdated = strings.TrimSpace(strings.Join(parts[1:], ":"))
			}
		}

		if strings.Contains(line, "📁 Data File:") {
			parts := strings.Split(line, "File:")
			if len(parts) > 1 {
				dataSource.DataFile = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "🔄 Collection Method:") {
			parts := strings.Split(line, "Method:")
			if len(parts) > 1 {
				dataSource.Method = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "🪙 Total Holdings:") {
			parts := strings.Split(line, "Holdings:")
			if len(parts) > 1 {
				dataSource.Holdings = strings.TrimSpace(parts[1])
			}
		}

		// Stop if we hit a new section
		if strings.Contains(line, "🪙") || strings.Contains(line, "📈") || strings.Contains(line, "📊") {
			if i > currentIndex {
				break
			}
		}
	}

	// Set name based on section type
	switch sectionType {
	case "bitcoin-data":
		dataSource.Name = "Bitcoin Price"
	case "mstr-data":
		dataSource.Name = "MSTR Stock"
	case "fbtc-data":
		dataSource.Name = "FBTC Price"
	case "holdings-data":
		dataSource.Name = "MSTR Holdings"
	}

	return dataSource
}

// sendError sends an error response
func (ws *WebServer) sendError(w http.ResponseWriter, message string) {
	result := ScriptOutput{
		Timestamp: time.Now().Format("2006-01-02 15:04:05 MST"),
		Success:   false,
		Error:     message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(result)
}

func main() {
	server := NewWebServer()

	// Routes
	http.HandleFunc("/", server.serveHTML)
	http.HandleFunc("/api/update", server.handleUpdate)

	port := ":8080"
	fmt.Printf("🌐 mNAV Web Dashboard starting on http://localhost%s\n", port)
	fmt.Println("📊 Open your browser and click 'Update Data' to fetch latest mNAV information")

	log.Fatal(http.ListenAndServe(port, nil))
}
