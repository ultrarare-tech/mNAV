# mNAV Architecture

## Overview

The mNAV application is a sophisticated Bitcoin treasury analysis tool built in Go. It follows a clean, modular architecture organized around three main data flow categories:

1. **Data Collection** - Gathering raw data from external sources
2. **Data Interpretation** - Parsing and extracting structured information
3. **Data Analysis** - Calculating metrics and generating insights

## System Architecture

```
mNAV/
├── cmd/                          # Command-line tools organized by function
│   ├── collection/               # Data gathering commands
│   │   ├── bitcoin-historical/   # Historical Bitcoin price collection
│   │   ├── stock-data/          # Stock data from FMP + Alpha Vantage
│   │   └── edgar-data/          # SEC filing downloads
│   ├── analysis/                # Analysis and calculation commands
│   │   ├── mnav-historical/     # Historical mNAV calculation
│   │   ├── mnav-chart/          # Interactive chart generation
│   │   └── comprehensive-analysis/ # Complete analysis suite
│   └── interpretation/          # Data parsing and extraction
│       └── bitcoin-parser/      # Bitcoin transaction extraction
├── pkg/                         # Shared packages and libraries
│   ├── collection/              # External API clients
│   │   ├── fmp/                # Financial Modeling Prep client
│   │   ├── alphavantage/       # Alpha Vantage client
│   │   └── coinmarketcap/      # CoinMarketCap client
│   ├── analysis/               # Calculation and metrics
│   │   └── metrics/            # mNAV and financial calculations
│   └── shared/                 # Common utilities
│       ├── models/             # Data structures
│       ├── config/             # Configuration management
│       └── utils/              # Utility functions
├── data/                       # Data storage (created at runtime)
│   ├── bitcoin-prices/         # Historical Bitcoin price data
│   ├── stock-data/            # Stock prices and company data
│   ├── analysis/              # Analysis results and mNAV data
│   └── charts/                # Generated charts and visualizations
└── docs/                      # Documentation
    └── mNAV_CHARTING.md      # Charting system documentation
```

## Core Components

### Data Collection Layer

**Purpose**: Gather raw data from external sources

**Components**:
- `bitcoin-historical`: Collects historical Bitcoin prices from CoinGecko
- `stock-data`: Fetches stock prices, market cap, and company data from FMP and Alpha Vantage
- `edgar-data`: Downloads SEC filings from EDGAR database

**Key Features**:
- Professional API integrations (FMP, Alpha Vantage)
- Rate limiting and error handling
- Data persistence and caching
- Multiple data source support

### Data Interpretation Layer

**Purpose**: Parse and extract structured information from raw data

**Components**:
- `bitcoin-parser`: Extracts Bitcoin transaction data from SEC filings using Grok AI

**Key Features**:
- AI-powered text extraction
- Natural language processing for financial documents
- Validation against known sources
- Structured data output

### Data Analysis Layer

**Purpose**: Calculate metrics and generate insights

**Components**:
- `mnav-historical`: Calculates historical mNAV ratios and premiums
- `mnav-chart`: Generates interactive charts and visualizations
- `comprehensive-analysis`: Complete analysis suite with multiple metrics

**Key Features**:
- Historical mNAV calculation
- Premium/discount analysis
- Interactive chart generation
- Multiple export formats (HTML, CSV, JSON)

## Data Flow

```
1. Collection Phase
   ├── Bitcoin Prices (CoinGecko) → bitcoin-historical
   ├── Stock Data (FMP + Alpha Vantage) → stock-data
   └── SEC Filings (EDGAR) → edgar-data

2. Interpretation Phase
   └── SEC Filings → bitcoin-parser → Bitcoin Transactions

3. Analysis Phase
   ├── All Data Sources → mnav-historical → Historical mNAV
   └── mNAV Data → mnav-chart → Interactive Charts
```

## API Integrations

### Financial Modeling Prep (FMP)
- **Purpose**: Stock prices, market cap, company profiles
- **Endpoints**: Historical prices, current quotes, company profiles
- **Rate Limits**: 250 calls/day (free tier)

