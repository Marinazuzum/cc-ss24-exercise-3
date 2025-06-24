# Build stage
FROM golang:1.22 as builder
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download
RUN go mod tidy # Though download should suffice, tidy ensures consistency

COPY . .

# Force static build for amd64 architecture
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api_get_books_service ./main.go

# Final image
FROM debian:bullseye-slim
WORKDIR /app

# Copy only the built binary from the builder stage
COPY --from=builder /app/api_get_books_service .

# No views or css needed for this API service

EXPOSE 3001 # Port defined in its main.go
CMD ["./api_get_books_service"]
