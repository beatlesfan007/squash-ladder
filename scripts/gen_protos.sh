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

bazel build //client/src/grpc:players_proto_ts //client/src/grpc:players_grpc_web
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Proto files generated${NC}"
    
    # Copy generated files to client source
    echo -e "  ℹ Copying generated files to client/src/grpc/..."
    mkdir -p client/src/grpc
    cp -f bazel-bin/client/src/grpc/players_proto_ts_pb/server/proto/players_pb.d.ts client/src/grpc/
    cp -f bazel-bin/client/src/grpc/players_proto_ts_pb/server/proto/players_pb.js client/src/grpc/
    cp -f bazel-bin/client/src/grpc/players_grpc_web_pb/server/proto/players_grpc_web_pb.d.ts client/src/grpc/
    cp -f bazel-bin/client/src/grpc/players_grpc_web_pb/server/proto/players_grpc_web_pb.js client/src/grpc/
    
    # Make files writable
    chmod +w client/src/grpc/*.js
    
    # Prepend 'var exports = {};' to make CJS compatible with browser
    echo "var exports = {};" | cat - client/src/grpc/players_pb.js > temp && mv temp client/src/grpc/players_pb.js
    echo "var exports = {};" | cat - client/src/grpc/players_grpc_web_pb.js > temp && mv temp client/src/grpc/players_grpc_web_pb.js

    # Fix: Clone proto.players if it's frozen
    sed -i '' 's/proto.players = require('\''\.\/players_pb.js'\'');/proto.players = require('\''\.\/players_pb.js'\''); if (!Object.isExtensible(proto.players)) { proto.players = Object.assign({}, proto.players); }/' client/src/grpc/players_grpc_web_pb.js

    # Fix: Replace readStringRequireUtf8
    sed -i '' 's/readStringRequireUtf8/readString/g' client/src/grpc/players_pb.js

    # Append ES exports
    echo "export const { Player, ListPlayersRequest, ListPlayersResponse } = proto.players;" >> client/src/grpc/players_pb.js
    echo "export const { PlayersServiceClient } = proto.players;" >> client/src/grpc/players_grpc_web_pb.js
    
    echo -e "${GREEN}  ✓ Generated files prepared${NC}"
else
    echo -e "${RED}  ✗ Failed to generate proto files${NC}"
    exit 1
fi
