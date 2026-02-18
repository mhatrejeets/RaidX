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
COPY ./scripts ./scripts
COPY ./dummy_data ./dummy_data
COPY .env .env
RUN chmod +x /app/scripts/seed.sh
EXPOSE 3000
CMD ["./raidx-server"]
