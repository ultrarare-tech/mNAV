# mNAV Charting System

This document explains how to generate historical mNAV charts for MSTR using the new API integrations with Financial Modeling Prep and Alpha Vantage.

## Overview

The mNAV charting system consists of three main components:

1. **Data Collection**: Gathering historical Bitcoin prices, stock prices, and company data
2. **mNAV Calculation**: Computing historical mNAV values and premiums over time  
3. **Chart Generation**: Creating interactive HTML charts or data exports

## Prerequisites

### API Keys Required

You'll need API keys from two providers:

1. **Financial Modeling Prep** (for stock prices and market data)
   - Sign up at: https://site.financialmodelingprep.com/
   - Used for historical stock prices and current market data
   
2. **Alpha Vantage** (for shares outstanding data)
   - Sign up at: https://www.alphavantage.co/
   - Used for company overview including shares outstanding

### Environment Setup

Set your API keys as environment variables:

```bash
export FMP_API_KEY="your_financial_modeling_prep_api_key"
export ALPHA_VANTAGE_API_KEY="your_alpha_vantage_api_key"
```

Or pass them as command line flags (see examples below).

## Step-by-Step Process

### Step 1: Build the Tools

```bash
# Build all mNAV charting tools
make bitcoin-historical mnav-historical mnav-chart stock-data

# Or build everything
make all
```

### Step 2: Collect Historical Bitcoin Prices

```bash
# Collect Bitcoin price history from August 2020 to present
./bin/bitcoin-historical \
  -start=2020-08-11 \
  -output=data/bitcoin-prices/historical
```

This fetches Bitcoin prices from CoinGecko (free, no API key required).

### Step 3: Collect Stock Data

```bash
# Collect MSTR stock data using both APIs
./bin/stock-data \
  -symbol=MSTR \
  -start=2020-08-11 \
  -fmp-api-key=$FMP_API_KEY \
  -av-api-key=$ALPHA_VANTAGE_API_KEY \
  -output=data/stock-data
```

This collects:
- Historical stock prices from Financial Modeling Prep
- Current stock price and company profile from FMP
- Company overview including shares outstanding from Alpha Vantage

### Step 4: Calculate Historical mNAV

```bash
# Calculate mNAV time series
./bin/mnav-historical \
  -symbol=MSTR \
  -start=2020-08-11 \
  -interval=daily \
  -fmp-api-key=$FMP_API_KEY \
  -av-api-key=$ALPHA_VANTAGE_API_KEY \
  -output=data/analysis/mnav
```

This calculates:
- Bitcoin holdings at each date (from SEC filings)
- mNAV ratio (Bitcoin value / Stock market cap)
- Premium/discount percentage
- Per-share metrics

### Step 5: Generate Charts

#### HTML Interactive Chart

```bash
# Generate HTML chart with Chart.js
./bin/mnav-chart \
  -input=data/analysis/mnav/MSTR_mnav_historical_2020-08-11_to_2024-12-19.json \
  -format=html \
  -output=data/charts
```

#### CSV Export

```bash
# Generate CSV for external tools
./bin/mnav-chart \
  -input=data/analysis/mnav/MSTR_mnav_historical_2020-08-11_to_2024-12-19.json \
  -format=csv \
  -output=data/charts
```

#### JSON Chart Data

```bash
# Generate JSON for custom charting
./bin/mnav-chart \
  -input=data/analysis/mnav/MSTR_mnav_historical_2020-08-11_to_2024-12-19.json \
  -format=json \
  -output=data/charts
```

## Output Files

### Historical mNAV Data

The mNAV calculator produces JSON files with this structure:

