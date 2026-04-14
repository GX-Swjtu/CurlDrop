# 构建阶段
FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o curldrop .

# 运行阶段
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/curldrop /app/curldrop

# 默认存储目录
VOLUME ["/data"]

# 默认端口
EXPOSE 8080 8443

ENTRYPOINT ["/app/curldrop"]
CMD ["-storage", "/data"]
