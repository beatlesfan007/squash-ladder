#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${YELLOW}Generating proto files for client...${NC}"
cd "$PROJECT_ROOT"

# Check bazel
if ! command -v bazel &> /dev/null; then
    echo -e "${RED}Error: bazel is not installed${NC}"
    exit 1
fi

bazel build //client/src/grpc:ladder_proto_ts //client/src/grpc:ladder_grpc_web
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Proto files generated${NC}"
    
    # Copy generated files to client source
    echo -e "  ℹ Copying generated files to client/src/grpc/..."
    mkdir -p client/src/grpc
    cp -f bazel-bin/client/src/grpc/ladder_proto_ts_pb/server/proto/ladder_pb.d.ts client/src/grpc/
    cp -f bazel-bin/client/src/grpc/ladder_proto_ts_pb/server/proto/ladder_pb.js client/src/grpc/
    cp -f bazel-bin/client/src/grpc/ladder_grpc_web_pb/server/proto/ladder_grpc_web_pb.d.ts client/src/grpc/
    cp -f bazel-bin/client/src/grpc/ladder_grpc_web_pb/server/proto/ladder_grpc_web_pb.js client/src/grpc/
    
    # Make files writable
    chmod +w client/src/grpc/*.js
    
    # Prepend 'var exports = {};' to make CJS compatible with browser
    echo "var exports = {};" | cat - client/src/grpc/ladder_pb.js > temp && mv temp client/src/grpc/ladder_pb.js
    echo "var exports = {};" | cat - client/src/grpc/ladder_grpc_web_pb.js > temp && mv temp client/src/grpc/ladder_grpc_web_pb.js

    # Fix: Clone proto.ladder if it's frozen
    sed -i '' 's/proto.ladder = require('\''\.\/ladder_pb.js'\'');/proto.ladder = require('\''\.\/ladder_pb.js'\''); if (!Object.isExtensible(proto.ladder)) { proto.ladder = Object.assign({}, proto.ladder); }/' client/src/grpc/ladder_grpc_web_pb.js

    # Fix: Replace readStringRequireUtf8 (incompatibility between some versions of protobuf)
    sed -i '' 's/readStringRequireUtf8/readString/g' client/src/grpc/ladder_pb.js

    # Append ES exports
    echo "export const { Player, ListPlayersRequest, ListPlayersResponse, AddPlayerRequest, AddPlayerResponse, RemovePlayerRequest, RemovePlayerResponse, MatchResult, AddMatchResultRequest, AddMatchResultResponse, InvalidateMatchResultRequest, InvalidateMatchResultResponse, ListRecentMatchesRequest, ListRecentMatchesResponse } = proto.ladder;" >> client/src/grpc/ladder_pb.js
    echo "export const { LadderServiceClient } = proto.ladder;" >> client/src/grpc/ladder_grpc_web_pb.js
    
    echo -e "${GREEN}  ✓ Generated files prepared${NC}"

    # Generate Server Go files
    echo -e "${YELLOW}Generating proto files for server...${NC}"
    bazel build //server/proto:ladder_go_proto
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓ Server proto files generated${NC}"
        
        # Determine the source path in bazel-bin dynamically
        # Find the directory containing ladder.pb.go
        SRC_FILE=$(find bazel-bin/server/proto -name "ladder.pb.go" | head -n 1)
        if [ -z "$SRC_FILE" ]; then
             echo -e "${RED}  Could not find ladder.pb.go in bazel-bin/server/proto${NC}"
             exit 1
        fi
        SRC_DIR=$(dirname "$SRC_FILE")
        
        echo -e "  ℹ Found generated files in $SRC_DIR"

        DEST_DIR="server/gen/ladder"
        mkdir -p "$DEST_DIR"
        
        # Copy Go files
        echo -e "  ℹ Copying generated Go files to $DEST_DIR/..."
        cp -f "$SRC_DIR"/*.pb.go "$DEST_DIR/"
        
        echo -e "${GREEN}  ✓ Server files updated${NC}"
    else
        echo -e "${RED}  ✗ Failed to generate server proto files${NC}"
        exit 1
    fi
else
    echo -e "${RED}  ✗ Failed to generate proto files${NC}"
    exit 1
fi
