FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o feednest ./cmd/feednest/

FROM alpine:3.23
RUN apk add --no-cache ca-certificates su-exec && \
    addgroup -S feednest && adduser -S feednest -G feednest
WORKDIR /app
COPY --from=builder /app/feednest .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh && mkdir -p /data && chown feednest:feednest /data

EXPOSE 8080
ENV DB_PATH=/data/feednest.db

ENTRYPOINT ["./entrypoint.sh"]
