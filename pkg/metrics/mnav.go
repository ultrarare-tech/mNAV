package metrics

import (
	"fmt"
	"math"
)

// Company represents a Bitcoin-treasury company
type Company struct {
	Symbol            string  // Stock symbol
	Name              string  // Company name
	StockPrice        float64 // Current stock price
	MarketCap         float64 // Market capitalization
	BTCHoldings       float64 // Bitcoin holdings (in BTC)
	BTCYield          float64 // Daily BTC yield (as a decimal, e.g., 0.0012 for 0.12%)
	OutstandingShares float64 // Outstanding shares
}

// BitcoinMetrics contains metrics related to Bitcoin treasury companies
type BitcoinMetrics struct {
	Company          Company             // Company information
	BTCPrice         float64             // Current Bitcoin price
	MNAV             float64             // Multiple of Net Asset Value
	DaysToCover      float64             // Days to Cover mNAV
	BTCValue         float64             // Total value of Bitcoin holdings
	BTCYieldDaily    float64             // How much BTC the company acquires daily
	MNAVPriceTargets map[float64]float64 // Map of mNAV values to corresponding stock prices
}

// CalculateMNAV calculates the Multiple of Net Asset Value
// mNAV = Market Capitalization / (Bitcoin Holdings × Bitcoin Price)
func CalculateMNAV(marketCap, btcHoldings, btcPrice float64) (float64, error) {
	if btcHoldings <= 0 {
		return 0, fmt.Errorf("Bitcoin holdings must be greater than zero")
	}
	if btcPrice <= 0 {
		return 0, fmt.Errorf("Bitcoin price must be greater than zero")
	}

	btcValue := btcHoldings * btcPrice
	return marketCap / btcValue, nil
}

// CalculateDaysToCover calculates the Days to Cover mNAV
// Days to Cover = ln(mNAV) / ln(1 + BTC Yield)
func CalculateDaysToCover(mnav, btcYield float64) (float64, error) {
	if mnav <= 0 {
		return 0, fmt.Errorf("mNAV must be greater than zero")
	}
	if btcYield <= 0 {
		return 0, fmt.Errorf("BTC Yield must be greater than zero")
	}

	// If mNAV < 1, the company already has more Bitcoin value than its market cap
	// In this case, return a negative value representing how many days ago this happened
	if mnav < 1 {
		// Use the inverse calculation to determine when mNAV crossed 1.0
		// This will give a negative value representing days in the past
		return math.Log(mnav) / math.Log(1+btcYield), nil
	}

	return math.Log(mnav) / math.Log(1+btcYield), nil
}

// CalculatePriceForMNAV calculates the stock price required to achieve a specific mNAV
// Stock Price = (mNAV × Bitcoin Value) / Outstanding Shares
func CalculatePriceForMNAV(targetMNAV, btcHoldings, btcPrice, outstandingShares float64) (float64, error) {
	if btcHoldings <= 0 {
		return 0, fmt.Errorf("Bitcoin holdings must be greater than zero")
	}
	if btcPrice <= 0 {
		return 0, fmt.Errorf("Bitcoin price must be greater than zero")
	}
	if outstandingShares <= 0 {
		return 0, fmt.Errorf("Outstanding shares must be greater than zero")
	}

	btcValue := btcHoldings * btcPrice
	targetMarketCap := targetMNAV * btcValue

	return targetMarketCap / outstandingShares, nil
}

// CalculateMNAVPriceTargets calculates stock prices for a range of mNAV values
func CalculateMNAVPriceTargets(btcHoldings, btcPrice, outstandingShares float64, minMNAV, maxMNAV, step float64) (map[float64]float64, error) {
	priceTargets := make(map[float64]float64)

	if btcHoldings <= 0 {
		return nil, fmt.Errorf("Bitcoin holdings must be greater than zero")
	}
	if btcPrice <= 0 {
		return nil, fmt.Errorf("Bitcoin price must be greater than zero")
	}

	// Calculate Bitcoin value
	btcValue := btcHoldings * btcPrice

	// For very small or missing outstanding shares, use a default to avoid division by zero
	if outstandingShares < 1000 {
		outstandingShares = 10000000 // Use 10 million as a reasonable default
	}

	for mnav := minMNAV; mnav <= maxMNAV; mnav += step {
		// Round to 2 decimal places to avoid floating point issues
		mnavRounded := math.Round(mnav*100) / 100

		// Calculate target market cap for this mNAV
		targetMarketCap := mnavRounded * btcValue

		// Calculate per-share price
		targetPrice := targetMarketCap / outstandingShares

		priceTargets[mnavRounded] = targetPrice
	}

	return priceTargets, nil
}

// CalculateMetrics calculates all metrics for a company
func CalculateMetrics(company Company, btcPrice float64) (*BitcoinMetrics, error) {
	// Calculate Bitcoin value
	btcValue := company.BTCHoldings * btcPrice

	// Handle case where market cap is zero but we have stock price and can estimate
	if company.MarketCap == 0 && company.StockPrice > 0 {
		// Try to calculate market cap from outstanding shares
		if company.OutstandingShares > 0 {
			company.MarketCap = company.StockPrice * company.OutstandingShares
		}
	}

	// Calculate mNAV
	mnav, err := CalculateMNAV(company.MarketCap, company.BTCHoldings, btcPrice)
	if err != nil {
		return nil, fmt.Errorf("error calculating mNAV: %w", err)
	}

	// Calculate Days to Cover mNAV
	daysToCover, err := CalculateDaysToCover(mnav, company.BTCYield)
	if err != nil {
		return nil, fmt.Errorf("error calculating Days to Cover: %w", err)
	}

	// Calculate daily BTC yield
	btcYieldDaily := company.BTCHoldings * company.BTCYield

	// Calculate price targets for different mNAV values
	// Get the estimated outstanding shares from market cap / current price
	outstandingShares := company.OutstandingShares
	if outstandingShares <= 0 && company.MarketCap > 0 && company.StockPrice > 0 {
		outstandingShares = company.MarketCap / company.StockPrice
	}

	// For price target calculation, if we still don't have shares, use a reasonable estimate
	if outstandingShares <= 0 {
		// Use 10 million shares as a default reasonable estimate
		outstandingShares = 10000000
	}

	priceTargets, err := CalculateMNAVPriceTargets(company.BTCHoldings, btcPrice, outstandingShares, 1.0, 5.0, 0.25)
	if err != nil {
		return nil, fmt.Errorf("error calculating mNAV price targets: %w", err)
	}

	return &BitcoinMetrics{
		Company:          company,
		BTCPrice:         btcPrice,
		MNAV:             mnav,
		DaysToCover:      daysToCover,
		BTCValue:         btcValue,
		BTCYieldDaily:    btcYieldDaily,
		MNAVPriceTargets: priceTargets,
	}, nil
}
