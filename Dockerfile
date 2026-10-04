FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run check && npm run build

FROM golang:1.27.1-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /ride-archive ./cmd/api

FROM alpine:3.23
RUN apk add --no-cache ca-certificates su-exec \
    && addgroup -g 10001 archive && adduser -D -u 10001 -G archive archive
WORKDIR /app
COPY --from=api /ride-archive /usr/local/bin/ride-archive
COPY --from=web /src/web/dist ./web
COPY deploy/entrypoint.sh /usr/local/bin/entrypoint
ENV ARCHIVE_ADDR=0.0.0.0:8080 ARCHIVE_DB=/data/archive.sqlite3 ARCHIVE_WEB_DIR=/app/web
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/entrypoint"]
