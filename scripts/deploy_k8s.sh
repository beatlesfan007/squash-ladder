#!/bin/bash
# Complete setup script for squash-ladder project (Kubernetes)
# Handles code generation, container building, and deployment

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

echo -e "${GREEN}=== Squash Ladder Kubernetes Setup ===${NC}\n"

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

check_command bazel
check_command docker
check_command kubectl

echo ""

# Step 1: Generate proto TypeScript/JavaScript files
# We need these files locally so they can be copied into the client Docker image
echo -e "${YELLOW}[1/4] Generating proto files...${NC}"
"$SCRIPT_DIR/gen_protos.sh"
if [ $? -ne 0 ]; then
    echo -e "${RED}  ✗ Failed to generate proto files${NC}"
    exit 1
fi
echo ""

# Step 2: Build Docker Images
echo -e "${YELLOW}[2/4] Building Docker images...${NC}"

echo -e "  Building server image..."
docker build --tag squash-ladder-server:latest -f server/Dockerfile .

echo -e "  Building client image..."
docker build --tag squash-ladder-client:latest -f client/Dockerfile client/

echo -e "${GREEN}  ✓ Docker images built${NC}"
echo ""

# Step 3: Deploy to Kubernetes
echo -e "${YELLOW}[3/4] Deploying to Kubernetes...${NC}"

# Check if k8s cluster is reachable
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Error: Cannot connect to Kubernetes cluster.${NC}"
    echo "Please ensure your cluster (Docker Desktop, Minikube, etc.) is running."
    exit 1
fi

kubectl apply -f k8s/
if [ $? -eq 0 ]; then
    echo -e "${GREEN}  ✓ Manifests applied${NC}"
else
    echo -e "${RED}  ✗ Failed to apply manifests${NC}"
    exit 1
fi
echo ""

# Step 4: Verification
echo -e "${YELLOW}[4/4] Verifying deployment...${NC}"
echo -e "  Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod --selector=app=server --timeout=60s
kubectl wait --for=condition=ready pod --selector=app=client --timeout=60s

echo ""
echo -e "${GREEN}=== Setup Complete ===${NC}"
echo -e "Access your application at:"
echo -e "  ${YELLOW}http://localhost${NC} (LoadBalancer/Ingress)"
echo -e "  Or check 'minikube service client-service' if using Minikube."
echo ""


