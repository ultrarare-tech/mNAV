package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MNAVPriceTarget represents a stock price target for a specific mNAV value
type MNAVPriceTarget struct {
	MNAV        float64 `json:"mnav"`
	TargetPrice float64 `json:"targetPrice"`
}

// CompanyData represents a single company's data
type CompanyData struct {
	Symbol            string            `json:"symbol"`
	Name              string            `json:"name"`
	OutstandingShares float64           `json:"outstandingShares"`
	BTCHoldings       float64           `json:"btcHoldings"`
	BTCYield          float64           `json:"btcYield"`
	MarketCap         float64           `json:"marketCap"`
	LastUpdated       time.Time         `json:"lastUpdated"`
	MNAVPriceTargets  []MNAVPriceTarget `json:"mnavPriceTargets,omitempty"`
	DaysToCover       float64           `json:"daysToCover,omitempty"`
}

// CompaniesConfig represents the structure of the companies.json file
type CompaniesConfig struct {
	Companies []CompanyData `json:"companies"`
}

// LoadCompaniesConfig loads company data from the JSON file
func LoadCompaniesConfig(basePath string) (*CompaniesConfig, error) {
	// Determine the path to the JSON file
	jsonPath := filepath.Join(basePath, "data", "companies.json")

	// Read the file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read companies config: %w", err)
	}

	// Parse the JSON
	var config CompaniesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse companies config: %w", err)
	}

	return &config, nil
}

// GetCompanyBySymbol returns company data for a specific symbol
func (c *CompaniesConfig) GetCompanyBySymbol(symbol string) (CompanyData, bool) {
	for _, company := range c.Companies {
		if company.Symbol == symbol {
			return company, true
		}
	}
	return CompanyData{}, false
}

// UpdateCompany updates the data for a specific company
func (c *CompaniesConfig) UpdateCompany(updatedCompany CompanyData) bool {
	// Set the LastUpdated timestamp to now
	updatedCompany.LastUpdated = time.Now()

	for i, company := range c.Companies {
		if company.Symbol == updatedCompany.Symbol {
			c.Companies[i] = updatedCompany
			return true
		}
	}
	return false
}

// SaveCompaniesConfig saves the companies configuration back to the JSON file
func (c *CompaniesConfig) SaveCompaniesConfig(basePath string) error {
	// Determine the path to the JSON file
	jsonPath := filepath.Join(basePath, "data", "companies.json")

	// Marshal the JSON with indentation for readability
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal companies config: %w", err)
	}

	// Write the file
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write companies config: %w", err)
	}

	return nil
}

// UpdateMNAVPriceTargets sets the mNAV price targets for a company
func (c *CompanyData) UpdateMNAVPriceTargets(priceTargets map[float64]float64) {
	// Clear existing targets
	c.MNAVPriceTargets = []MNAVPriceTarget{}

	// Convert the map to a slice of MNAVPriceTarget for JSON serialization
	for mnav, price := range priceTargets {
		c.MNAVPriceTargets = append(c.MNAVPriceTargets, MNAVPriceTarget{
			MNAV:        mnav,
			TargetPrice: price,
		})
	}
}
