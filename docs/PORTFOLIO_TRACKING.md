# Portfolio Tracking System

The mNAV project now includes a comprehensive portfolio tracking system for managing and analyzing investment portfolios, with special focus on Bitcoin exposure through FBTC and MSTR holdings.

## Overview

The portfolio tracking system provides:

- **Import & Storage**: Parse Fidelity CSV exports and store portfolio snapshots
- **Historical Tracking**: Maintain portfolio evolution over time
- **Asset Allocation Analysis**: Detailed breakdown of holdings and allocations
- **Bitcoin Exposure Metrics**: Track total Bitcoin exposure via FBTC + MSTR
- **Rebalancing Calculations**: Calculate optimal trades to achieve target ratios
- **Performance Analytics**: Track returns, volatility, and drawdown metrics

## Quick Start

### 1. Build Portfolio Tools

```bash
make portfolio-tools
```

### 2. Import Portfolio Data

Download your portfolio CSV from Fidelity and import it:

```bash
./bin/portfolio-importer -csv Portfolio_Positions_Jun-11-2025.csv -v
```

This will:
- Parse the CSV file
- Store processed data in `data/portfolio/processed/`
- Archive raw CSV in `data/portfolio/raw/`
- Display portfolio summary

### 3. Analyze Portfolio

View latest portfolio analysis:

```bash
./bin/portfolio-analyzer -latest
```

View specific date:

```bash
./bin/portfolio-analyzer -date 2025-06-11
```

Calculate rebalancing for 5:1 FBTC:MSTR ratio:

```bash
./bin/portfolio-analyzer -latest -rebalance 5.0
```

View historical performance:

```bash
./bin/portfolio-analyzer -historical
```

## Data Structure

### Directory Layout

```
data/portfolio/
├── raw/                 # Original CSV files
├── processed/           # JSON snapshots by date  
├── analysis/           # Analysis results
└── historical/         # Historical summaries
```

### Supported CSV Format

The system parses Fidelity portfolio CSV exports with these fields:
- Account Number, Account Name
- Symbol, Description
- Quantity, Last Price, Last Price Change
- Current Value, Today's Gain/Loss Dollar/Percent
- Total Gain/Loss Dollar/Percent
- Percent Of Account, Cost Basis Total, Average Cost Basis
- Type

## Key Features

### Asset Allocation Analysis

- **FBTC**: Direct Bitcoin exposure via Fidelity's Bitcoin ETF
- **MSTR**: Indirect Bitcoin exposure via MicroStrategy stock
- **GLD**: Gold exposure via SPDR Gold Trust
- **Other**: All other holdings (TSLA, etc.)

### Bitcoin Exposure Calculation

Total Bitcoin Exposure = FBTC Value + MSTR Value

The system calculates:
- Total Bitcoin exposure percentage
- FBTC to MSTR ratio
- Individual asset percentages

### Rebalancing Recommendations

Input target FBTC:MSTR ratio to get:
- Specific share amounts to trade
- Dollar amounts involved
- New allocation percentages
- Warning for large trades (>10% of portfolio)

### Performance Metrics

- Total return (dollar and percentage)
- Compound Annual Growth Rate (CAGR)
- Volatility (annualized standard deviation)
- Maximum drawdown
- Period analysis

## Command Reference

### Portfolio Importer

```bash
./bin/portfolio-importer [options]

Options:
  -csv string     Path to portfolio CSV file (required)
  -data string    Directory to store processed data (default: data/portfolio/processed)
  -v             Verbose output
```

### Portfolio Analyzer

```bash
./bin/portfolio-analyzer [options]

Options:
  -data string      Directory containing processed data (default: data/portfolio/processed)
  -latest          Analyze latest portfolio
  -date string     Analyze specific date (YYYY-MM-DD)
  -rebalance string Calculate rebalancing for target FBTC:MSTR ratio
  -historical      Show historical summary
  -performance     Show performance metrics
  -v              Verbose output (shows all positions)
```

## Workflows

### Regular Portfolio Import

1. Export portfolio from Fidelity as CSV
2. Import into system:
   ```bash
   ./bin/portfolio-importer -csv Portfolio_Positions_$(date +%b-%d-%Y).csv
   ```
