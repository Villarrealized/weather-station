FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install build dependencies only once, cache this layer
RUN apk add --no-cache gcc musl-dev

# Copy only go.mod and go.sum first, so dependency download is cached
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code after dependencies are cached
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags "linux" -ldflags="-s -w" -trimpath -o weather-station

# Final
FROM alpine:3.21.3
RUN apk add --no-cache tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app

COPY --from=builder /build/weather-station /app/weather-station
RUN chmod +x /app/weather-station && chown -R appuser:appgroup /app
USER appuser

EXPOSE 8367
CMD ["/app/weather-station"]
