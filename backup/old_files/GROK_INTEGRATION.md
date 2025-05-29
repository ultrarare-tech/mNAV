# Grok AI Integration for Bitcoin Transaction Extraction

## Overview

The mNAV project now includes **Grok AI integration** to enhance Bitcoin transaction extraction from SEC filings. This hybrid approach combines fast regex patterns with AI-powered analysis for maximum accuracy and coverage.

## Architecture

### Hybrid Parser System
- **Primary**: Fast regex patterns for known transaction formats
- **Fallback**: Grok AI analysis when regex finds no transactions
- **Cost Optimization**: Grok only activates when needed
- **Graceful Degradation**: Works without API key (regex-only mode)

### Key Components
- `pkg/edgar/grok_parser.go` - Grok API client and enhanced parser
- `pkg/edgar/parser.go` - Updated with hybrid parsing support
- `cmd/edgar-enhanced/main.go` - Production tool with `-grok` flag
- `cmd/validate-grok/main.go` - Validation and testing tool

## Setup

### 1. Get Grok API Key
1. Sign up at [x.ai](https://x.ai) for Grok API access
2. Generate an API key from your dashboard

### 2. Configure Environment
```bash
# Add to your .env file or export directly
export GROK_API_KEY="your-api-key-here"
```

### 3. Build Tools
```bash
make build
# Or build specific tools:
make edgar-enhanced
make validate-grok
```

## Usage

### Production Integration

#### Enhanced EDGAR Tool with Grok
```bash
# Standard processing (regex only)
./bin/edgar-enhanced -ticker=MSTR -start=2020-01-01

# AI-enhanced processing with Grok fallback
./bin/edgar-enhanced -ticker=MSTR -start=2020-01-01 -grok

# Incremental update with Grok
./bin/edgar-enhanced -ticker=MSTR -incremental -grok -verbose
```

#### Key Features
- **Smart Activation**: Grok only runs when regex finds no transactions
- **Cost Control**: Minimizes API usage while maximizing coverage
- **Seamless Integration**: Drop-in replacement for existing workflows
- **Progress Indicators**: Shows when Grok is being used

### Validation and Testing

#### Validate Grok Performance
```bash
# Test on local raw filings (no re-downloading)
./bin/validate-grok -ticker=MSTR -max=20 -verbose

# Compare regex vs Grok on specific date range
./bin/validate-grok -ticker=MSTR -max=10 -output=validation.json
```

#### Grok-Only Testing
```bash
# Test Grok extraction on specific filings
./bin/grok-test -ticker=MSTR -method=grok -verbose
```

## Performance Characteristics

### Speed Comparison
- **Regex**: 0.5-2ms per filing
- **Grok**: ~2000ms per filing (when activated)
- **Hybrid**: Regex speed + Grok only when needed

### Cost Optimization
- Grok API calls only when regex finds 0 transactions
- Typical usage: 10-20% of filings require Grok analysis
- Estimated cost: $0.01-0.05 per filing (when Grok is used)

### Accuracy Results
Based on MSTR validation (2020-2021 Bitcoin era):
- **Agreement Rate**: 100% on key Bitcoin transactions
- **False Positives**: Both methods correctly filter financing activities
- **Coverage**: Grok can find transactions missed by regex patterns

## Configuration

### Environment Variables
```bash
# Required for Grok functionality
GROK_API_KEY=your-api-key-here

# Optional: Grok API endpoint (defaults to x.ai)
GROK_API_URL=https://api.x.ai/v1
```

### Command Line Options
```bash
# edgar-enhanced tool
-grok                    # Enable Grok AI enhancement
-verbose                 # Show Grok activation messages

# validate-grok tool
-ticker=SYMBOL          # Company to validate
-max=N                  # Max filings to test
-verbose                # Show detailed comparisons
-output=file.json       # Save results to file
```

## API Details

### Grok Model
- **Model**: `grok-2-1212` (latest stable version)
- **Endpoint**: `https://api.x.ai/v1/chat/completions`
- **Timeout**: 60 seconds per request
- **Format**: OpenAI-compatible chat completions

### Prompt Engineering
The system uses carefully crafted prompts that:
- Distinguish actual transactions from financing activities
- Exclude future intentions ("intends to invest")
- Extract confidence scores and reasoning
- Provide structured JSON output

### Response Format
```json
{
  "transactions": [
    {
      "btc_amount": 21454.0,
      "usd_amount": 1150000000.0,
      "price_per_btc": 53617.0,
      "transaction_type": "purchase",
      "date": "2021-01-22",
      "confidence": 0.95,
      "reasoning": "Clear statement of Bitcoin purchase",
      "source_text": "purchased approximately 21,454 bitcoins for $1.15 billion"
    }
  ],
  "analysis": "Filing describes completed Bitcoin purchase transaction",
  "confidence": 0.95
}
```

## Error Handling

### Graceful Degradation
- **No API Key**: Falls back to regex-only mode with warning
- **API Errors**: Logs error, continues with regex results
- **Rate Limits**: Respects API limits, retries with backoff
- **Timeouts**: 60-second timeout prevents hanging

### Monitoring
- Processing time tracking for both methods
- Success/failure rates logged
- API usage statistics in validation reports

## Best Practices

### When to Use Grok
✅ **Recommended for:**
- Historical data analysis (one-time processing)
- Companies with complex filing language
- Validation and accuracy verification
- Research and development

⚠️ **Consider carefully for:**
- Real-time processing (due to latency)
- High-frequency updates (due to cost)
- Large-scale batch processing (monitor costs)

### Cost Management
1. **Start with validation**: Test on small datasets first
2. **Use incremental mode**: Only process new filings
3. **Monitor usage**: Check validation reports for API call frequency
4. **Set budgets**: Consider API rate limits and costs

### Quality Assurance
1. **Validate results**: Use `validate-grok` tool regularly
2. **Review differences**: Check cases where Grok disagrees with regex
3. **Update patterns**: Improve regex based on Grok findings
4. **Monitor confidence**: Low confidence scores may need review

## Troubleshooting

### Common Issues

#### "GROK_API_KEY environment variable must be set"
```bash
# Solution: Set your API key
export GROK_API_KEY="your-key-here"
```

#### "API request failed with status 401"
- Check API key validity
- Verify account has Grok access
- Ensure key has proper permissions

#### "API request failed with status 429"
- Rate limit exceeded
- Wait and retry
- Consider reducing batch size

#### Slow performance
- Normal: Grok adds ~2 seconds per filing when activated
- Check: Ensure Grok only activates when regex finds no transactions
- Monitor: Use `-verbose` flag to see when Grok is used

### Debug Mode
```bash
# Enable verbose logging
./bin/edgar-enhanced -ticker=MSTR -grok -verbose

# Validate specific filings
./bin/validate-grok -ticker=MSTR -max=5 -verbose
```

## Future Enhancements

### Planned Features
- **Batch Processing**: Process multiple filings in single API call
- **Caching**: Cache Grok results to avoid re-processing
- **Model Selection**: Support for different Grok models
- **Custom Prompts**: Configurable extraction prompts
- **Confidence Thresholds**: Configurable minimum confidence levels

### Integration Opportunities
- **Real-time Alerts**: Grok-powered filing analysis
- **Trend Analysis**: AI-powered pattern recognition
- **Compliance Monitoring**: Automated regulatory analysis
- **Portfolio Insights**: Cross-company transaction analysis

## Support

For issues or questions:
1. Check this documentation
2. Review validation results with `validate-grok`
3. Test with `grok-test` tool
4. Check API status at x.ai
5. Review logs with `-verbose` flag

---

**Note**: Grok integration is designed to enhance, not replace, existing regex patterns. The hybrid approach ensures reliability while providing AI-powered insights when needed. 