package analyzer

import (
	"fmt"

	"github.com/ultrarare-tech/mNAV/pkg/config"
)

// RebalancingRule represents a single row in the dynamic rebalancing table
type RebalancingRule struct {
	DownThreshold float64 // mNAV level when dropping
	UpThreshold   float64 // mNAV level when rising
	TargetRatio   float64 // Target Bitcoin:MSTR ratio (X:1)
}

// DynamicRebalancingTable contains all rebalancing rules
type DynamicRebalancingTable struct {
	Rules  []RebalancingRule
	config *config.RebalancingConfig
}

// NewDynamicRebalancingTable creates the rebalancing table from configuration
func NewDynamicRebalancingTable() (*DynamicRebalancingTable, error) {
	// Load configuration from CSV/JSON
	rebalanceConfig, err := config.LoadRebalancingConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load rebalancing configuration: %w", err)
	}

	// Validate configuration
	if err := rebalanceConfig.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid rebalancing configuration: %w", err)
	}

	// Convert config rules to analyzer rules
	var rules []RebalancingRule
	for _, configRule := range rebalanceConfig.Rules {
		rules = append(rules, RebalancingRule{
			DownThreshold: configRule.MinThreshold,
			UpThreshold:   configRule.MaxThreshold,
			TargetRatio:   configRule.TargetRatio,
		})
	}

	return &DynamicRebalancingTable{
		Rules:  rules,
		config: rebalanceConfig,
	}, nil
}

// GetConfigSummary returns a summary of the loaded configuration
func (dt *DynamicRebalancingTable) GetConfigSummary() string {
	return dt.config.GetSummary()
}

// GetTargetRatio determines the target Bitcoin:MSTR ratio based on current mNAV and transition thresholds
func (dt *DynamicRebalancingTable) GetTargetRatio(currentMNAV float64) (float64, string, error) {
	// First, check if mNAV falls within any ratio's Down-Up range
	for _, rule := range dt.Rules {
		if currentMNAV >= rule.DownThreshold && currentMNAV <= rule.UpThreshold {
			return rule.TargetRatio,
				fmt.Sprintf("mNAV %.4f within range [%.4f - %.4f] - maintain %v:1 ratio",
					currentMNAV, rule.DownThreshold, rule.UpThreshold, rule.TargetRatio), nil
		}
	}

	// If not in any range, find transition points
	for i := 0; i < len(dt.Rules); i++ {
		rule := dt.Rules[i]

		// If mNAV is above the Up threshold, we need to move to a lower ratio (less MSTR)
		if currentMNAV > rule.UpThreshold {
			// Look for the next lower ratio (higher index = lower ratio)
			for j := i + 1; j < len(dt.Rules); j++ {
				nextRule := dt.Rules[j]
				if currentMNAV >= nextRule.DownThreshold {
					return nextRule.TargetRatio,
						fmt.Sprintf("mNAV %.4f > %.4f - transition from %v:1 to %v:1 (sell MSTR)",
							currentMNAV, rule.UpThreshold, rule.TargetRatio, nextRule.TargetRatio), nil
				}
			}
			// If no lower ratio found, use minimum ratio
			return dt.Rules[len(dt.Rules)-1].TargetRatio,
				fmt.Sprintf("mNAV %.4f very high - use minimum MSTR ratio %v:1",
					currentMNAV, dt.Rules[len(dt.Rules)-1].TargetRatio), nil
		}

		// If mNAV is below the Down threshold, we need to move to a higher ratio (more MSTR)
		if currentMNAV < rule.DownThreshold {
			// Look for the next higher ratio (lower index = higher ratio)
			if i > 0 {
				prevRule := dt.Rules[i-1]
				return prevRule.TargetRatio,
					fmt.Sprintf("mNAV %.4f < %.4f - transition from %v:1 to %v:1 (buy MSTR)",
						currentMNAV, rule.DownThreshold, rule.TargetRatio, prevRule.TargetRatio), nil
			} else {
				// Already at highest ratio
				return rule.TargetRatio,
					fmt.Sprintf("mNAV %.4f very low - use maximum MSTR ratio %v:1",
						currentMNAV, rule.TargetRatio), nil
			}
		}
	}

	// Fallback - should not reach here with proper table
	return dt.Rules[len(dt.Rules)-1].TargetRatio,
		fmt.Sprintf("mNAV %.4f - fallback to conservative ratio", currentMNAV), nil
}

// RebalanceRecommendation contains the recommended portfolio adjustments
type RebalanceRecommendation struct {
	CurrentRatio      float64
	TargetRatio       float64
	IsWellBalanced    bool
	RecommendedAction string
	FBTCAction        string // "BUY" or "SELL"
	FBTCShares        float64
	FBTCValue         float64
	MSTRAction        string // "BUY" or "SELL"
	MSTRShares        float64
	MSTRValue         float64
	Explanation       string
}

