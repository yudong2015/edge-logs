# Multi-stage Docker build for edge-logs frontend
# Stage 1: Build stage
FROM node:18-alpine AS builder

# Install pnpm
RUN npm install -g pnpm

# Set working directory
WORKDIR /app

# Copy package files
COPY package.json pnpm-lock.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source files
COPY . .

# Build application
RUN pnpm run build

# Stage 2: Production stage with nginx
FROM nginx:alpine

# Install curl for health check
RUN apk add --no-cache curl

# Copy nginx config as template for envsubst processing
# nginx:alpine's entrypoint will process .template files in /etc/nginx/templates/
COPY nginx.conf /etc/nginx/templates/default.conf.template

# Copy built assets from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Set default environment variables
# API_SERVER_HOST will be substituted by envsubst at container startup
ENV API_SERVER_HOST=edge-logs-apiserver.logging-system.svc.cluster.local:8080

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/healthz || exit 1

# Start nginx (uses default entrypoint which runs envsubst)
CMD ["nginx", "-g", "daemon off;"]
