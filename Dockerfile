# ---- build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod first for better layer caching. There are no external
# dependencies (standard library only), so no `go mod download` is needed.
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /ticket-system .

# ---- run stage ----
FROM alpine:3.20

RUN adduser -D -H appuser
WORKDIR /app
COPY --from=builder /ticket-system /app/ticket-system

USER appuser
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/ticket-system"]