// CalculateRebalanceRecommendation determines what trades are needed
func (dt *DynamicRebalancingTable) CalculateRebalanceRecommendation(
	currentMNAV float64,
	fbtcValue, mstrValue float64,
	fbtcPrice, mstrPrice float64,
) (*RebalanceRecommendation, error) {

	// Get target ratio based on current mNAV
	targetRatio, explanation, err := dt.GetTargetRatio(currentMNAV)
	if err != nil {
		return nil, err
	}

	// Calculate current ratio (Bitcoin:MSTR)
	currentRatio := fbtcValue / mstrValue

	// Define "well balanced" tolerance (within 5% of target)
	tolerance := 0.05
	ratioTolerance := targetRatio * tolerance

	recommendation := &RebalanceRecommendation{
		CurrentRatio:   currentRatio,
		TargetRatio:    targetRatio,
		IsWellBalanced: false,
		Explanation:    explanation,
	}

	// Check if portfolio is well balanced
	if currentRatio >= (targetRatio-ratioTolerance) && currentRatio <= (targetRatio+ratioTolerance) {
		recommendation.IsWellBalanced = true
		recommendation.RecommendedAction = "HOLD - Portfolio is well balanced"
		return recommendation, nil
	}

	// Calculate total Bitcoin-related value
	totalBitcoinValue := fbtcValue + mstrValue

	// Calculate target allocations
	targetFBTCValue := totalBitcoinValue * (targetRatio / (targetRatio + 1))
	targetMSTRValue := totalBitcoinValue * (1 / (targetRatio + 1))

	// Calculate differences
	fbtcDifference := targetFBTCValue - fbtcValue
	mstrDifference := targetMSTRValue - mstrValue

	if currentRatio > targetRatio {
		// Too much FBTC, need more MSTR
		recommendation.RecommendedAction = "REBALANCE - Reduce FBTC, Increase MSTR"
		recommendation.FBTCAction = "SELL"
		recommendation.FBTCShares = -fbtcDifference / fbtcPrice
		recommendation.FBTCValue = -fbtcDifference
		recommendation.MSTRAction = "BUY"
		recommendation.MSTRShares = mstrDifference / mstrPrice
		recommendation.MSTRValue = mstrDifference
	} else {
		// Too much MSTR, need more FBTC
		recommendation.RecommendedAction = "REBALANCE - Increase FBTC, Reduce MSTR"
		recommendation.FBTCAction = "BUY"
		recommendation.FBTCShares = fbtcDifference / fbtcPrice
		recommendation.FBTCValue = fbtcDifference
		recommendation.MSTRAction = "SELL"
		recommendation.MSTRShares = -mstrDifference / mstrPrice
		recommendation.MSTRValue = -mstrDifference
	}

	return recommendation, nil
}

// PrintRebalanceRecommendation formats and displays the recommendation
func (r *RebalanceRecommendation) Print() {
	fmt.Printf("🎯 DYNAMIC REBALANCING ANALYSIS\n")
	fmt.Printf("===============================\n\n")

	fmt.Printf("📊 Ratio Analysis:\n")
	fmt.Printf("   Current Bitcoin:MSTR Ratio: %.2f:1\n", r.CurrentRatio)
	fmt.Printf("   Target Bitcoin:MSTR Ratio:  %.2f:1\n", r.TargetRatio)
	fmt.Printf("   %s\n\n", r.Explanation)

	if r.IsWellBalanced {
		fmt.Printf("✅ PORTFOLIO IS WELL BALANCED\n")
		fmt.Printf("   No rebalancing needed at current mNAV levels\n")
		fmt.Printf("   Continue monitoring mNAV for future adjustments\n\n")
		return
	}

	fmt.Printf("⚖️ REBALANCING RECOMMENDED\n")
	fmt.Printf("   %s\n\n", r.RecommendedAction)

	fmt.Printf("💱 Recommended Trades:\n")
	if r.FBTCAction == "BUY" {
		fmt.Printf("   📈 BUY %.2f shares of FBTC (~$%.2f)\n", r.FBTCShares, r.FBTCValue)
	} else {
		fmt.Printf("   📉 SELL %.2f shares of FBTC (~$%.2f)\n", r.FBTCShares, -r.FBTCValue)
	}

	if r.MSTRAction == "BUY" {
		fmt.Printf("   📈 BUY %.2f shares of MSTR (~$%.2f)\n", r.MSTRShares, r.MSTRValue)
	} else {
		fmt.Printf("   📉 SELL %.2f shares of MSTR (~$%.2f)\n", r.MSTRShares, -r.MSTRValue)
	}

	fmt.Printf("\n🎯 After Rebalancing:\n")
	fmt.Printf("   New Bitcoin:MSTR Ratio: %.2f:1\n", r.TargetRatio)
	fmt.Printf("   Portfolio optimized for current mNAV level\n\n")
}
