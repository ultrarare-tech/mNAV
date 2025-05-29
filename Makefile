# mNAV Project Makefile - Organized by Data Flow Categories

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# =============================================================================
# DATA COLLECTION TOOLS - Gather raw data from external sources
# =============================================================================
EDGAR_DATA_BINARY=bin/collection/edgar-data
EDGAR_DATA_SRC=cmd/collection/edgar-data

# =============================================================================
# DATA INTERPRETATION TOOLS - Parse and extract structured data
# =============================================================================
BITCOIN_PARSER_BINARY=bin/interpretation/bitcoin-parser
BITCOIN_PARSER_SRC=cmd/interpretation/bitcoin-parser

GROK_TEST_BINARY=bin/interpretation/grok-test
GROK_TEST_SRC=cmd/interpretation/grok-test

# =============================================================================
# DATA ANALYSIS TOOLS - Calculate metrics and insights
# =============================================================================
MNAV_CALCULATOR_BINARY=bin/analysis/mnav-calculator
MNAV_CALCULATOR_SRC=cmd/analysis/mnav-calculator

# =============================================================================
# LEGACY TOOLS - Original commands (transitional)
# =============================================================================
EDGAR_ENHANCED_BINARY=bin/legacy/edgar-enhanced
RAW_FILING_MANAGER_BINARY=bin/legacy/raw-filing-manager
VALIDATE_GROK_BINARY=bin/legacy/validate-grok

EDGAR_ENHANCED_SRC=cmd/edgar-enhanced
RAW_FILING_MANAGER_SRC=cmd/raw-filing-manager
VALIDATE_GROK_SRC=cmd/validate-grok

.PHONY: all build clean test deps help
.PHONY: collection interpretation analysis legacy
.PHONY: edgar-data bitcoin-parser mnav-calculator

# Default target
all: build

# Build all categories
build: collection interpretation analysis

# =============================================================================
# CATEGORY BUILDERS
# =============================================================================

# Build all collection tools
collection: $(EDGAR_DATA_BINARY)
	@echo "✅ Collection tools built successfully"

# Build all interpretation tools  
interpretation: $(BITCOIN_PARSER_BINARY) $(GROK_TEST_BINARY)
	@echo "✅ Interpretation tools built successfully"

# Build all analysis tools
analysis: $(MNAV_CALCULATOR_BINARY)
	@echo "✅ Analysis tools built successfully"

# Build legacy tools (when needed)
legacy: $(RAW_FILING_MANAGER_BINARY) $(VALIDATE_GROK_BINARY)
	@echo "✅ Legacy tools built successfully"

# =============================================================================
# INDIVIDUAL TOOL BUILDERS
# =============================================================================

# Collection Tools
$(EDGAR_DATA_BINARY):
	@echo "🗂️  Building EDGAR data collection tool..."
	@mkdir -p bin/collection
	$(GOBUILD) -o $(EDGAR_DATA_BINARY) ./$(EDGAR_DATA_SRC)

# Interpretation Tools
$(BITCOIN_PARSER_BINARY):
	@echo "🔍 Building Bitcoin transaction parser..."
	@mkdir -p bin/interpretation
	$(GOBUILD) -o $(BITCOIN_PARSER_BINARY) ./$(BITCOIN_PARSER_SRC)

$(GROK_TEST_BINARY):
	@echo "🤖 Building Grok AI test tool..."
	@mkdir -p bin/interpretation
	$(GOBUILD) -o $(GROK_TEST_BINARY) ./$(GROK_TEST_SRC)

# Analysis Tools
$(MNAV_CALCULATOR_BINARY):
	@echo "📊 Building mNAV calculator..."
	@mkdir -p bin/analysis
	$(GOBUILD) -o $(MNAV_CALCULATOR_BINARY) ./$(MNAV_CALCULATOR_SRC)

# Legacy Tools
$(RAW_FILING_MANAGER_BINARY):
	@echo "📁 Building raw filing manager (legacy)..."
	@mkdir -p bin/legacy
	$(GOBUILD) -o $(RAW_FILING_MANAGER_BINARY) ./$(RAW_FILING_MANAGER_SRC)

$(VALIDATE_GROK_BINARY):
	@echo "🤖 Building Grok validator (legacy)..."
	@mkdir -p bin/legacy
	$(GOBUILD) -o $(VALIDATE_GROK_BINARY) ./$(VALIDATE_GROK_SRC)

# =============================================================================
# INDIVIDUAL TARGETS
# =============================================================================
edgar-data: $(EDGAR_DATA_BINARY)
bitcoin-parser: $(BITCOIN_PARSER_BINARY)
grok-test: $(GROK_TEST_BINARY)
mnav-calculator: $(MNAV_CALCULATOR_BINARY)

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
	@mkdir -p data/edgar/companies
	@mkdir -p debug

# =============================================================================
# EXAMPLE WORKFLOWS
# =============================================================================

# Complete workflow example: MSTR analysis
workflow-mstr: collection interpretation analysis
	@echo "🚀 Running complete MSTR analysis workflow..."
	@echo "Step 1: Collecting EDGAR filings for MSTR..."
	./$(EDGAR_DATA_BINARY) -ticker=MSTR -filing-types="8-K,10-Q,10-K" -start=2023-01-01 -dry-run
	@echo "Step 2: Parsing Bitcoin transactions..."
	./$(BITCOIN_PARSER_BINARY) -ticker=MSTR -dry-run
	@echo "Step 3: Calculating mNAV metrics..."
	./$(MNAV_CALCULATOR_BINARY) -symbols=MSTR -verbose

# Demo the new categorized structure
demo:
	@echo "📋 mNAV Tool Categories:"
	@echo ""
	@echo "🗂️  DATA COLLECTION:"
	@echo "   edgar-data        - Downloads SEC filings"
	@echo ""
	@echo "🔍 DATA INTERPRETATION:"
	@echo "   bitcoin-parser    - Extracts Bitcoin transactions"
	@echo "   grok-test         - Tests Grok AI integration"
	@echo ""
	@echo "📊 DATA ANALYSIS:"
	@echo "   mnav-calculator   - Calculates mNAV metrics"
	@echo ""
	@echo "🏗️  Build commands:"
	@echo "   make collection   - Build all collection tools"
	@echo "   make interpretation - Build all interpretation tools"
	@echo "   make analysis     - Build all analysis tools"

# Help target
help:
	@echo "mNAV Project - Organized Build System"
	@echo "===================================="
	@echo ""
	@echo "📂 CATEGORY TARGETS:"
	@echo "  collection       - Build data collection tools"
	@echo "  interpretation   - Build data interpretation tools"
	@echo "  analysis         - Build data analysis tools"
	@echo "  legacy           - Build legacy tools"
	@echo ""
	@echo "🔧 INDIVIDUAL TOOLS:"
	@echo "  edgar-data       - SEC filing collector"
	@echo "  bitcoin-parser   - Bitcoin transaction extractor"
	@echo "  grok-test        - Grok AI integration tester"
	@echo "  mnav-calculator  - mNAV metrics calculator"
	@echo ""
	@echo "🛠️  UTILITY TARGETS:"
	@echo "  all              - Build all tools (default)"
	@echo "  build            - Build all tools"
	@echo "  clean            - Clean build artifacts"
	@echo "  test             - Run tests"
	@echo "  deps             - Download dependencies"
	@echo "  dev-setup        - Set up development environment"
	@echo ""
	@echo "🚀 WORKFLOWS:"
	@echo "  workflow-mstr    - Complete MSTR analysis demo"
	@echo "  demo             - Show tool organization"
	@echo "  help             - Show this help" 