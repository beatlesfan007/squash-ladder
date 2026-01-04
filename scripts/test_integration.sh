#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${YELLOW}Running Integration Tests...${NC}"

# Load environment variables from .env if it exists
if [ -f .env ]; then
    export $(cat .env | xargs)
fi

if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}Error: DATABASE_URL is not set.${NC}"
    echo "Please set DATABASE_URL or configure it in .env"
    exit 1
fi

echo -e "${YELLOW}WARNING: Integration tests will TRUNCATE tables 'players', 'transactions', and 'invitations'.${NC}"
echo -e "${YELLOW}Press Ctrl+C to abort, or wait 3 seconds to continue...${NC}"
sleep 3

# Run tests
echo -e "${BLUE}Running Go Tests...${NC}"
cd "$PROJECT_ROOT/server"
# Use the test database
export DATABASE_URL="postgres://postgres:password@localhost:5432/squash_ladder_test?sslmode=disable" 
go test ./... -v

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed${NC}"
else
    echo -e "${RED}✗ Tests failed${NC}"
    exit 1
fi