### Alpha Vantage
- **Purpose**: Shares outstanding, company fundamentals
- **Endpoints**: Company overview, financial metrics
- **Rate Limits**: 5 calls/minute, 500/day (free tier)

### CoinGecko
- **Purpose**: Historical Bitcoin prices
- **Endpoints**: Market chart data
- **Rate Limits**: 10-50 calls/minute (free)

### Grok AI
- **Purpose**: Bitcoin transaction extraction from SEC filings
- **Usage**: Natural language processing of financial documents
- **Rate Limits**: Varies by plan

## Key Design Principles

### 1. Separation of Concerns
Each layer has a distinct responsibility:
- Collection: Data gathering only
- Interpretation: Parsing and extraction only  
- Analysis: Calculations and insights only

### 2. Interface-Driven Development
- All external APIs accessed through well-defined interfaces
- Easy to mock for testing
- Simple to swap implementations

### 3. Data Persistence
- All collected data is saved locally
- Enables offline analysis and reduces API calls
- Structured file organization for easy access

### 4. Error Handling and Resilience
- Comprehensive error handling at all levels
- Graceful degradation when APIs are unavailable
- Retry logic with exponential backoff

### 5. Observability
- Structured logging throughout the application
- Clear progress indicators for long-running operations
- Detailed error messages with context

## Configuration Management

### Environment Variables
```bash
FMP_API_KEY=your_financial_modeling_prep_key
ALPHA_VANTAGE_API_KEY=your_alpha_vantage_key
GROK_API_KEY=your_grok_api_key
```

### Configuration Files
- Company configurations for supported symbols
- API endpoint configurations
- Default parameters for analysis

## Data Models

### Core Entities

**BitcoinTransaction**
```go
type BitcoinTransaction struct {
    Date         time.Time
    BTCPurchased float64
    USDSpent     float64
    PricePerBTC  float64
    Source       string
    FilingType   string
}
```

**HistoricalMNAVPoint**
```go
type HistoricalMNAVPoint struct {
    Date              string
    StockPrice        float64
    BitcoinPrice      float64
    BitcoinHoldings   float64
    SharesOutstanding float64
    MarketCap         float64
    BitcoinValue      float64
    MNAV              float64
    MNAVPerShare      float64
    Premium           float64
}
```

## Testing Strategy

### Unit Tests
- Individual package testing
- Mock external dependencies
- Table-driven test patterns

### Integration Tests
- End-to-end workflow testing
- API integration validation
- Data consistency checks

### Validation Tests
- Cross-reference with known sources (SaylorTracker.com)
- Historical data accuracy verification
- Calculation validation

## Performance Considerations

### API Rate Limiting
- Respect API rate limits with built-in delays
- Batch requests where possible
- Cache responses to minimize API calls

### Data Processing
- Stream processing for large datasets
- Parallel processing where appropriate
- Memory-efficient data structures

### Storage Optimization
- Compressed JSON for large datasets
- Indexed data structures for fast lookups
- Cleanup of temporary files

## Security Considerations

### API Key Management
- Environment variable storage
- No hardcoded credentials
- Secure key rotation support

### Data Validation
- Input sanitization for all external data
- Schema validation for API responses
- Error handling for malformed data

### Network Security
- HTTPS for all external communications
- Certificate validation
- Timeout configurations

## Deployment and Operations

### Build System
- Makefile-based build automation
- Category-based building (collection, analysis, interpretation)
- Cross-platform compilation support

### Monitoring
- Structured logging with levels
- Progress tracking for long operations
- Error reporting and alerting

### Maintenance
- Automated dependency updates
- Regular data validation checks
- Performance monitoring

## Future Architecture Considerations

### Scalability
- Microservice decomposition for high-volume usage
- Database backend for large-scale data storage
- Distributed processing capabilities

### Real-time Processing
- WebSocket connections for live data
- Event-driven architecture
- Stream processing frameworks

### Web Interface
- REST API for web frontend
- Real-time dashboard capabilities
- User authentication and authorization

This architecture provides a solid foundation for Bitcoin treasury analysis while maintaining flexibility for future enhancements and scaling requirements. 