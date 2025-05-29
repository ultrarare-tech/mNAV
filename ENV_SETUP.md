# Environment Setup Instructions

To run this application, you need to set up your environment variables. Follow these steps:

## 1. Create a .env file

Create a file named `.env` in the root directory of the project with the following content:

```
# API Keys for mNAV Application

# CoinMarketCap API (required for Bitcoin price)
COINMARKETCAP_API_KEY=your_coinmarketcap_api_key_here

# Yahoo Finance API (public API currently used, but adding for future extensibility)
YAHOO_FINANCE_API_KEY=
```

## 2. Get a CoinMarketCap API Key

1. Go to [CoinMarketCap Developer Portal](https://coinmarketcap.com/api/)
2. Sign up for a free account
3. Navigate to the API Keys section in your account
4. Create a new API key
5. Copy the API key and replace `your_coinmarketcap_api_key_here` in your `.env` file

## 3. Yahoo Finance API

The current implementation uses the public Yahoo Finance API which doesn't require authentication. The `YAHOO_FINANCE_API_KEY` field is included for future extensibility if Yahoo changes their API requirements.

## 4. Environment Variables Usage

These environment variables are loaded when the application starts. If the `.env` file is not found, the application will check for environment variables set in your system.

You can also set these environment variables directly in your system instead of using a `.env` file:

### macOS/Linux:
```bash
export COINMARKETCAP_API_KEY=your_key_here
```

### Windows Command Prompt:
```cmd
set COINMARKETCAP_API_KEY=your_key_here
```

### Windows PowerShell:
```powershell
$env:COINMARKETCAP_API_KEY = "your_key_here"
``` 