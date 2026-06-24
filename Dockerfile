# 1. Build Stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o semhal-crypto main.go

# 2. Final Execution Stage
FROM alpine:latest
WORKDIR /root/
# Copy the built binary from the builder stage
COPY --from=builder /app/semhal-crypto .
# Expose the port your Go app listens on
EXPOSE 8085
# Run the binary
CMD ["./semhal-crypto"]
