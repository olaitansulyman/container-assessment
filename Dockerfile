# ---------- Build Stage ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install CA certs
RUN apk add --no-cache ca-certificates git

# Copy go mod files first (caching)
COPY application-code/go.mod application-code/go.sum ./
RUN go mod download

# Copy source code
COPY application-code/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o muchtodo-api .

# ---------- Runtime Stage ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Create non-root user
USER nonroot:nonroot

COPY --from=builder /app/muchtodo-api /app/muchtodo-api

EXPOSE 8080

CMD ["/app/muchtodo-api"]