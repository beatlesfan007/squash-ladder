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

# Ensure DB is up
echo -e "${BLUE}Starting PostgreSQL...${NC}"
cd "$PROJECT_ROOT"
docker-compose up -d db

# Wait for DB to be ready
echo -e "${BLUE}Waiting for Database to be ready...${NC}"
until docker exec squash_ladder_db pg_isready -U postgres > /dev/null 2>&1; do
  echo -n "."
  sleep 1
done
echo ""

# Create Test Database (idempotent)
echo -e "${BLUE}Ensuring test database exists...${NC}"
docker exec squash_ladder_db psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'squash_ladder_test'" | grep -q 1 || \
docker exec squash_ladder_db psql -U postgres -c "CREATE DATABASE squash_ladder_test"

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
