# syntax=docker/dockerfile:1

# Multi-stage Bun + Next.js Dockerfile (standalone output)

FROM oven/bun:1.2 AS builder
WORKDIR /app

# Only copy the manifests first for better layer caching
COPY package.json ./
# If you have a bun.lock or bun.lockb, copy it for reproducible installs
COPY bun.lock ./
COPY bun.lockb* ./

# Toolchain for native modules needed by node-gyp (e.g., better-sqlite3)
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 make g++ \
    && rm -rf /var/lib/apt/lists/*

# Install only production deps (exclude dev)
ENV NODE_ENV=production
# Production install only (dev deps excluded). bun.lock is now up-to-date so pruning won't trigger freeze.
RUN CI=0 BUN_INSTALL_FROZEN_LOCKFILE=0 bun install --production --no-progress --ignore-scripts

# Copy the rest of the source
COPY . .

ENV NEXT_TELEMETRY_DISABLED=1

# Build Next.js app (standalone)
RUN bun run build


FROM oven/bun:1.2 AS runner
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

# Copy standalone server and static assets
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000

# Start the standalone server with Bun's Node-compatible runtime
CMD ["bun", "server.js"]
