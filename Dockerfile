FROM golang:1.22-alpine AS build

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.20

WORKDIR /app

ENV PACK_CALCULATOR_CONFIG=/app/config/config.json
ENV PORT=8080

RUN addgroup -S app && \
    adduser -S -G app app && \
    mkdir -p /app/config /app/data && \
    chown -R app:app /app

COPY --from=build /out/api /app/api
COPY backend/config/config.json /app/config/config.json

RUN chown -R app:app /app

USER app

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/healthz" >/dev/null || exit 1

CMD ["/app/api"]
