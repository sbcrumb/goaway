FROM node:22-alpine AS frontend
WORKDIR /app
RUN npm install -g pnpm
COPY client/package.json client/pnpm-lock.yaml ./client/
RUN pnpm -C client install --frozen-lockfile
COPY client ./client
RUN pnpm -C client build

FROM golang:1.26.0-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/client/dist ./client/dist
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -o goaway .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/goaway ./goaway
EXPOSE 53/udp 53/tcp 8080/tcp
ENTRYPOINT ["./goaway"]
