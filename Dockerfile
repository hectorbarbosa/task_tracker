# ─── Build stage ─────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /build/task_tracker ./cmd/api

# ─── Migrate tool stage ──────────────────────────────────────────────────────
FROM golang:1.23-alpine AS migrate-builder

RUN go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ─── Runtime stage ───────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/task_tracker /usr/local/bin/task_tracker
COPY --from=migrate-builder /go/bin/migrate /usr/local/bin/migrate
COPY migrations /migrations

EXPOSE 8080

ENTRYPOINT ["task_tracker"]
