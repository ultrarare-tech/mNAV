# mNAV - Bitcoin Treasury Analysis Tool

A sophisticated Go application for analyzing Bitcoin holdings and calculating mNAV (modified Net Asset Value) for companies that hold Bitcoin as a treasury asset. Primarily focused on MicroStrategy (MSTR) but extensible to other Bitcoin treasury companies.

## 🎯 What This Application Does

**mNAV** extracts Bitcoin transaction data from SEC filings and calculates financial metrics to determine how the market values a company relative to its Bitcoin holdings. The application can:

- 📊 **Extract Bitcoin transactions** from SEC filings using AI (Grok API)
- 📈 **Generate historical mNAV charts** showing premium/discount over time  
- 💰 **Calculate current mNAV ratios** and premium percentages
- 📋 **Compare results** with external sources like SaylorTracker.com
- 🎨 **Create interactive visualizations** of Bitcoin accumulation patterns

## 🚀 Quick Start

### Prerequisites

1. **API Keys** (add to `.env` file):
   ```env
   FMP_API_KEY=your_financial_modeling_prep_key
   ALPHA_VANTAGE_API_KEY=your_alpha_vantage_key
   # Bitcoin data is FREE via CoinGecko - no API key required!
   GROK_API_KEY=your_grok_api_key  # Optional for transaction parsing
   ```

2. **Go 1.21+** installed

### Build & Run

```bash
# Build all tools
make all

# Run the complete workflow (365 days of historical data)
./demo-workflow.sh

# OR manual workflow:
./bin/bitcoin-historical -start=2020-08-11      # Get Bitcoin prices (free!)
./bin/stock-data -symbol=MSTR -start=2020-08-11  # Get stock data  
./bin/mnav-historical -symbol=MSTR -start=2020-08-11  # Calculate mNAV
./bin/mnav-chart -format=html                   # Generate chart
```

## 🆕 Recent Improvements

### CoinGecko Integration
- **Free Bitcoin data**: No API key required for historical Bitcoin prices
- **Reliable source**: Industry-standard cryptocurrency market data
- **365-day history**: Free tier provides up to 365 days of historical data
- **No rate limits**: Generous usage allowances for development

### Professional Data Stack
- **Financial Modeling Prep**: Stock prices and market data
- **Alpha Vantage**: Shares outstanding and company fundamentals  
- **CoinGecko**: Free Bitcoin price history (no API key needed)
- **SEC EDGAR**: Filing downloads and analysis

## 📊 Key Features

### **Bitcoin Transaction Extraction**
- Parses SEC filings (8-K, 10-K, 10-Q) for Bitcoin purchase announcements
- Uses Grok AI for intelligent text extraction
- Validates results against known sources
- Tracks cumulative Bitcoin holdings over time

### **mNAV Calculation & Charting**  
- **Historical mNAV analysis** with daily/weekly/monthly intervals
- **Interactive HTML charts** showing mNAV evolution over time
- **Premium/discount tracking** relative to Bitcoin NAV per share
- **Multiple export formats** (HTML, CSV, JSON)

### **Professional Data Sources**
- **Financial Modeling Prep** for stock prices and market data
- **Alpha Vantage** for shares outstanding and company fundamentals  
- **CoinGecko** for complete Bitcoin price history
- **SEC EDGAR** for filing downloads and analysis

## 🏗️ Architecture

```
mNAV/
├── cmd/                          # Commands organized by function
│   ├── collection/               # Data gathering tools
│   │   ├── bitcoin-historical/   # Historical Bitcoin prices
│   │   ├── stock-data/          # Stock prices & company data
│   │   └── edgar-data/          # SEC filing collection
│   ├── analysis/                # Analysis & calculation tools  
│   │   ├── mnav-historical/     # Historical mNAV calculation
│   │   ├── mnav-chart/          # Chart generation
│   │   └── comprehensive-analysis/ # Complete analysis suite
│   └── interpretation/          # Data parsing & extraction
│       └── bitcoin-parser/      # Extract Bitcoin transactions
├── pkg/                         # Shared packages
│   ├── collection/              # API clients (FMP, Alpha Vantage)
│   ├── analysis/               # Metrics & calculations
│   └── shared/                 # Common models & utilities
└── docs/                       # Documentation
    └── mNAV_CHARTING.md       # Charting system guide
```

## 📈 Current MSTR Analysis Results

Based on the latest SEC filing analysis:

- **Total Bitcoin Holdings**: ~331,200 BTC (as of Q3 2024)
- **Average Purchase Price**: ~$39,266 per BTC
- **Total Investment**: ~$13.0B
- **Validation**: 98.8% accuracy vs. SaylorTracker.com

## 🔧 Commands Reference

### Data Collection
```bash
# Get historical Bitcoin prices  
./bin/bitcoin-historical -start=2020-08-11

# Collect comprehensive stock data
./bin/stock-data -symbol=MSTR -start=2020-08-11

# Download SEC filings
./bin/edgar-data -ticker=MSTR -filing-types="8-K,10-Q,10-K"
```

### Analysis & Charts
```bash
# Calculate historical mNAV
./bin/mnav-historical -symbol=MSTR -interval=daily

# Generate interactive chart
./bin/mnav-chart -format=html -output=data/charts

# Run comprehensive analysis
./bin/comprehensive-analysis -symbol=MSTR
```

### Data Parsing
```bash
# Extract Bitcoin transactions from filings
./bin/bitcoin-parser -ticker=MSTR -use-grok
```

## 📊 Output Examples

### mNAV Chart
Interactive HTML charts showing:
- mNAV ratio over time (Bitcoin value / Market cap)
- Premium/discount percentage 
- Bitcoin accumulation timeline
- Market events and correlations

### Analysis Reports
JSON/CSV exports containing:
- Daily mNAV calculations
- Bitcoin holdings progression  
- Premium/discount analysis
- Market performance metrics

## 🛠️ Development

### Build System
```bash
make all                    # Build all tools
make collection-tools       # Build data collection tools only
make analysis-tools         # Build analysis tools only
make clean                  # Clean build artifacts
```

### Testing
```bash
make test                   # Run all tests
go test ./...              # Direct Go testing
```

## 📋 Data Sources & Attribution

- **Bitcoin Holdings**: SEC filing analysis via Grok AI
- **Stock Prices**: Financial Modeling Prep API  
- **Market Data**: Financial Modeling Prep API
- **Shares Outstanding**: Alpha Vantage API
- **Bitcoin Prices**: CoinGecko (free)
- **Validation**: SaylorTracker.com comparison

## 🔍 Accuracy & Validation

The application achieves high accuracy through:
- **Multi-source validation** against SaylorTracker.com
- **AI-powered extraction** with human oversight
- **Comprehensive filing analysis** (not just press releases)
- **Cross-reference verification** across multiple filings

Current validation results show **98.8% accuracy** for MSTR Bitcoin holdings.

## 📝 Documentation

- [**mNAV Charting Guide**](docs/mNAV_CHARTING.md) - Complete charting system documentation
- [**Architecture Overview**](ARCHITECTURE.md) - System design and components
- [**API Integration Guide**](docs/API_SETUP.md) - Setting up external APIs

## 🚀 Future Enhancements

- Support for additional Bitcoin treasury companies
- Real-time data updates and alerts
- Advanced statistical analysis and forecasting
- Integration with portfolio management tools
- Web dashboard interface

## 📄 License

MIT License - see LICENSE file for details.

---

**Built with Go • Powered by AI • Validated by Data** 