```json
{
  "symbol": "MSTR",
  "start_date": "2020-08-11",
  "end_date": "2024-12-19",
  "data_points": [
    {
      "date": "2020-08-11",
      "stock_price": 135.50,
      "bitcoin_price": 11449.78,
      "bitcoin_holdings": 21454.0,
      "shares_outstanding": 100000000,
      "market_cap": 13550000000,
      "bitcoin_value": 245500000,
      "mnav": 0.018,
      "mnav_per_share": 2.45,
      "premium_percentage": 5428.57
    }
  ],
  "metadata": {
    "interval": "daily",
    "source": "SEC filings + FMP + Alpha Vantage",
    "current_shares_outstanding": 100000000,
    "bitcoin_transactions_count": 42
  }
}
```

### HTML Charts

Interactive charts showing:
- mNAV ratio over time (left axis)
- Premium/discount percentage (right axis)
- Hover tooltips with detailed data
- Responsive design for mobile/desktop

## Advanced Usage

### Custom Date Ranges

```bash
# Calculate mNAV for specific period
./bin/mnav-historical \
  -symbol=MSTR \
  -start=2023-01-01 \
  -end=2023-12-31 \
  -interval=weekly
```

### Different Intervals

- `daily`: Daily calculations (default)
- `weekly`: Weekly calculations (every 7 days)
- `monthly`: Monthly calculations

### Multiple Symbols

Currently optimized for MSTR, but can be extended:

```bash
# For other Bitcoin companies (requires their SEC filing data)
./bin/mnav-historical -symbol=COIN -start=2021-04-14
```

## Data Sources and Attribution

- **Bitcoin Prices**: CoinGecko API (free)
- **Stock Prices**: Financial Modeling Prep API
- **Market Cap**: Financial Modeling Prep API  
- **Shares Outstanding**: Alpha Vantage API
- **Bitcoin Holdings**: SEC filing analysis (your mNAV application)

## API Limits and Considerations

### Financial Modeling Prep
- Free tier: 250 API calls per day
- Paid tiers available for higher limits
- Rate limit: ~5 requests per minute

### Alpha Vantage  
- Free tier: 5 API calls per minute, 500 per day
- Premium tiers available
- Company overview data is updated quarterly

### CoinGecko
- Free tier: 10-50 calls per minute (depending on endpoint)
- No API key required for basic endpoints

## Troubleshooting

### Common Issues

1. **API Key Missing**
   ```
   ❌ Financial Modeling Prep API key is required
   ```
   Solution: Set environment variables or use flags

2. **No Historical Bitcoin Prices**
   ```
   ❌ no historical Bitcoin price files found
   ```
   Solution: Run `bitcoin-historical` command first

3. **No Bitcoin Transaction Data**
   ```
   ❌ Error loading Bitcoin transactions
   ```
   Solution: Ensure SEC filing analysis has been run for the symbol

4. **Rate Limiting**
   ```
   API error (status 429): Too Many Requests
   ```
   Solution: Wait and retry, or upgrade API plan

### Data Validation

Compare results with external sources:
- Bitcoin holdings vs. SaylorTracker.com
- Stock prices vs. Yahoo Finance/Google Finance
- Market cap calculations for consistency

## Future Enhancements

Planned improvements:
- Historical shares outstanding tracking
- Multiple cryptocurrency support
- Real-time data updates
- Dashboard interface
- Automated report generation
- Integration with portfolio management tools

## Example Workflow

Complete example for generating MSTR mNAV charts:

```bash
# 1. Set environment variables
export FMP_API_KEY="your_fmp_key"
export ALPHA_VANTAGE_API_KEY="your_av_key"

# 2. Build tools
make bitcoin-historical mnav-historical mnav-chart stock-data

# 3. Collect all data
./bin/bitcoin-historical -start=2020-08-11
./bin/stock-data -symbol=MSTR -start=2020-08-11

# 4. Calculate mNAV
./bin/mnav-historical -symbol=MSTR -start=2020-08-11

# 5. Generate chart
./bin/mnav-chart -format=html

# 6. Open chart in browser
open data/charts/MSTR_mnav_chart_2024-12-19.html
```

This complete workflow will produce an interactive HTML chart showing MSTR's mNAV evolution over time. 