# API Setup Guide

This guide explains how to obtain and configure the API keys required for the mNAV application.

## Required API Keys

### 1. Financial Modeling Prep (FMP)
**Purpose**: Stock prices, market cap, and company profile data

**Sign up**: https://site.financialmodelingprep.com/
- Free tier: 250 API calls per day
- Paid plans available for higher limits

**Features used**:
- Historical stock prices
- Current stock quotes
- Company profiles with market cap

### 2. Alpha Vantage
**Purpose**: Shares outstanding and company fundamentals

**Sign up**: https://www.alphavantage.co/
- Free tier: 5 calls per minute, 500 per day
- Premium plans available

**Features used**:
- Company overview data
- Shares outstanding information
- Financial metrics

### 3. CoinMarketCap (New!)
**Purpose**: Historical Bitcoin price data

**Sign up**: https://pro.coinmarketcap.com/
- Free tier: 10,000 call credits per month
- Full historical data access (back to 2013)
- Professional grade data quality

**Features used**:
- Historical Bitcoin prices (complete dataset)
- Daily OHLCV data
- Volume and market cap data

### 4. Grok AI (Optional)
**Purpose**: Bitcoin transaction extraction from SEC filings

**Sign up**: https://x.ai/
- Required for automated Bitcoin transaction parsing
- Manual parsing available as fallback

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# Required for stock data collection
FMP_API_KEY=your_financial_modeling_prep_api_key
ALPHA_VANTAGE_API_KEY=your_alpha_vantage_api_key

# Required for Bitcoin historical data
COINMARKETCAP_API_KEY=your_coinmarketcap_api_key

# Optional for automated Bitcoin parsing
GROK_API_KEY=your_grok_api_key
```

### Command Line Flags

Alternatively, pass API keys as command line flags:

```bash
# Stock data collection
./bin/stock-data -fmp-api-key=YOUR_FMP_KEY -av-api-key=YOUR_AV_KEY

# Historical mNAV calculation
./bin/mnav-historical -fmp-api-key=YOUR_FMP_KEY -av-api-key=YOUR_AV_KEY
```

Note: CoinMarketCap API key is read from environment variable only.

## Testing Your Setup

### Verify FMP API Key
```bash
curl "https://financialmodelingprep.com/api/v3/profile/MSTR?apikey=YOUR_FMP_KEY"
```

### Verify Alpha Vantage API Key
```bash
curl "https://www.alphavantage.co/query?function=OVERVIEW&symbol=MSTR&apikey=YOUR_AV_KEY"
```

### Verify CoinMarketCap API Key
```bash
curl -H "X-CMC_PRO_API_KEY: YOUR_CMC_KEY" \
     "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest?id=1&convert=USD"
```

### Test with mNAV Application
```bash
# Test Bitcoin price collection (full historical data)
./bin/bitcoin-historical -start=2020-08-11

# Test stock data collection
./bin/stock-data -symbol=MSTR -start=2020-08-11

# Should output data and save files to respective directories
```

## Rate Limits and Best Practices

### Financial Modeling Prep
- **Free tier**: 250 calls/day, ~5 calls/minute
- **Best practice**: Cache responses, avoid repeated calls for same data
- **Upgrade**: Consider paid plan for production use

### Alpha Vantage
- **Free tier**: 5 calls/minute, 500/day
- **Best practice**: Use company overview sparingly (data doesn't change often)
- **Upgrade**: Premium plans offer higher limits

### CoinMarketCap
- **Free tier**: 10,000 call credits/month (generous)
- **Rate limit**: 30 calls/minute
- **Best practice**: Historical data doesn't change, cache locally
- **Advantage**: Full historical access on free tier

## Troubleshooting

### Common Issues

**"API key is required" error**:
- Verify `.env` file exists and contains correct keys
- Check environment variable names match exactly
- Ensure no extra spaces or quotes in `.env` file

**"Rate limit exceeded" error**:
- Wait before retrying (respect rate limits)
- Consider upgrading to paid API plan
- Use cached data when available

**"Invalid API key" error**:
- Verify API key is correct and active
- Check if API key has required permissions
- Regenerate API key if necessary

**CoinMarketCap "Credit limit exceeded"**:
- Check usage in CoinMarketCap dashboard
- Free tier provides 10,000 credits/month
- Historical data requests use 1 credit per data point

### Getting Help

- **FMP Support**: https://site.financialmodelingprep.com/contact
- **Alpha Vantage Support**: https://www.alphavantage.co/support/
- **CoinMarketCap Support**: https://coinmarketcap.com/api/documentation/
- **mNAV Issues**: Create an issue in the project repository

## Cost Considerations

### Free Tier Limitations
- **FMP**: 250 calls/day (sufficient for daily analysis)
- **Alpha Vantage**: 500 calls/day (sufficient for occasional company data updates)
- **CoinMarketCap**: 10,000 credits/month (sufficient for complete historical analysis)
- **Total cost**: $0/month for complete mNAV analysis

### Paid Plans
- **FMP**: Starting at $15/month for 1,000 calls/day
- **Alpha Vantage**: Starting at $25/month for higher limits
- **CoinMarketCap**: Starting at $79/month for higher limits
- **Recommended**: Start with free tiers, upgrade as needed

### Usage Optimization
- Cache historical data locally (it doesn't change)
- Avoid redundant API calls
- Use batch operations when available
- Monitor usage with API dashboards

## Migration from CoinGecko

If you were previously using CoinGecko:

1. **Get CoinMarketCap API key** (free)
2. **Add to .env file**: `COINMARKETCAP_API_KEY=your_key`
3. **Rebuild tools**: `make bitcoin-historical`
4. **Test with full range**: `./bin/bitcoin-historical -start=2020-08-11`

**Benefits of migration**:
- ✅ Full historical data (vs. 365-day limit)
- ✅ Professional grade data quality
- ✅ Same free tier cost ($0)
- ✅ Better API reliability 