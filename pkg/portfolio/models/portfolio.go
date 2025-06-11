package models

import (
	"time"
)

// Position represents a single position in the portfolio
type Position struct {
	AccountNumber    string  `json:"account_number" csv:"Account Number"`
	AccountName      string  `json:"account_name" csv:"Account Name"`
	Symbol           string  `json:"symbol" csv:"Symbol"`
	Description      string  `json:"description" csv:"Description"`
	Quantity         float64 `json:"quantity" csv:"Quantity"`
	LastPrice        float64 `json:"last_price" csv:"Last Price"`
	LastPriceChange  float64 `json:"last_price_change" csv:"Last Price Change"`
	CurrentValue     float64 `json:"current_value" csv:"Current Value"`
	TodayGainLoss    float64 `json:"today_gain_loss_dollar" csv:"Today's Gain/Loss Dollar"`
	TodayGainLossPct float64 `json:"today_gain_loss_percent" csv:"Today's Gain/Loss Percent"`
	TotalGainLoss    float64 `json:"total_gain_loss_dollar" csv:"Total Gain/Loss Dollar"`
	TotalGainLossPct float64 `json:"total_gain_loss_percent" csv:"Total Gain/Loss Percent"`
	PercentOfAccount float64 `json:"percent_of_account" csv:"Percent Of Account"`
	CostBasisTotal   float64 `json:"cost_basis_total" csv:"Cost Basis Total"`
	AverageCostBasis float64 `json:"average_cost_basis" csv:"Average Cost Basis"`
	Type             string  `json:"type" csv:"Type"`
}

// Portfolio represents a complete portfolio snapshot
type Portfolio struct {
	Date             time.Time           `json:"date"`
	SourceFile       string              `json:"source_file"`
	Positions        []Position          `json:"positions"`
	Accounts         map[string]*Account `json:"accounts"`
	TotalValue       float64             `json:"total_value"`
	TotalCostBasis   float64             `json:"total_cost_basis"`
	TotalGainLoss    float64             `json:"total_gain_loss"`
	TotalGainLossPct float64             `json:"total_gain_loss_percent"`
	AssetAllocation  AssetAllocation     `json:"asset_allocation"`
	CreatedAt        time.Time           `json:"created_at"`
}

// Account represents account-level aggregation
type Account struct {
	AccountNumber    string     `json:"account_number"`
	AccountName      string     `json:"account_name"`
	TotalValue       float64    `json:"total_value"`
	TotalCostBasis   float64    `json:"total_cost_basis"`
	TotalGainLoss    float64    `json:"total_gain_loss"`
	TotalGainLossPct float64    `json:"total_gain_loss_percent"`
	Positions        []Position `json:"positions"`
}

// AssetAllocation represents portfolio allocation breakdown
type AssetAllocation struct {
	FBTCValue       float64 `json:"fbtc_value"`
	FBTCPercent     float64 `json:"fbtc_percent"`
	MSTRValue       float64 `json:"mstr_value"`
	MSTRPercent     float64 `json:"mstr_percent"`
	GLDValue        float64 `json:"gld_value"`
	GLDPercent      float64 `json:"gld_percent"`
	OtherValue      float64 `json:"other_value"`
	OtherPercent    float64 `json:"other_percent"`
	BitcoinExposure float64 `json:"bitcoin_exposure"`
	BitcoinPercent  float64 `json:"bitcoin_percent"`
	FBTCMSTRRatio   float64 `json:"fbtc_mstr_ratio"`
}

// SymbolSummary represents aggregated data for a specific symbol across all accounts
type SymbolSummary struct {
	Symbol           string  `json:"symbol"`
	Description      string  `json:"description"`
	TotalQuantity    float64 `json:"total_quantity"`
	TotalValue       float64 `json:"total_value"`
	TotalCostBasis   float64 `json:"total_cost_basis"`
	TotalGainLoss    float64 `json:"total_gain_loss"`
	TotalGainLossPct float64 `json:"total_gain_loss_percent"`
	LastPrice        float64 `json:"last_price"`
	PercentOfTotal   float64 `json:"percent_of_total"`
}

// HistoricalTracking represents portfolio changes over time
type HistoricalTracking struct {
	Date             time.Time          `json:"date"`
	TotalValue       float64            `json:"total_value"`
	TotalGainLoss    float64            `json:"total_gain_loss"`
	AssetAllocation  AssetAllocation    `json:"asset_allocation"`
	TopHoldings      []SymbolSummary    `json:"top_holdings"`
	AccountBreakdown map[string]float64 `json:"account_breakdown"`
	Changes          *PortfolioChanges  `json:"changes,omitempty"`
}

// PortfolioChanges represents changes from previous period
type PortfolioChanges struct {
	ValueChange        float64            `json:"value_change"`
	ValueChangePercent float64            `json:"value_change_percent"`
	NewPositions       []string           `json:"new_positions"`
	ClosedPositions    []string           `json:"closed_positions"`
	AllocationChanges  map[string]float64 `json:"allocation_changes"`
}

// PortfolioSummary represents a high-level portfolio summary
type PortfolioSummary struct {
	Date            time.Time       `json:"date"`
	TotalValue      float64         `json:"total_value"`
	BitcoinExposure float64         `json:"bitcoin_exposure"`
	FBTCMSTRRatio   float64         `json:"fbtc_mstr_ratio"`
	TopSymbols      []SymbolSummary `json:"top_symbols"`
	AssetAllocation AssetAllocation `json:"asset_allocation"`
}

// RebalanceRecommendation represents suggested portfolio rebalancing
type RebalanceRecommendation struct {
	CurrentRatio    float64            `json:"current_ratio"`
	TargetRatio     float64            `json:"target_ratio"`
	TradeAmount     float64            `json:"trade_amount"`
	FBTCToSell      float64            `json:"fbtc_to_sell"`
	MSTRToBuy       float64            `json:"mstr_to_buy"`
	NewAllocation   AssetAllocation    `json:"new_allocation"`
	ReasonableRange bool               `json:"reasonable_range"`
	Trades          []RecommendedTrade `json:"trades"`
}

// RecommendedTrade represents a specific trade recommendation
type RecommendedTrade struct {
	Action         string  `json:"action"` // "BUY" or "SELL"
	Symbol         string  `json:"symbol"`
	Shares         float64 `json:"shares"`
	EstimatedValue float64 `json:"estimated_value"`
	Account        string  `json:"account"`
}
