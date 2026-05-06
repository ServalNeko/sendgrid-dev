# syntax=docker/dockerfile:1

# ── ビルドステージ ───────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o sendgrid-dev .

# ── ランタイムステージ ────────────────────────────────────────────
FROM node:22-alpine

# tini: シグナル転送とゾンビプロセス回収を担う PID 1
RUN apk add --no-cache tini

# maildev インストール
RUN npm install -g maildev --omit=dev && npm cache clean --force

WORKDIR /app

# Go バイナリをコピー
COPY --from=builder /app/sendgrid-dev .

# 起動スクリプトをインライン定義
RUN <<'EOF' tee /entrypoint.sh
#!/bin/sh
set -e
echo "[maildev] SMTP :${MAILDEV_SMTP_PORT:-1025}  Web UI :${MAILDEV_WEB_PORT:-1080}"
maildev \
  --smtp "${MAILDEV_SMTP_PORT:-1025}" \
  --web  "${MAILDEV_WEB_PORT:-1080}"  \
  --no-open &
echo "[sendgrid-dev] ${SENDGRID_DEV_API_SERVER:-:3030}"
exec /app/sendgrid-dev
EOF
RUN chmod +x /entrypoint.sh

# sendgrid-dev API | maildev SMTP | maildev Web UI
EXPOSE 3030 1025 1080

# デフォルト値（-e または compose.yml で上書き可）
ENV SENDGRID_DEV_API_SERVER=":3030" \
    SENDGRID_DEV_API_KEY="SG.xxxxx" \
    SENDGRID_DEV_SMTP_SERVER="127.0.0.1:1025"

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
