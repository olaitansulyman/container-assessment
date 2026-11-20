# syntax=docker/dockerfile:1

# BuildKit supplies TARGETOS/TARGETARCH for the requested build platform.
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY application-code/go.mod application-code/go.sum ./
RUN go mod download

COPY application-code/ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/muchtodo-api .

# Minimal non-root runtime. The application exposes /health on port 8080.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder --chown=nonroot:nonroot /out/muchtodo-api /app/muchtodo-api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/muchtodo-api"]
