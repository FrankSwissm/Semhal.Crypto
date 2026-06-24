# Stage 1: Build
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o semhal-crypto .

# Stage 2: Final image
FROM alpine:latest
WORKDIR /root/
# Copy the binary
COPY --from=builder /app/semhal-crypto .
# COPY the folders explicitly!
COPY templates/ ./templates/
COPY static/ ./static/

EXPOSE 8085
CMD ["./semhal-crypto"]
