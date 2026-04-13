package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
)

// App 应用实例，持有配置信息
type App struct {
	Config Config
}

// NewRouter 创建路由，所有路由均使用 basicAuth 中间件
func (a *App) NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// 首页
	mux.HandleFunc("/", a.basicAuth(a.indexHandler))

	// 文件上传（POST: multipart 表单 / PUT: 流式上传 + 断点续传）
	mux.HandleFunc("/upload", a.basicAuth(a.uploadRouter))
	mux.HandleFunc("/upload/", a.basicAuth(a.uploadRouter))

	// 文件下载（支持 Range 断点续下载）
	mux.HandleFunc("/download", a.basicAuth(a.downloadHandler))

	// 视频流播放
	mux.HandleFunc("/video", a.basicAuth(a.videoHandler))

	// JSON 文件列表 API
	mux.HandleFunc("/api/files", a.basicAuth(a.apiFilesHandler))

	// 文件删除 API
	mux.HandleFunc("/api/delete", a.basicAuth(a.apiDeleteHandler))

	// 目录浏览
	mux.HandleFunc("/files/", a.basicAuth(a.filesHandler))

	return mux
}

// startHTTPServer 启动 HTTP 服务
func startHTTPServer(cfg Config, mux *http.ServeMux) error {
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.HTTPPort)
	log.Printf("HTTP 服务监听 %s", addr)
	return http.ListenAndServe(addr, mux)
}

// startHTTPSWithFiles 使用证书文件启动 HTTPS 服务
func startHTTPSWithFiles(cfg Config, mux *http.ServeMux) error {
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.HTTPSPort)
	log.Printf("HTTPS 服务监听 %s（文件证书）", addr)
	return http.ListenAndServeTLS(addr, cfg.CertFile, cfg.KeyFile, mux)
}

// startHTTPSWithAutoTLS 使用内存自签名证书启动 HTTPS 服务
func startHTTPSWithAutoTLS(cfg Config, mux *http.ServeMux) error {
	cert, err := generateSelfSignedCert()
	if err != nil {
		return fmt.Errorf("生成自签名证书失败: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.HTTPSPort)
	log.Printf("HTTPS 服务监听 %s（自签名证书）", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听失败: %v", err)
	}

	tlsListener := tls.NewListener(listener, tlsConfig)
	return http.Serve(tlsListener, mux)
}
