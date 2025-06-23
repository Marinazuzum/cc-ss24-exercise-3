# Build stage
FROM golang:1.22 as builder
WORKDIR /app
COPY . .
RUN go mod tidy
# Force static build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/main.go

# Final image
FROM debian:bullseye-slim
WORKDIR /app
COPY --from=builder /app/app .
COPY views/ ./views/
COPY css/ ./css/
EXPOSE 3000
CMD ["./app"]