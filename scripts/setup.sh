#!/bin/bash
# Complete setup script for squash-ladder project
# Handles dependency installation, code generation, building, and running

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo -e "${GREEN}=== Squash Ladder Setup ===${NC}\n"

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        echo "Please install $1 and try again"
        exit 1
    fi
    echo -e "  ✓ $1 found"
}

check_command go
check_command node
check_command npm
check_command bazel

echo ""

# Step 1: Install Go dependencies
echo -e "${YELLOW}[1/5] Installing Go dependencies...${NC}"
cd "$PROJECT_ROOT/server"
go mod download
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Go dependencies installed${NC}"
else
    echo -e "${RED}  ✗ Failed to install Go dependencies${NC}"
    exit 1
fi
echo ""

# Step 2: Install Node.js dependencies
echo -e "${YELLOW}[2/5] Installing Node.js dependencies...${NC}"
cd "$PROJECT_ROOT/client"
if [ ! -d "node_modules" ]; then
    npm install
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓ Node.js dependencies installed${NC}"
    else
        echo -e "${RED}  ✗ Failed to install Node.js dependencies${NC}"
        exit 1
    fi
else
    echo -e "  ✓ Node.js dependencies already installed (skipping)"
fi
echo ""

# Step 3: Generate proto TypeScript/JavaScript files with Bazel
# Using rules_proto_grpc for proper TypeScript and gRPC-Web code generation
echo -e "${YELLOW}[3/5] Generating proto files for client with Bazel...${NC}"
cd "$PROJECT_ROOT"
bazel build //client/src/grpc:players_proto_ts //client/src/grpc:players_grpc_web
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Proto files generated (TypeScript + gRPC-Web)${NC}"
    
    # Copy generated files to client source so Vite can see them
    echo -e "  ℹ Copying generated files to client/src/grpc/..."
    cp -f bazel-bin/client/src/grpc/players_proto_ts_pb/server/proto/players_pb.d.ts client/src/grpc/
    cp -f bazel-bin/client/src/grpc/players_proto_ts_pb/server/proto/players_pb.js client/src/grpc/
    cp -f bazel-bin/client/src/grpc/players_grpc_web_pb/server/proto/players_grpc_web_pb.d.ts client/src/grpc/
    cp -f bazel-bin/client/src/grpc/players_grpc_web_pb/server/proto/players_grpc_web_pb.js client/src/grpc/
    
    # Make files writable
    chmod +w client/src/grpc/*.js
    
    # Prepend 'var exports = {};' to make CJS compatible with browser
    echo "var exports = {};" | cat - client/src/grpc/players_pb.js > temp && mv temp client/src/grpc/players_pb.js
    echo "var exports = {};" | cat - client/src/grpc/players_grpc_web_pb.js > temp && mv temp client/src/grpc/players_grpc_web_pb.js

    # Fix: Clone proto.players if it's frozen (from ESM import) so we can extend it
    # We use sed to insert the check after the require
    sed -i '' 's/proto.players = require('\''\.\/players_pb.js'\'');/proto.players = require('\''\.\/players_pb.js'\''); if (!Object.isExtensible(proto.players)) { proto.players = Object.assign({}, proto.players); }/' client/src/grpc/players_grpc_web_pb.js

    # Fix: Replace readStringRequireUtf8 with readString to match google-protobuf npm package
    sed -i '' 's/readStringRequireUtf8/readString/g' client/src/grpc/players_pb.js

    # Append ES exports to generated JS files for Vite compatibility
    echo "export const { Player, ListPlayersRequest, ListPlayersResponse } = proto.players;" >> client/src/grpc/players_pb.js
    echo "export const { PlayersServiceClient } = proto.players;" >> client/src/grpc/players_grpc_web_pb.js
    
    echo -e "${GREEN}  ✓ Generated files copied and patched for Vite${NC}"
else
    echo -e "${RED}  ✗ Failed to generate proto files${NC}"
    echo -e "  ℹ Proto generation is handled by rules_proto_grpc in Bazel"
    echo -e "  ℹ No manual protoc installation needed"
    exit 1
fi
echo ""

# Step 4: Build with Bazel (proto code is generated automatically)
echo -e "${YELLOW}[4/5] Building with Bazel...${NC}"
cd "$PROJECT_ROOT"
bazel build //server:server
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Server built successfully${NC}"
else
    echo -e "${RED}  ✗ Failed to build server${NC}"
    exit 1
fi

# Note: Client build with Bazel is optional since we use npm for development
echo -e "  ℹ Client will be built/run with npm (Bazel build optional)"
echo ""

# Step 5: Run applications
echo -e "${YELLOW}[5/5] Starting applications...${NC}"
echo -e "${GREEN}Setup complete!${NC}\n"

# Ask if user wants to run the applications
read -p "Do you want to start the server and client now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Starting server in background...${NC}"
    cd "$PROJECT_ROOT"
    bazel run //server:server > /tmp/squash-ladder-server.log 2>&1 &
    SERVER_PID=$!
    echo -e "${GREEN}  ✓ Server started (PID: $SERVER_PID)${NC}"
    echo -e "  ℹ Server logs: /tmp/squash-ladder-server.log"
    echo -e "  ℹ Server URL: http://localhost:8080"
    
    # Wait a moment for server to start
    sleep 2
    
    echo -e "${YELLOW}Starting client...${NC}"
    cd "$PROJECT_ROOT/client"
    echo -e "${GREEN}  ✓ Client starting...${NC}"
    echo -e "  ℹ Client URL: http://localhost:3000"
    echo -e "  ℹ Press Ctrl+C to stop both server and client"
    echo ""
    
    # Trap to cleanup server on exit
    trap "kill $SERVER_PID 2>/dev/null; exit" INT TERM
    
    # Run client in foreground
    npm run dev
    
    # Cleanup
    kill $SERVER_PID 2>/dev/null
else
    echo ""
    echo -e "${GREEN}To run the applications manually:${NC}"
    echo -e "  Server: ${YELLOW}cd server && bazel run //server:server${NC}"
    echo -e "  Client: ${YELLOW}cd client && npm run dev${NC}"
    echo ""
fi

