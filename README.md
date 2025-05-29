# mNAV - Bitcoin Treasury Company Tracker

**Organized Data Pipeline for Bitcoin Treasury Analysis**

## Quick Start

```bash
# Build all tools
make build

# Demo the new structure
make demo

# Run complete MSTR analysis workflow
make workflow-mstr
```

## 🏗️ Architecture Overview

The mNAV project is organized into three distinct data flow categories:

- **🗂️ Data Collection** - Gather raw data from external sources
- **🔍 Data Interpretation** - Parse and extract structured information  
- **📊 Data Analysis** - Calculate metrics and generate insights

This separation ensures clear workflows and maintainable code.

## 🛠️ Available Tools

### Collection Tools 🗂️
```bash
# Download SEC filings
./bin/collection/edgar-data -ticker=MSTR -start=2023-01-01
```

### Interpretation Tools 🔍  
```bash
# Extract Bitcoin transactions from filings
./bin/interpretation/bitcoin-parser -ticker=MSTR
```

### Analysis Tools 📊
```bash
# Calculate mNAV metrics
./bin/analysis/mnav-calculator -symbols=MSTR,SMLR,MARA -verbose
```

## 📊 Current Capabilities

### ✅ Working Features
- **mNAV Calculation**: Net asset value based on Bitcoin holdings
- **Price Target Analysis**: Stock price targets for different mNAV levels
- **Real-time Data**: Bitcoin prices, stock prices, market caps
- **Multiple Companies**: MSTR, SMLR, MARA, Metaplanet support
- **Transaction Parsing**: Extract Bitcoin purchases from SEC filings
- **Web Scraping**: Real-time MSTR holdings from company website

### 🔄 In Development  
- **Cross-package Dependencies**: Currently being resolved
- **Enhanced Parsing**: AI-powered transaction extraction
- **Portfolio Analysis**: Multi-company comparative metrics
- **Historical Tracking**: Time-series analysis of mNAV trends

## 🚀 Quick Examples

### Analyze Single Company
```bash
# Complete MSTR analysis
./bin/analysis/mnav-calculator -symbols=MSTR -verbose
```

Output:
```
📊 DATA ANALYSIS - mNAV Calculator
==================================

🏢 Analyzing MSTR...
------------------------
📈 Stock Price: $347.80
💎 Bitcoin Holdings: 331,200.00 BTC  
🏦 Market Cap: $69,400.00 million
💰 Bitcoin Value: $32,832.32 million
📊 mNAV: 2.11
⏱️  Days to Cover mNAV: 89.3 days
📈 Daily BTC Accumulation: 397.44 BTC
```

### Build by Category
```bash
# Build only analysis tools
make analysis

# Build only collection tools  
make collection

# Build only interpretation tools
make interpretation
```

## 📁 Project Structure

```
mNAV/
├── cmd/                    # Categorized command-line tools
│   ├── collection/         # Data collection commands
│   ├── interpretation/     # Data parsing commands
│   └── analysis/          # Metrics calculation commands
├── pkg/                   # Categorized packages
│   ├── collection/        # External data gathering
│   ├── interpretation/    # Data parsing and extraction
│   ├── analysis/         # Calculations and metrics
│   └── shared/           # Common components
└── bin/                  # Compiled binaries by category
    ├── collection/
    ├── interpretation/
    └── analysis/
```

## 🔧 Build System

```bash
# Category-based building
make collection      # Build data collection tools
make interpretation  # Build data interpretation tools  
make analysis       # Build data analysis tools

# Individual tools
make edgar-data      # SEC filing collector
make bitcoin-parser  # Bitcoin transaction extractor
make mnav-calculator # mNAV metrics calculator

# Utilities
make clean          # Clean all build artifacts
make test           # Run tests
make deps           # Download dependencies
```

## 💡 Key Benefits

1. **Clear Data Flow**: Collection → Interpretation → Analysis
2. **Category Identification**: Every command shows its category
3. **Independent Execution**: Run any stage independently
4. **Maintainable Code**: Clear separation of concerns
5. **Scalable Architecture**: Easy to add new tools in any category

## 📚 Documentation

- **[Architecture Guide](ARCHITECTURE.md)** - Detailed architecture explanation
- **[Environment Setup](ENV_SETUP.md)** - API keys and configuration
- **[Build Guide](Makefile)** - Complete build system reference

## 🤝 Contributing

The categorized structure makes contributions easier:

- **Collection**: Add new data sources (APIs, scrapers)
- **Interpretation**: Improve parsing algorithms, add AI models
- **Analysis**: Create new metrics, visualization tools

## 📈 Supported Companies

- **MSTR** (MicroStrategy) - Complete support with real-time data
- **SMLR** (Semler Scientific) - Full mNAV analysis  
- **MARA** (Marathon Digital) - Market cap and holdings tracking
- **3350.T** (Metaplanet) - International support

## 🎯 Roadmap

### Phase 1: Foundation (Current)
- ✅ Categorized architecture
- ✅ Core mNAV calculations  
- 🔄 Dependency resolution
- 🔄 Enhanced documentation

### Phase 2: Enhanced Interpretation
- 🔄 AI-powered parsing
- 📋 Shares outstanding extraction
- 📋 Financial metrics parsing
- 📋 Validation and quality scoring

### Phase 3: Advanced Analysis
- 📋 Portfolio management
- 📋 Risk analysis
- 📋 Forecasting models
- 📋 Automated reporting

### Phase 4: Platform
- 📋 Web interface
- 📋 Real-time monitoring
- 📋 Alert system
- 📋 API service

---

**Status**: ✅ Core functionality working | 🔄 Dependencies being resolved | 📋 Architecture established 