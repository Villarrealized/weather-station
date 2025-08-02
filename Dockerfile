FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o weather-station

FROM alpine:3.21.3

RUN apk add --no-cache tzdata

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /build/weather-station .

RUN chmod +x weather-station \
    && chown appuser:appgroup weather-station

USER appuser
EXPOSE 8367
CMD ["./weather-station"]
