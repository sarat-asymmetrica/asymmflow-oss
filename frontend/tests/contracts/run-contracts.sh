#!/bin/bash
#
# ACE Engine API Contract Test Runner
# Automated validation for Nigeria production deployment
#
# Usage:
#   ./run-contracts.sh              # Run all tests
#   ./run-contracts.sh --ui          # Run with UI mode
#   ./run-contracts.sh --debug       # Run with debugger
#   ./run-contracts.sh --report      # Generate HTML report
#

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                                                                ║"
echo "║    ACE Engine API Contract Test Suite                         ║"
echo "║    Scientific Validation for Production Deployment            ║"
echo "║                                                                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if backend is running
echo -e "${YELLOW}[1/4] Checking backend availability...${NC}"
if curl -s -f http://localhost:5000/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Backend is running on port 5000${NC}"
else
    echo -e "${RED}✗ Backend is NOT running on port 5000${NC}"
    echo -e "${YELLOW}Please start the backend:${NC}"
    echo "  cd C:\\Projects\\ACE Engine\\Asymmetrica.Runtime\\Asymmetrica.Runtime.Host"
    echo "  dotnet run"
    exit 1
fi

# Check Node.js dependencies
echo -e "${YELLOW}[2/4] Checking Node.js dependencies...${NC}"
if [ -d "node_modules/@playwright" ]; then
    echo -e "${GREEN}✓ Playwright is installed${NC}"
else
    echo -e "${YELLOW}Installing Playwright...${NC}"
    npm install
fi

# Verify test files exist
echo -e "${YELLOW}[3/4] Verifying test files...${NC}"
if [ -f "tests/contracts/api.spec.ts" ]; then
    echo -e "${GREEN}✓ Contract tests found (api.spec.ts)${NC}"
else
    echo -e "${RED}✗ Contract tests NOT found${NC}"
    exit 1
fi

# Run tests based on argument
echo -e "${YELLOW}[4/4] Running contract tests...${NC}"
echo ""

if [ "$1" == "--ui" ]; then
    echo -e "${BLUE}Running in UI mode (interactive)...${NC}"
    npx playwright test tests/contracts/api.spec.ts --ui
elif [ "$1" == "--debug" ]; then
    echo -e "${BLUE}Running in debug mode...${NC}"
    npx playwright test tests/contracts/api.spec.ts --debug
elif [ "$1" == "--report" ]; then
    echo -e "${BLUE}Generating HTML report...${NC}"
    npx playwright test tests/contracts/api.spec.ts --reporter=html
    npx playwright show-report
else
    echo -e "${BLUE}Running all contract tests...${NC}"
    npx playwright test tests/contracts/api.spec.ts --reporter=list
fi

# Check exit code
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                                                                ║"
    echo "║    ✓ All Contract Tests Passed!                               ║"
    echo "║    Production Ready for Nigeria Deployment                    ║"
    echo "║                                                                ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
else
    echo ""
    echo -e "${RED}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                                                                ║"
    echo "║    ✗ Some Contract Tests Failed                               ║"
    echo "║    Review failures before deployment                          ║"
    echo "║                                                                ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    exit 1
fi
