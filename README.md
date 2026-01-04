# Squash Ladder

A web application for managing a squash ladder where players can view rankings and log matches.

## Architecture

- **Backend**: gRPC server (Go) with gRPC-Web support - serves player ranking APIs via Protocol Buffers. Persists data to PostgreSQL.
- **Frontend**: React TypeScript application - displays player rankings using gRPC-Web client
- **Build System**: Bazel for unified builds with automatic proto code generation using `rules_proto_grpc`
- **Auth & Database**: Supabase (Self-hosted via Docker) - provides Authentication (GoTrue) and PostgreSQL database.

## Prerequisites

- Go 1.23 or later
- Node.js 20.x or later
- Bazel 6.0 or later
- Docker Desktop with Kubernetes enabled (Settings > Kubernetes > Enable Kubernetes)
- Protocol Buffers: Proto code generation is handled automatically by Bazel
- **Supabase Project** (or self-hosted instance): Required for Authentication
  - You must provide the `SUPABASE_JWT_SECRET` to the server
  - You must provide `VITE_SUPABASE_URL` and `VITE_SUPABASE_ANON_KEY` to the client

## Authentication Setup

The application uses **Supabase Authentication** to secure access.

### 1. Environment Variables
Add the following to your environment (e.g., in `.env` or exported in your shell):

**Server**:
- `SUPABASE_JWT_SECRET`: The JWT secret from your Supabase instance.

**Client**:
- `VITE_SUPABASE_URL`: URL of your Supabase API.
- `VITE_SUPABASE_ANON_KEY`: Anonymous public key.

### 2. Initial Setup (Bootstrap)
1. **Create Admin**: Manually sign up the first user in your Supabase instance.
2. **Login**: Access the application and log in via the Magic Link sent to your email.
3. **Populate Ladder**: The first authenticated user can access the "Add Player" form to populate the ladder.
4. **Invite Players**: Adding a player generates a unique **Invitation Link**. Share this link with the new player to let them claim their profile.

## Development Workflows

We support two distinct workflows:

### 1. Local Iterative Development (Recommended)

Fast feedback loop using a local Go process and Vite dev server.

```bash
./scripts/dev_local.sh
```

This script:
1. Generates proto files (`scripts/gen_protos.sh`).
2. Starts the **Supabase Stack** (Auth, DB, Realtime, Dashboard, etc.) via Docker Compose.
3. Starts the Go server via Bazel (`bazel run //server:server`).
   - *Note*: It automatically loads credentials from the local `.env` file.
4. Starts the Vite client (`npm run dev`).
5. Cleans up processes on exit (Ctrl+C).

> **Note**: The first time you run this, it will pull several Docker images for Supabase, which may take a few minutes.

#### Supabase Dashboard
When running locally, you can access the Supabase Dashboard at `http://localhost:3000` to manage users and inspect the database. Default credentials are in `infrastructure/supabase/.env` (usually `supabase` / `this_password_is_insecure_and_should_be_updated`).

### 2. Running Verification Tests

To run the full suite of tests including integration tests (requires Postgres):

```bash
./scripts/test_integration.sh
```

This script handles starting the database, waiting for it to be ready, and running `go test`.

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
  - **Auth Required**: Request must include `Authorization: Bearer <token>` metadata
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
- [x] **Authentication**: Add Supabase authentication for secure player login
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
