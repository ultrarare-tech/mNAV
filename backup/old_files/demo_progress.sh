#!/bin/bash

# Load environment variables and export them
source .env
export GROK_API_KEY GROK_MODEL

echo "🎯 Enhanced Progress Reporting Demo"
echo "=================================="
echo ""
echo "This demo shows the new progress features:"
echo "  📊 Real-time progress bar with percentage"
echo "  ⏱️  Elapsed time and ETA calculations"
echo "  🔍 Step-by-step processing details"
echo "  🪙 Bitcoin transaction details (in verbose mode)"
echo "  📈 Running totals and statistics"
echo "  ⏳ Rate limiting countdown"
echo ""
echo "Testing with MSTR filings from January 2021 (Bitcoin purchase period)..."
echo ""

# Run with enhanced progress reporting
./bin/edgar-enhanced -ticker=MSTR -start=2021-01-01 -end=2021-01-31 -grok -verbose

echo ""
echo "✅ Demo complete!"
echo ""
echo "Key features demonstrated:"
echo "  • Progress bar: [████████████████████] 100%"
echo "  • Time tracking: Elapsed and ETA"
echo "  • Step details: Fetching → Parsing → Saving"
echo "  • Transaction details: BTC amounts and USD values"
echo "  • Running totals: Cumulative counts"
echo "  • Rate limiting: Visual countdown" 