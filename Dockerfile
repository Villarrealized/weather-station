# Stage 1: Dependencies
FROM golang:1.24-alpine AS deps

WORKDIR /build
RUN apk add --no-cache gcc musl-dev

# Copy mod files and download deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Precompile heavy CGO dependency (cached)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install github.com/mattn/go-sqlite3

# Stage 2 - builder
FROM deps AS builder
# Copy source and build binary
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 \
    go build -tags "linux" -ldflags="-s -w" -trimpath -o weather-station

# Stage 3: Final image
FROM alpine:3.21.3

# Install timezone data
RUN apk add --no-cache tzdata
ENV TZ=America/Denver
RUN cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app
COPY --from=builder /build/weather-station /app/weather-station
RUN mkdir -p /app/data && chmod +x /app/weather-station

EXPOSE 8367
CMD ["/app/weather-station"]
