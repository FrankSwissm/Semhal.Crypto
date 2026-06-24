# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o semhal-crypto .

# Stage 2: Create a minimal production image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/semhal-crypto .
# Expose the port your Go app uses (usually 8080)
EXPOSE 8080
CMD ["./semhal-crypto"]

