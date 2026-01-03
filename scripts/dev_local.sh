#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
CLEAN_DB=false
for arg in "$@"; do
    if [ "$arg" == "--clean" ]; then
        CLEAN_DB=true
    fi
done

echo -e "${BLUE}=== Starting Local Development ===${NC}"
if [ "$CLEAN_DB" = true ]; then
    echo -e "${YELLOW}Mode: Clean start (Database will be reset)${NC}"
fi

# 1. Check prerequisites
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        echo "Please install $1 and try again"
        exit 1
    fi
}

echo -e "${YELLOW}[1/4] Checking prerequisites...${NC}"
check_command go
check_command node
check_command npm
check_command bazel
echo -e "${GREEN}  ✓ Prerequisites met${NC}"

# 2. Install Dependencies
echo -e "\n${YELLOW}[2/4] Checking dependencies...${NC}"

# Go dependencies
echo -e "  Checking Go dependencies..."
cd "$PROJECT_ROOT/server"
go mod download
echo -e "${GREEN}  ✓ Go dependencies satisfied${NC}"

# Node dependencies
echo -e "  Checking Node.js dependencies..."
cd "$PROJECT_ROOT/client"
if [ ! -d "node_modules" ]; then
    echo -e "  Installing Node modules..."
    npm install
    echo -e "${GREEN}  ✓ Node.js dependencies installed${NC}"
else
    echo -e "${GREEN}  ✓ Node.js dependencies already present${NC}"
fi

# 3. Generate Protos
echo -e "\n${YELLOW}[3/4] Generating proto files...${NC}"
"$SCRIPT_DIR/gen_protos.sh"

# 4. Start Servers
echo -e "\n${BLUE}[4/4] Starting servers...${NC}"

# Start Postgres
echo -e "Starting PostgreSQL..."
docker-compose up -d db

# Wait for DB to be ready
until docker exec squash_ladder_db pg_isready -U postgres > /dev/null 2>&1; do
  echo -n "."
  sleep 1
done
echo ""

if [ "$CLEAN_DB" = true ]; then
    echo -e "${YELLOW}Cleaning database...${NC}"
    # Terminate existing connections first
    docker exec squash_ladder_db psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'squash_ladder' AND pid <> pg_backend_pid();" > /dev/null 2>&1 || true
    # Drop and recreate
    docker exec squash_ladder_db dropdb -U postgres --if-exists squash_ladder
    docker exec squash_ladder_db createdb -U postgres squash_ladder
    echo -e "${GREEN}  ✓ Database reset${NC}"
fi

echo -e "${GREEN}  ✓ PostgreSQL started${NC}"

# Cleanup function
cleanup() {
    echo -e "\n${BLUE}Stopping servers...${NC}"
    kill $(jobs -p) 2>/dev/null
}
trap cleanup EXIT

# Start Go Server (using Bazel for dependency management)
echo -e "Starting Go Server (logs: /tmp/squash-ladder-server.log)..."
cd "$PROJECT_ROOT"
DATABASE_URL="postgres://postgres:password@localhost:5432/squash_ladder?sslmode=disable" bazel run //server:server > /tmp/squash-ladder-server.log 2>&1 &
SERVER_PID=$!

# Wait for server to be somewhat ready (not perfect check, but good UX)
sleep 2
if ps -p $SERVER_PID > /dev/null; then
   echo -e "${GREEN}  ✓ Server started${NC}"
else
   echo -e "${RED}  ✗ Server failed to start. Check /tmp/squash-ladder-server.log${NC}"
   exit 1
fi

# Start Vite Client
echo -e "Starting Vite Client..."
cd "$PROJECT_ROOT/client"
npm run dev

# Note: npm run dev runs in foreground, so script waits here.
# When user hits Ctrl+C, trap cleanup triggers.