3. Analyze current state:
   ```bash
   ./bin/portfolio-analyzer -latest
   ```

### Rebalancing Analysis

1. Check current allocation:
   ```bash
   ./bin/portfolio-analyzer -latest
   ```
2. Calculate rebalancing to target ratio:
   ```bash
   ./bin/portfolio-analyzer -latest -rebalance 5.0
   ```
3. Execute trades in Fidelity
4. Re-import updated portfolio

### Historical Performance Review

1. View historical summary:
   ```bash
   ./bin/portfolio-analyzer -historical
   ```
2. View detailed performance metrics:
   ```bash
   ./bin/portfolio-analyzer -performance
   ```

## Integration with mNAV Analysis

The portfolio tracking system integrates with the main mNAV analysis:

- **Bitcoin Price Data**: Uses same Bitcoin historical prices
- **MSTR Analysis**: Correlates with MSTR mNAV calculations
- **Market Context**: Portfolio performance vs Bitcoin/MSTR trends

### Combined Analysis Workflow

```bash
# Update all data
make workflow-mstr

# Import current portfolio
./bin/portfolio-importer -csv latest_portfolio.csv

# Analyze portfolio in context of MSTR mNAV
./bin/portfolio-analyzer -latest -rebalance 5.0
./bin/mnav-historical -symbol=MSTR -days=30
```

## Example Output

### Portfolio Summary
```
📊 Portfolio Analysis - June 11, 2025
============================================================
💰 Total Portfolio Value: $108,595.92
📈 Total Gain/Loss: $47,506.92 (77.77%)
₿  Bitcoin Exposure: $99,395.35 (91.5%)
⚖️  FBTC/MSTR Ratio: 6.01:1

🏦 Account Breakdown:
   ROTH IRA                  $72,301.05 (66.6%)
   Traditional IRA           $32,090.11 (29.6%)  
   ROTH IRA for Minor        $4,204.76 (3.9%)

💎 Asset Allocation:
   FBTC (Bitcoin ETF)   $85,218.16 (78.5%)
   MSTR (MicroStrategy) $14,177.19 (13.1%)
   GLD (Gold ETF)       $3,682.20 (3.4%)
   Other Assets         $5,518.37 (5.1%)
```

### Rebalancing Recommendation
```
🔄 Rebalancing Analysis (Target FBTC:MSTR = 5.0:1)
============================================================
Current FBTC:MSTR Ratio: 6.01:1
Target FBTC:MSTR Ratio:  5.00:1

💱 Recommended Trades:
   SELL 24.98 shares of FBTC (~$2,388.70)
   BUY 6.16 shares of MSTR (~$2,388.70)

🎯 After Rebalancing:
   FBTC: $82,829.46 (76.3%)
   MSTR: $16,565.89 (15.3%)
   New Ratio: 5.00:1
```

## Data Privacy & Security

- All data stored locally
- No external data transmission
- Raw CSV files archived for audit trail
- JSON format for processed data enables easy backup/restore

## Troubleshooting

### CSV Import Issues

1. **"wrong number of fields"** - CSV may have formatting issues
   - Check for extra commas or line breaks in description fields
   - Ensure proper CSV export from Fidelity

2. **"No portfolio data found"** - No data imported yet
   - Import at least one CSV file first

3. **"Failed to load portfolio for date"** - Date not found
   - Check available dates with: `./bin/portfolio-analyzer`

### Analysis Issues

1. **"No historical data available"** - Need multiple snapshots
   - Import portfolios from different dates to enable historical analysis

2. **Rebalancing shows 0 trades** - Already at target ratio
   - Current ratio matches target ratio

## Architecture

The portfolio tracking system follows the project's clean architecture:

```
cmd/portfolio/           # CLI applications
├── importer/           # CSV import tool
└── analyzer/           # Analysis tool

pkg/portfolio/          # Core packages
├── models/             # Data structures
├── analyzer/           # Business logic
└── tracker/            # Historical data management
```

This modular design enables:
- Easy extension for new analysis types
- Integration with existing mNAV tools
- Clean separation of concerns
- Comprehensive testing capabilities 