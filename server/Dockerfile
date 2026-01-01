FROM ubuntu:22.04 AS builder

# Install dependencies for Bazelisk
RUN apt-get update && apt-get install -y \
    curl \
    git \
    g++ \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Install Bazelisk
RUN ARCH=$(dpkg --print-architecture) && \
    curl -L -o /usr/local/bin/bazel "https://github.com/bazelbuild/bazelisk/releases/download/v1.19.0/bazelisk-linux-${ARCH}" \
    && chmod +x /usr/local/bin/bazel

WORKDIR /workspace

# Copy the entire workspace first to ensure all dependencies can be resolved
# We need the WORKSPACE / MODULE.bazel files at the root
COPY . .

# Build the server binary
# We are currently in /workspace. The server code is in /workspace/server.
# The target is //server:server
WORKDIR /workspace
RUN bazel build //server:server

# Extract the binary
# The output location depends on the bazel configuration, but typically it's inside bazel-bin
RUN cp bazel-bin/server/server_/server /server-binary

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /server-binary /app/server

EXPOSE 8080

CMD ["/app/server"]
