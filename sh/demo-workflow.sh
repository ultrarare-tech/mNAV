#!/bin/bash

# mNAV Demo Workflow Script
# This script demonstrates the mNAV analysis workflow with CoinGecko integration

echo "🚀 mNAV Demo Workflow"
echo "====================="
echo ""

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ Missing .env file with API keys"
    echo ""
    echo "📋 Required API keys:"
    echo "   • FMP_API_KEY (Financial Modeling Prep)"
    echo "   • ALPHA_VANTAGE_API_KEY (Alpha Vantage)"
    echo "   • Bitcoin data is FREE via CoinGecko - no API key needed!"
    echo ""
    echo "📝 Steps to get API keys:"
    echo "   1. Copy env.example to .env: cp env.example .env"
    echo "   2. Get FMP API key: https://site.financialmodelingprep.com/"
    echo "   3. Get Alpha Vantage key: https://www.alphavantage.co/"
    echo "   4. CoinGecko is free - no API key needed for Bitcoin historical data!"
    echo "   5. Edit .env file with your actual API keys"
    echo ""
    exit 1
fi

# Source environment variables
source .env

# Check required API keys
if [ -z "$FMP_API_KEY" ]; then
    echo "❌ FMP_API_KEY not set in .env file"
    exit 1
fi

if [ -z "$ALPHA_VANTAGE_API_KEY" ]; then
    echo "❌ ALPHA_VANTAGE_API_KEY not set in .env file"
    exit 1
fi

# CoinGecko doesn't require an API key - it's free!
echo "✅ API keys configured"
echo "💡 Using CoinGecko API for Bitcoin data - free, no API key required!"
echo ""

# Build tools if needed
echo "🔨 Building mNAV tools..."
make all
echo ""

# Step 1: Collect Bitcoin prices using CoinGecko (free, comprehensive historical data)
echo "📊 Step 1: Collecting Bitcoin prices..."
echo "   🔗 Using CoinGecko API for reliable historical data back to 2020"
echo "   💰 Free API - no rate limits or subscription required!"
./bin/bitcoin-historical -start=2020-08-11
echo ""

# Step 2: Collect MSTR stock data (using free Yahoo Finance)
echo "📈 Step 2: Collecting MSTR stock data from Yahoo Finance..."
./bin/update-stock-data -symbol=MSTR -verbose
echo ""

# Step 3: Calculate mNAV (requires existing Bitcoin transaction data)
echo "📊 Step 3: Calculating historical mNAV..."
echo "   Using existing MSTR Bitcoin transaction data from SEC filings"
./bin/mnav-historical -symbol=MSTR -start=2020-08-11
echo ""

# Step 4: Generate chart
echo "📈 Step 4: Generating mNAV chart..."
./bin/mnav-chart -format=html
echo ""

echo "✅ Demo workflow complete!"
echo ""
echo "📂 Check these directories for results:"
echo "   • data/bitcoin-prices/historical/ - Bitcoin price data"
echo "   • data/stock-data/ - Stock price and company data"
echo "   • data/analysis/mnav/ - mNAV calculations"
echo "   • data/charts/ - Interactive HTML charts"
echo ""
echo "🌐 Open the HTML chart in your browser to see the results!"

# Check if chart was created
if [ -f "data/charts/mstr_mnav_chart.html" ]; then
    echo "   📊 Chart location: data/charts/mstr_mnav_chart.html"
fi

echo ""
echo "🔍 New improvements:"
echo "   ✅ CoinGecko integration: Reliable Bitcoin Price Index data"
echo "   ✅ Free Bitcoin data: No API key or subscription required"
echo "   ✅ Professional APIs: FMP + Alpha Vantage + CoinGecko"
echo "   ✅ Complete mNAV analysis: Full historical perspective" 