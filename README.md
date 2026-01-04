# Squash Ladder

A web application for managing a squash ladder where players can view rankings and log matches.

## Architecture

- **Backend**: gRPC server (Go) with gRPC-Web support - serves player ranking APIs via Protocol Buffers. Persists data to Supabase Cloud (PostgreSQL).
- **Frontend**: React TypeScript application - displays player rankings using gRPC-Web client and Supabase SDK for authentication.
- **Build System**: Bazel for unified builds with automatic proto code generation using `rules_proto_grpc`

## Prerequisites

- Go 1.23 or later
- Node.js 20.x or later
- Bazel 6.0 or later
- Docker Desktop with Kubernetes enabled (Settings > Kubernetes > Enable Kubernetes)
- Protocol Buffers: Proto code generation is handled automatically by Bazel

## Supabase Cloud Setup

This project requires a [Supabase Cloud](https://supabase.com/) project for authentication and database storage.

### 1. Create a Supabase Project
- Sign up at [supabase.com](https://supabase.com/) and create a new project.
- [Getting Started Guide](https://supabase.com/docs/guides/getting-started)

### 2. Database Configuration
- Obtain your `DATABASE_URL` from **Project Settings > Database > Connection string > URI**.
- [Database Connection Docs](https://supabase.com/docs/guides/database/connecting-to-postgres)

### 3. Authentication Configuration
- Enable **Magic Links** under **Authentication > Providers > Email**.
- [Magic Link Auth Docs](https://supabase.com/docs/guides/auth/auth-email)

### 4. API & Secret Keys
- **Server Keys**: Get your `SUPABASE_JWT_SECRET` from **Project Settings > API > JWT Secret**.
- **Client Keys**: Get your `VITE_SUPABASE_URL` and `VITE_SUPABASE_ANON_KEY` from **Project Settings > API**.
- [API Key Docs](https://supabase.com/docs/guides/api/api-keys)

## Environment Setup

1. **Server**: Copy `.env.example` to `.env` and fill in:
   - `DATABASE_URL`
   - `SUPABASE_JWT_SECRET`

2. **Client**: Copy `client/.env.example` to `client/.env` and fill in:
   - `VITE_SUPABASE_URL`
   - `VITE_SUPABASE_ANON_KEY`

## Development Workflows

We support two distinct workflows:

### 1. Local Iterative Development (Recommended)

Fast feedback loop using a local Go process and Vite dev server.

```bash
./scripts/dev_local.sh
```

This script:
1. Generates proto files (`scripts/gen_protos.sh`).
2. Loads environment variables from `.env`.
3. Starts the Go server via Bazel (`bazel run //server:server`).
4. Starts the Vite client (`npm run dev`).

**Note**: You must have a Supabase Cloud project configured. See [Supabase Cloud Setup](#supabase-cloud-setup) for details.
### 2. Running Verification Tests

To run the full suite of tests including integration tests (requires Postgres):

```bash
./scripts/test_integration.sh
```

This script runs the Go integration tests against the database specified in your `.env` file.

### 3. Kubernetes Deployment

Production-like environment using Docker and Kubernetes.

```bash
./scripts/deploy_k8s.sh
```

This script:
1. Generates proto files.
2. Builds Docker images for Server and Client.
3. Deploys to the configured Kubernetes cluster.

## Manual Setup

If you prefer to set up manually:

### 1. Build Docker Images

```bash
# Build Server
docker buildx build -t squash-ladder-server:latest -f server/Dockerfile .

# Build Client
# Note: You must generate proto files first (run ./scripts/gen_protos.sh) if not present
docker buildx build -t squash-ladder-client:latest -f client/Dockerfile client/
```

### 2. Deploy to Kubernetes

```bash
kubectl apply -f k8s/
```

### 3. Testing with Database

The backend tests (`server:server_test`) require a PostgreSQL database.

**Option 1: Automatic (Recommended)**
Ensure Docker is running. The tests will automatically spin up a temporary PostgreSQL container using [Testcontainers](https://golang.testcontainers.org/).

```bash
bazel test //server:server_test
```

**Option 2: Manual (Faster)**
Available if you want to reuse an existing database instance or debug the database state.

1. Start Postgres:
```bash
docker run --rm -d --name squash-ladder-test-db \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=squash_ladder_test \
  -p 5432:5432 postgres:15
```

2. Run Tests with `DATABASE_URL`:
```bash
bazel test //server:server_test \
  --action_env=DATABASE_URL="postgres://postgres:password@localhost:5432/squash_ladder_test?sslmode=disable"
```

3. Cleanup:
```bash
docker stop squash-ladder-test-db
```

## Running the Application

Once deployed, the application will be available at:

- **Client**: `http://localhost` (or your cluster IP)
- **Server**: Accessed internally by the client via the Nginx proxy

### Verifying Deployment

```bash
kubectl get pods
kubectl get services
```


## API Endpoints

### gRPC-Web Service

- **Service**: `players.PlayersService`
- **Method**: `ListPlayers(ListPlayersRequest) returns (ListPlayersResponse)`
  - Returns a list of all players ordered by rank
  - Uses gRPC-Web protocol for browser compatibility
  - Service path: `/players.PlayersService/ListPlayers`

### REST Fallback (JSON)

- `GET /api/players` - Returns a JSON list of all players ordered by rank
  - Provided for compatibility, but the client uses gRPC-Web by default

## Project Structure

```
squash-ladder/
├── MODULE.bazel           # Bazel module configuration (Bazel 6+ uses Bzlmod)
├── BUILD                  # Root BUILD file
├── server/                # gRPC server
│   ├── proto/            # Protocol Buffer definitions
│   ├── handlers/         # gRPC service handlers
│   ├── cmd/server/       # Server entry point
│   └── BUILD             # Bazel build rules (proto code generated automatically)
├── client/               # React TypeScript frontend
│   ├── src/              # React source code
│   │   └── grpc/         # gRPC-Web client code (proto code generated by Bazel)
│   ├── public/           # Static assets
│   └── BUILD             # Bazel build rules (uses rules_proto_grpc)
└── scripts/              # Utility scripts
```

## Production Roadmap

### Security
- [x] **Authentication**: Integrated Supabase Cloud for secure player login (Magic Link)
- [ ] **Authorization**: Implement Role-Based Access Control (RBAC) with **Admin** (manage players/invites) and **User** (log matches) roles
- [ ] **Secret Management**: Move credentials from plain text YAML to Kubernetes Secrets
- [ ] **TLS/SSL**: Enable SSL for database connections and secure ingress for the web client

### Features & Workflow
- [x] **Player Invites**: Mechanism to generate unique invite links for new players to link their account to a ladder profile

### Infrastructure
- [ ] **Persistent Storage**: Update Postgres deployment to use PersistentVolumeClaims (PVC) instead of `emptyDir`
- [ ] **Resource Management**: Define CPU/Memory requests and limits in Kubernetes manifests
- [ ] **Health Checks**: Implement Liveness and Readiness probes
- [ ] **Ingress**: Configure an Ingress Controller with cert-manager for HTTPS

### DevOps & CI/CD
- [ ] **CI Pipeline**: Set up GitHub Actions for automated testing and linting
- [ ] **CD Pipeline**: Automate Docker image building and deployment
- [ ] **Versioning**: Use Git SHA tagging for Docker images instead of `latest`

### Database
- [ ] **Migrations**: Implement a proper migration tool (e.g., `golang-migrate`) instead of `IF NOT EXISTS` checks
- [ ] **Backup**: Establish a database backup and restore strategy

### Observability
- [ ] **Structured Logging**: Replace standard logging with structured JSON logging (e.g., `zap` or `slog`)
- [ ] **Metrics**: Expose Prometheus metrics for monitoring server performance
