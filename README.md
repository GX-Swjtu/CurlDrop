# CurlDrop

[![CI](https://github.com/GaoXinLXL/CurlDrop/actions/workflows/ci.yml/badge.svg)](https://github.com/GaoXinLXL/CurlDrop/actions/workflows/ci.yml)

轻量级文件传输工具，专为 `curl` 设计，同时提供 Web 界面。单文件二进制，零依赖部署。

## 功能特性

- **curl 上传/下载** - 支持 `curl -F` 表单上传和 `curl -T` 流式上传
- **断点续传** - 上传和下载均支持 `curl -C -` 断点续传
- **超大文件** - 流式传输，内存占用极低，支持任意大小文件
- **Basic Auth** - 简单的用户名密码认证，保护上传和下载
- **HTTPS** - 支持自动生成自签名证书（内存中，不写磁盘），也支持指定证书文件
- **Web 界面** - 内置文件管理页面，支持拖拽上传、在线播放视频
- **视频流播放** - 支持 MP4/WebM/OGG 等格式的流媒体点播
- **自动清理** - 可配置自动删除过期文件
- **单文件部署** - 编译为单个二进制文件，内嵌 Web 界面，无外部依赖

## 快速开始

### 下载二进制

从 [Releases](../../releases) 页面下载对应平台的二进制文件。

### 运行

```bash
# 最简启动（HTTP，默认端口 8080，用户名密码 admin/admin）
./curldrop

# 指定用户名密码和存储目录
./curldrop -user myuser -pass mypass -storage /data/files

# 启用 HTTPS（自动生成自签名证书）
./curldrop -tls

# 使用已有证书
./curldrop -cert cert.pem -key key.pem
```

### 使用 .env 配置

```bash
cp .env.example .env
# 编辑 .env 文件修改配置
./curldrop
```

## curl 使用示例

```bash
# 上传单个文件
curl -u admin:admin http://localhost:8080/upload -F "file=@myfile.txt"

# 上传多个文件
curl -u admin:admin http://localhost:8080/upload -F "file=@file1.txt" -F "file=@file2.txt"

# PUT 方式上传（推荐大文件使用）
curl -u admin:admin -T bigfile.bin http://localhost:8080/upload/bigfile.bin

# 断点续上传（中断后继续）
curl -u admin:admin -C - -T bigfile.bin http://localhost:8080/upload/bigfile.bin

# 下载文件
curl -u admin:admin -OJ http://localhost:8080/download?filename=myfile.txt

# 断点续下载
curl -u admin:admin -C - -OJ http://localhost:8080/download?filename=bigfile.bin

# 查看文件列表（JSON）
curl -u admin:admin http://localhost:8080/api/files

# HTTPS 上传（自签名证书需 -k 跳过验证）
curl -k -u admin:admin https://localhost:8443/upload -F "file=@myfile.txt"
```

## 配置参数

| CLI 参数 | 环境变量 | 默认值 | 说明 |
|----------|----------|--------|------|
| `-port` | `CURLDROP_HTTP_PORT` | `8080` | HTTP 端口 |
| `-https-port` | `CURLDROP_HTTPS_PORT` | `8443` | HTTPS 端口 |
| `-storage` | `CURLDROP_STORAGE` | `./uploads` | 文件存储目录 |
| `-user` | `CURLDROP_USER` | `admin` | 用户名 |
| `-pass` | `CURLDROP_PASS` | `admin` | 密码 |
| `-cert` | `CURLDROP_CERT` | | 证书文件路径 |
| `-key` | `CURLDROP_KEY` | | 密钥文件路径 |
| `-tls` | `CURLDROP_AUTO_TLS` | `false` | 自动生成自签名证书 |
| `-auto-clean` | `CURLDROP_AUTO_CLEAN` | `0` | 自动清理天数（0=不清理） |

配置优先级：CLI 参数 > .env 文件 > 默认值

## Docker

### 使用预构建镜像

```bash
docker run -d \
  -p 8080:8080 \
  -p 8443:8443 \
  -v ./data:/data \
  -e CURLDROP_USER=admin \
  -e CURLDROP_PASS=secret \
  -e CURLDROP_AUTO_TLS=true \
  ghcr.io/gaoxinlxl/curldrop:latest
```

### 自行构建

```bash
docker build -t curldrop .
docker run -d -p 8080:8080 -v ./data:/data curldrop
```

## 从源码构建

```bash
# 需要 Go 1.22+
go build -o curldrop .

# 优化构建（减小体积）
CGO_ENABLED=0 go build -ldflags="-s -w" -o curldrop .
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | Web 文件管理界面 |
| POST | `/upload` | Multipart 表单上传 |
| PUT | `/upload/{filename}` | 流式上传（支持断点续传） |
| GET | `/download?filename=` | 文件下载（支持断点续下载） |
| GET | `/video?filename=` | 视频流播放 |
| GET | `/api/files` | JSON 文件列表 |
| POST | `/api/delete` | 删除文件 |
| GET | `/files/` | 目录浏览 |

## License

[MIT](LICENSE)
