# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o crossplane-mcp .

# ── Runtime stage (distroless for minimal attack surface) ─────────────────────
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/crossplane-mcp /crossplane-mcp

# stdio by default; set MCP_TRANSPORT=sse + MCP_ADDR=:8080 for HTTP
ENTRYPOINT ["/crossplane-mcp"]
