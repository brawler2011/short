# ── Stage 1: Build Frontend ──────────────────────────────────────────
FROM oven/bun:1-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock* ./
RUN bun install
COPY frontend/ ./
RUN bun run build

# ── Stage 2: Build Go Backend ─────────────────────────────────────────
FROM golang:1.27-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/bin/short .

# ── Stage 3: Minimal Runtime ──────────────────────────────────────────
FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
RUN mkdir -p /data && chown -R app:app /data /app
USER app

COPY --from=backend-builder /app/bin/short /app/short

ENV PORT=8080
ENV DB_PATH=/data/short.db
EXPOSE 8080

ENTRYPOINT ["/app/short"]
