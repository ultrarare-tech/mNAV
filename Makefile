# mNAV Project Makefile - Bitcoin Treasury Analysis Tools

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

.PHONY: all build clean test deps help
.PHONY: collection-tools analysis-tools interpretation-tools

# Default target
all: collection-tools analysis-tools interpretation-tools

# =============================================================================
# CATEGORY BUILDERS
# =============================================================================

# Build all collection tools
collection-tools: bitcoin-historical stock-data edgar-data
	@echo "✅ Collection tools built successfully"

# Build all analysis tools  
analysis-tools: mnav-historical mnav-chart comprehensive-analysis
	@echo "✅ Analysis tools built successfully"

# Build all interpretation tools
interpretation-tools: bitcoin-parser
	@echo "✅ Interpretation tools built successfully"

# =============================================================================
# INDIVIDUAL TOOL BUILDERS
# =============================================================================

# Collection Tools
bitcoin-historical:
	@echo "🔨 Building bitcoin-historical..."
	@mkdir -p bin
	@go build -o bin/bitcoin-historical cmd/collection/bitcoin-historical/main.go

stock-data:
	@echo "🔨 Building stock-data..."
	@mkdir -p bin
	@go build -o bin/stock-data cmd/collection/stock-data/main.go

edgar-data:
	@echo "🔨 Building edgar-data..."
	@mkdir -p bin
	@go build -o bin/edgar-data cmd/collection/edgar-data/main.go

# Analysis Tools
mnav-historical:
	@echo "🔨 Building mnav-historical..."
	@mkdir -p bin
	@go build -o bin/mnav-historical cmd/analysis/mnav-historical/main.go

mnav-chart:
	@echo "🔨 Building mnav-chart..."
	@mkdir -p bin
	@go build -o bin/mnav-chart cmd/analysis/mnav-chart/main.go

comprehensive-analysis:
	@echo "🔨 Building comprehensive-analysis..."
	@mkdir -p bin
	@go build -o bin/comprehensive-analysis cmd/analysis/comprehensive-analysis/main.go

# Interpretation Tools
bitcoin-parser:
	@echo "🔨 Building bitcoin-parser..."
	@mkdir -p bin
	@go build -o bin/bitcoin-parser cmd/interpretation/bitcoin-parser/main.go

# =============================================================================
# UTILITY TARGETS
# =============================================================================

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf bin/

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Development helpers
dev-setup: deps
	@echo "🛠️  Setting up development environment..."
	@mkdir -p data/bitcoin-prices/historical
	@mkdir -p data/stock-data
	@mkdir -p data/analysis/mnav
	@mkdir -p data/charts
	@mkdir -p debug

# =============================================================================
# WORKFLOWS
# =============================================================================

# Complete mNAV analysis workflow for MSTR
workflow-mstr: all
	@echo "🚀 Running complete MSTR mNAV analysis workflow..."
	@echo "Step 1: Collecting Bitcoin price history..."
	@./bin/bitcoin-historical -start=2020-08-11 || echo "⚠️ Bitcoin price collection failed"
	@echo "Step 2: Collecting MSTR stock data..."
	@./bin/stock-data -symbol=MSTR -start=2020-08-11 || echo "⚠️ Stock data collection failed (check API keys)"
	@echo "Step 3: Calculating historical mNAV..."
	@./bin/mnav-historical -symbol=MSTR -start=2020-08-11 || echo "⚠️ mNAV calculation failed"
	@echo "Step 4: Generating mNAV chart..."
	@./bin/mnav-chart -format=html || echo "⚠️ Chart generation failed"
	@echo "✅ Workflow complete! Check data/charts/ for results"

# Demo the application capabilities
demo:
	@echo "📋 mNAV Bitcoin Treasury Analysis Tool"
	@echo ""
	@echo "🗂️  DATA COLLECTION TOOLS:"
	@echo "   bitcoin-historical   - Download historical Bitcoin prices"
	@echo "   stock-data          - Collect stock prices & company data"
	@echo "   edgar-data          - Download SEC filings"
	@echo ""
	@echo "📊 ANALYSIS TOOLS:"
	@echo "   mnav-historical     - Calculate historical mNAV ratios"
	@echo "   mnav-chart          - Generate interactive charts"
	@echo "   comprehensive-analysis - Complete analysis suite"
	@echo ""
	@echo "🔍 INTERPRETATION TOOLS:"
	@echo "   bitcoin-parser      - Extract Bitcoin transactions from filings"
	@echo ""
	@echo "🚀 WORKFLOWS:"
	@echo "   make workflow-mstr  - Complete MSTR analysis"
	@echo "   make all           - Build all tools"
	@echo "   make dev-setup     - Setup development environment"

# Help target
help:
	@echo "📋 mNAV - Bitcoin Treasury Analysis Tool"
	@echo ""
	@echo "🏗️  BUILD COMMANDS:"
	@echo "   make all                - Build all tools"
	@echo "   make collection-tools   - Build data collection tools"
	@echo "   make analysis-tools     - Build analysis tools"
	@echo "   make interpretation-tools - Build interpretation tools"
	@echo ""
	@echo "🔧 INDIVIDUAL TOOLS:"
	@echo "   make bitcoin-historical - Historical Bitcoin price collector"
	@echo "   make stock-data        - Stock data collector (FMP + Alpha Vantage)"
	@echo "   make edgar-data        - SEC filing downloader"
	@echo "   make mnav-historical   - Historical mNAV calculator"
	@echo "   make mnav-chart        - Interactive chart generator"
	@echo "   make bitcoin-parser    - Bitcoin transaction extractor"
	@echo ""
	@echo "🛠️  UTILITY COMMANDS:"
	@echo "   make clean             - Clean build artifacts"
	@echo "   make test              - Run tests"
	@echo "   make deps              - Download dependencies"
	@echo "   make dev-setup         - Setup development environment"
	@echo ""
	@echo "🚀 WORKFLOWS:"
	@echo "   make workflow-mstr     - Complete MSTR analysis workflow"
	@echo "   make demo             - Show available tools"
	@echo ""
	@echo "📋 PREREQUISITES:"
	@echo "   • Go 1.21+ installed"
	@echo "   • API keys in .env file:"
	@echo "     - FMP_API_KEY (Financial Modeling Prep)"
	@echo "     - ALPHA_VANTAGE_API_KEY (Alpha Vantage)"
	@echo "     - GROK_API_KEY (for Bitcoin parsing)"
	@echo ""
	@echo "📚 DOCUMENTATION:"
	@echo "   • README.md - Main documentation"
	@echo "   • docs/mNAV_CHARTING.md - Charting guide"
	@echo "   • ARCHITECTURE.md - System architecture" 