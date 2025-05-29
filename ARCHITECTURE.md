# mNAV Project Architecture

## Overview

The mNAV project is organized around a **data flow pipeline** that separates concerns into three distinct categories:

1. **🗂️ Data Collection** - Gather raw data from external sources
2. **🔍 Data Interpretation** - Parse and extract structured information  
3. **📊 Data Analysis** - Calculate metrics and generate insights

This separation makes the codebase more maintainable, testable, and allows for clear workflow execution.

## Directory Structure

```
mNAV/
├── cmd/                          # Command-line tools organized by category
│   ├── collection/               # Data Collection Tools
│   │   └── edgar-data/           # SEC filing downloader
│   ├── interpretation/           # Data Interpretation Tools  
│   │   └── bitcoin-parser/       # Bitcoin transaction extractor
│   ├── analysis/                 # Data Analysis Tools
│   │   └── mnav-calculator/      # mNAV metrics calculator
│   └── utilities/                # General utilities
│
├── pkg/                          # Package code organized by category
│   ├── collection/               # External data gathering
│   │   ├── edgar/                # SEC EDGAR client
│   │   ├── yahoo/                # Yahoo Finance API
│   │   ├── coinmarketcap/        # CoinMarketCap API
│   │   └── scraper/              # Web scraping utilities
│   │
│   ├── interpretation/           # Data parsing and extraction
│   │   ├── parser/               # Document parsers (regex, etc.)
│   │   ├── grok/                 # AI-enhanced parsing
│   │   ├── validators/           # Data validation
│   │   └── normalizers/          # Data normalization
│   │
│   ├── analysis/                 # Calculations and metrics
│   │   ├── metrics/              # mNAV, price targets, etc.
│   │   ├── portfolio/            # Multi-company analysis
│   │   ├── forecasting/          # Predictive analytics
│   │   └── reporting/            # Report generation
│   │
│   └── shared/                   # Common components
│       ├── models/               # Data structures
│       ├── storage/              # Data persistence
│       ├── config/               # Configuration management
│       └── utils/                # General utilities
│
├── bin/                          # Compiled binaries (organized by category)
│   ├── collection/
│   ├── interpretation/
│   ├── analysis/
│   └── legacy/
│
└── data/                         # Data storage
    └── edgar/companies/[SYMBOL]/ # Company-specific data
```

## Data Flow Pipeline

### Stage 1: Collection 🗂️

**Purpose**: Gather raw data from external sources without interpretation.

**Tools**:
- `edgar-data`: Downloads SEC filings (8-K, 10-Q, 10-K)
- Future: `yahoo-data`, `coinmarketcap-data`

**Example**:
```bash
# Download all MSTR filings from 2023
./bin/collection/edgar-data -ticker=MSTR -start=2023-01-01
```

**Output**: Raw HTML/XML files stored in `data/edgar/companies/MSTR/raw_filings/`

### Stage 2: Interpretation 🔍  

**Purpose**: Parse raw data and extract structured information.

**Tools**:
- `bitcoin-parser`: Extracts Bitcoin transactions from SEC filings
- Future: `shares-parser`, `financials-parser`

**Example**:
```bash
# Parse Bitcoin transactions from downloaded filings
./bin/interpretation/bitcoin-parser -ticker=MSTR
```

**Output**: Structured JSON files with extracted transactions, shares data, etc.

### Stage 3: Analysis 📊

**Purpose**: Calculate metrics and generate insights from structured data.

**Tools**:
- `mnav-calculator`: Calculates mNAV, price targets, days to cover
- Future: `portfolio-analyzer`, `risk-calculator`

**Example**:
```bash
# Calculate mNAV metrics for multiple companies
./bin/analysis/mnav-calculator -symbols=MSTR,SMLR,MARA -verbose
```

**Output**: Calculated metrics, reports, and insights

## Package Dependencies

### Collection Layer
- **Dependencies**: External APIs, HTTP clients
- **Exports**: Raw data files, API responses
- **No dependencies on**: Interpretation or Analysis layers

### Interpretation Layer  
- **Dependencies**: Shared models, storage interfaces
- **Exports**: Structured data objects
- **No dependencies on**: Collection layer (operates on stored files)

### Analysis Layer
- **Dependencies**: Shared models, structured data
- **Exports**: Calculated metrics, reports  
- **No dependencies on**: Collection or Interpretation layers

### Shared Components
- **Models**: Common data structures used across all layers
- **Storage**: Data persistence and retrieval interfaces
- **Config**: Configuration management
- **Utils**: General utilities

## Command Categories

All commands clearly identify their category:

### Collection Commands
```
🗂️  DATA COLLECTION - SEC EDGAR Filings
Collects raw SEC filing documents for future interpretation and analysis.
```

### Interpretation Commands  
```
🔍 DATA INTERPRETATION - Bitcoin Transaction Parser
Extracts Bitcoin transaction data from SEC filing documents.
```

### Analysis Commands
```
📊 DATA ANALYSIS - mNAV Calculator  
Calculates net asset value metrics for Bitcoin treasury companies.
```

## Build System

The Makefile is organized by categories:

```bash
# Build all tools by category
make collection      # Build data collection tools
make interpretation  # Build data interpretation tools  
make analysis       # Build data analysis tools

# Build individual tools
make edgar-data      # SEC filing collector
make bitcoin-parser  # Bitcoin transaction extractor
make mnav-calculator # mNAV metrics calculator

# Complete workflow example
make workflow-mstr   # Full MSTR analysis pipeline
```

## Usage Patterns

### Complete Workflow
```bash
# 1. Collect raw data
./bin/collection/edgar-data -ticker=MSTR -start=2023-01-01

# 2. Extract structured data  
./bin/interpretation/bitcoin-parser -ticker=MSTR

# 3. Calculate metrics
./bin/analysis/mnav-calculator -symbols=MSTR -verbose
```

### Selective Processing
```bash
# Only run analysis if data already exists
./bin/analysis/mnav-calculator -symbols=MSTR,SMLR,MARA

# Re-parse existing raw filings with new logic
./bin/interpretation/bitcoin-parser -ticker=MSTR

# Collect only specific filing types
./bin/collection/edgar-data -ticker=MSTR -filing-types="10-Q,10-K"
```

## Benefits of This Architecture

1. **Clear Separation of Concerns**: Each layer has a distinct responsibility
2. **Independent Development**: Teams can work on different layers independently  
3. **Testability**: Each layer can be tested in isolation
4. **Scalability**: Easy to add new tools in each category
5. **Workflow Clarity**: Users understand what each tool does
6. **Maintainability**: Dependencies are clearly defined and minimized
7. **Flexibility**: Can run partial workflows as needed

## Migration from Legacy

Legacy tools are preserved in `bin/legacy/` and can be built with `make legacy` during the transition period. The new categorized tools provide the same functionality with better organization.

## Future Enhancements

### Planned Tools

**Collection**:
- `yahoo-data`: Stock price and financial data collector
- `crypto-data`: Cryptocurrency price data collector  
- `news-data`: Financial news collector

**Interpretation**:
- `shares-parser`: Extract shares outstanding from filings
- `financials-parser`: Extract financial metrics
- `sentiment-analyzer`: Analyze news sentiment

**Analysis**:
- `portfolio-analyzer`: Multi-company portfolio analysis
- `risk-calculator`: Risk metrics and scenarios
- `report-generator`: Automated report creation

### Planned Features
- Web UI for workflow management
- Automated scheduling and monitoring
- Real-time data updates
- Advanced AI/ML interpretation models
- Integration with external analytics platforms 