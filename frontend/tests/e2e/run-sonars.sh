#!/bin/bash

###############################################################################
# Six Sonar E2E Test Runner
# Unified Intelligence Monitoring System Validation
#
# Usage:
#   ./run-sonars.sh                  # Run all sonars
#   ./run-sonars.sh --headed         # Run with visible browser
#   ./run-sonars.sh --debug          # Run in debug mode
#   ./run-sonars.sh --shm            # Run only SHM calculation
#   ./run-sonars.sh --baseline       # Collect baseline metrics
#   ./run-sonars.sh --nigeria        # Enforce Nigeria deployment (SHM ≥ 0.85)
###############################################################################

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║          SIX SONAR E2E VALIDATION SYSTEM                       ║"
echo "║          Unified Intelligence Monitoring                       ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if dev server is running
echo -e "${YELLOW}Checking dev server...${NC}"
if ! curl -s http://127.0.0.1:5173 > /dev/null; then
    echo -e "${RED}ERROR: Dev server not running!${NC}"
    echo -e "${YELLOW}Please start it with: npm run dev${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Dev server is running${NC}"

# Parse arguments
MODE="all"
HEADED=""
DEBUG=""
ENV_VARS=""

for arg in "$@"; do
    case $arg in
        --headed)
            HEADED="--headed"
            ;;
        --debug)
            DEBUG="--debug"
            ;;
        --shm)
            MODE="shm"
            ;;
        --baseline)
            MODE="baseline"
            ;;
        --nigeria)
            ENV_VARS="DEPLOYMENT=nigeria"
            echo -e "${BLUE}🇳🇬 Nigeria deployment mode: SHM ≥ 0.85 required${NC}"
            ;;
        *)
            ;;
    esac
done

# Determine which tests to run
GREP_PATTERN=""
case $MODE in
    shm)
        GREP_PATTERN="-g SHM"
        echo -e "${BLUE}Running System Health Metric (SHM) calculation...${NC}"
        ;;
    baseline)
        GREP_PATTERN="-g Baseline"
        echo -e "${BLUE}Collecting baseline metrics...${NC}"
        ;;
    all)
        echo -e "${BLUE}Running all 16 Sonar tests...${NC}"
        ;;
esac

# Run tests
echo -e "${GREEN}Starting test execution...${NC}"
$ENV_VARS npx playwright test tests/e2e/sonars.spec.ts $GREP_PATTERN $HEADED $DEBUG

# Check exit code
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                  ✓ ALL TESTS PASSED!                           ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    # Show report location
    echo -e "${BLUE}Test results saved to:${NC}"
    echo "  - HTML Report: test-results/index.html"
    echo "  - JSON Results: test-results/results.json"
    echo "  - JUnit XML: test-results/results.xml"

    if [ "$MODE" = "baseline" ]; then
        echo -e "${GREEN}Baseline saved to: test-results/baselines/${NC}"
    fi

    # Open report (optional)
    echo -e "${YELLOW}Open HTML report? (y/n)${NC}"
    read -r response
    if [ "$response" = "y" ]; then
        npx playwright show-report
    fi
else
    echo -e "${RED}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                  ✗ TESTS FAILED                                ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    echo -e "${YELLOW}Check test-results/ directory for details${NC}"
    echo -e "${YELLOW}Run with --headed to see browser interactions${NC}"
    echo -e "${YELLOW}Run with --debug to step through failures${NC}"

    exit $EXIT_CODE
fi
