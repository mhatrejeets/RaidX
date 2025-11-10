# Build stage
FROM golang:1.24.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o raidx-server .

# Final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/raidx-server .
COPY ./Static ./Static
COPY ./views ./views
# Copy .env if it exists (optional for Docker, will use environment variables)
COPY .env* ./
EXPOSE 3000
CMD ["./raidx-server"]
