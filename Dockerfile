FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o weather-station

FROM alpine:3.21.3
# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy the Go binary
COPY --from=builder /build/weather-station /app/
WORKDIR /app

RUN chmod +x weather-station

USER appuser
EXPOSE 8367
CMD ["./weather-station"]
