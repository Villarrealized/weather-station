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
ENV TZ=America/Denver
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app

COPY --from=builder /build/weather-station /app/weather-station
RUN mkdir -p /app/data && chmod +x /app/weather-station

EXPOSE 8367
CMD ["/app/weather-station"]
