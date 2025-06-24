# Build stage
FROM golang:1.22 AS builder
WORKDIR /app

COPY . .
RUN go mod tidy

# Copy views and css directories from the repository root into the builder stage
# These paths are relative to the Dockerfile's location if the build context is services/frontend_renderer
# However, to access files outside this context (repo root), the build context for docker build
# command would need to be the repo root, and then these paths adjusted.
# For docker-compose, the context is services/frontend_renderer.
# A common pattern is to copy these assets into the service directory before build,
# or handle this in a more global build script.

# Let's assume for now that the docker-compose build context will be the root,
# and these paths are relative to that.
# If context is ./services/frontend_renderer, then these need to be:
# COPY ../../views ./views
# COPY ../../css ./css
# This is tricky with Docker contexts.

# Simpler builder stage: Copy them from the context (which will be services/frontend_renderer)
# This means we'll need to copy views/ and css/ into services/frontend_renderer/ before building.
# Let's stick to this for now and adjust later if needed.
# The Dockerfile itself will expect them to be in its context.

# Force static build for amd64 architecture
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o frontend_renderer_service ./main.go

# Final image
FROM debian:bullseye-slim
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/frontend_renderer_service .

# Copy views and css. These must exist in the build context of this Dockerfile.
# This means we need to ensure views/ and css/ are copied into services/frontend_renderer/
# before this Dockerfile is built.
COPY views/ ./views/
COPY css/ ./css/

EXPOSE 3005
CMD ["./frontend_renderer_service"]
