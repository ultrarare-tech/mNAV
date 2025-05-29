#!/bin/bash

# Load environment variables and export them
source .env
export GROK_API_KEY GROK_MODEL

echo "🚀 Complete Fresh Start: MSTR Bitcoin Transactions + mNAV Calculations"
echo "====================================================================="
echo ""
echo "🎯 Comprehensive Objective:"
echo "  1. 🪙 Extract complete Bitcoin transaction history"
echo "  2. 📊 Calculate shares outstanding over time"
echo "  3. 💰 Compute mNAV (Bitcoin value per share)"
echo "  4. 📈 Generate complete treasury tracking data"
echo ""
echo "📅 Date Range: 2020-01-01 to 2025-05-28 (today)"
echo "🤖 AI Enhancement: Grok AI enabled for maximum accuracy"
echo "📊 Progress: Enhanced real-time reporting"
echo ""
echo "🔧 Configuration:"
echo "  • API Key: ${#GROK_API_KEY} characters configured"
echo "  • Model: $GROK_MODEL"
echo "  • Filing Types: 8-K, 10-Q, 10-K"
echo "  • Verbose Mode: Enabled"
echo ""

# Confirm before starting
read -p "🤔 Ready to start complete extraction + mNAV calculations? This may take 15-25 minutes. (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Process cancelled."
    exit 1
fi

echo ""
echo "🎬 Phase 1: Bitcoin Transaction Extraction"
echo "=========================================="
echo "⏰ Started at: $(date)"
echo ""

# Phase 1: Extract Bitcoin transactions with enhanced approach
echo "🔍 Extracting Bitcoin transactions with Grok AI enhancement..."
./bin/edgar-enhanced -ticker=MSTR -start=2020-01-01 -end=2025-05-28 -grok -verbose

echo ""
echo "✅ Phase 1 Complete: Bitcoin Transaction Extraction"
echo ""

# Check if we have Bitcoin transaction data
if [ ! -f "data/edgar/companies/MSTR/btc_transactions.json" ]; then
    echo "❌ Error: No Bitcoin transactions file found. Cannot proceed with mNAV calculations."
    exit 1
fi

echo "🎬 Phase 2: mNAV Calculations"
echo "============================="
echo ""

# Phase 2: Calculate mNAV values
echo "💰 Calculating mNAV (Bitcoin value per share)..."

# Check if we have the mNAV calculation tool
if [ ! -f "bin/mnav" ]; then
    echo "🔧 Building mNAV calculation tool..."
    make mnav
fi

# Run mNAV calculations
echo "📊 Running mNAV calculations for MSTR..."
./bin/mnav -ticker=MSTR -verbose

echo ""
echo "✅ Phase 2 Complete: mNAV Calculations"
echo ""

echo "🎉 Complete Fresh Start Finished!"
echo "⏰ Finished at: $(date)"
echo ""

# Show comprehensive results summary
echo "📊 Comprehensive Results Summary:"
echo "================================"

# Bitcoin Transaction Summary
if [ -f "data/edgar/companies/MSTR/btc_transactions.json" ]; then
    echo ""
    echo "🪙 Bitcoin Transaction Data:"
    echo "  ✅ Transactions file created"
    
    # Count total transactions
    TOTAL_TRANSACTIONS=$(cat data/edgar/companies/MSTR/btc_transactions.json | jq 'length' 2>/dev/null || echo "0")
    echo "  📈 Total transactions found: $TOTAL_TRANSACTIONS"
    
    # Calculate total BTC holdings
    TOTAL_BTC=$(cat data/edgar/companies/MSTR/btc_transactions.json | jq '[.[] | select(.btcPurchased > 0)] | map(.btcPurchased) | add' 2>/dev/null || echo "0")
    echo "  🪙 Total BTC purchased: $TOTAL_BTC"
    
    # Calculate total USD spent
    TOTAL_USD=$(cat data/edgar/companies/MSTR/btc_transactions.json | jq '[.[] | select(.usdSpent > 0)] | map(.usdSpent) | add' 2>/dev/null || echo "0")
    echo "  💵 Total USD spent: \$$(printf "%.0f" $TOTAL_USD)"
    
    # Show latest transaction
    echo ""
    echo "📋 Latest Bitcoin Transaction:"
    cat data/edgar/companies/MSTR/btc_transactions.json | jq '.[-1] | {date, btcPurchased, usdSpent, totalBtcAfter}' 2>/dev/null || echo "  No transactions found"
    
else
    echo "❌ No Bitcoin transactions file found"
fi

# Shares Outstanding Summary
if [ -f "data/edgar/companies/MSTR/financial_data.json" ]; then
    echo ""
    echo "📈 Shares Outstanding Data:"
    echo "  ✅ Financial data file created"
    
    # Show latest shares data
    echo ""
    echo "📋 Latest Shares Outstanding:"
    cat data/edgar/companies/MSTR/financial_data.json | jq '.sharesHistory[-1] | {date, totalShares, confidenceScore}' 2>/dev/null || echo "  No shares data found"
else
    echo "❌ No financial data file found"
fi

# mNAV Summary
if [ -f "data/edgar/companies/MSTR/latest_snapshot.json" ]; then
    echo ""
    echo "💰 mNAV Data:"
    echo "  ✅ Latest snapshot created"
    
    # Show current mNAV
    echo ""
    echo "📋 Current mNAV Snapshot:"
    cat data/edgar/companies/MSTR/latest_snapshot.json | jq '.' 2>/dev/null || echo "  No snapshot data found"
else
    echo "❌ No mNAV snapshot found"
fi

echo ""
echo "📁 Data Locations:"
echo "=================="
echo "  🪙 Bitcoin Transactions: data/edgar/companies/MSTR/btc_transactions.json"
echo "  📈 Financial Data: data/edgar/companies/MSTR/financial_data.json"
echo "  💰 Latest Snapshot: data/edgar/companies/MSTR/latest_snapshot.json"
echo "  🗂️  Raw Filings: data/edgar/companies/MSTR/raw_filings/"
echo "  📊 Processing Results: data/edgar/companies/MSTR/processing_results/"
echo ""

echo "🎯 Next Steps & Analysis Commands:"
echo "================================="
echo "  📊 View all transactions: cat data/edgar/companies/MSTR/btc_transactions.json | jq ."
echo "  💰 Check current mNAV: cat data/edgar/companies/MSTR/latest_snapshot.json | jq ."
echo "  📈 View shares history: cat data/edgar/companies/MSTR/financial_data.json | jq '.sharesHistory'"
echo "  🔍 Validate extraction: ./bin/validate-grok -ticker=MSTR -max=10"
echo "  📊 Recalculate mNAV: ./bin/mnav -ticker=MSTR -verbose"
echo ""
echo "✅ Complete fresh start finished with comprehensive Bitcoin + mNAV tracking!"
echo "🎉 MSTR treasury data is now ready for analysis!" 