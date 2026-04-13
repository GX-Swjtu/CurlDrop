package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	HTTPPort      int    // HTTP 监听端口
	HTTPSPort     int    // HTTPS 监听端口
	StoragePath   string // 文件存储目录
	AutoCleanDays int64  // 自动清理天数（0=不清理）
	Username      string // Basic Auth 用户名
	Password      string // Basic Auth 密码
	CertFile      string // TLS 证书文件路径
	KeyFile       string // TLS 密钥文件路径
	AutoTLS       bool   // 自动生成自签名证书
}

// getEnvInt 从环境变量获取整数值，不存在则返回默认值
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

// getEnvInt64 从环境变量获取 int64 值
func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

// getEnvStr 从环境变量获取字符串值
func getEnvStr(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvBool 从环境变量获取布尔值
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes"
	}
	return defaultVal
}

// loadConfig 按优先级加载配置：默认值 → .env → CLI 参数
func loadConfig() Config {
	// 加载 .env 文件（可选，不存在不报错）
	_ = godotenv.Load()

	// 从环境变量读取（覆盖默认值）
	cfg := Config{
		HTTPPort:      getEnvInt("CURLDROP_HTTP_PORT", 8080),
		HTTPSPort:     getEnvInt("CURLDROP_HTTPS_PORT", 8443),
		StoragePath:   getEnvStr("CURLDROP_STORAGE", "./uploads"),
		AutoCleanDays: getEnvInt64("CURLDROP_AUTO_CLEAN", 0),
		Username:      getEnvStr("CURLDROP_USER", "admin"),
		Password:      getEnvStr("CURLDROP_PASS", "admin"),
		CertFile:      getEnvStr("CURLDROP_CERT", ""),
		KeyFile:       getEnvStr("CURLDROP_KEY", ""),
		AutoTLS:       getEnvBool("CURLDROP_AUTO_TLS", false),
	}

	// CLI 参数（最高优先级，覆盖环境变量）
	flag.IntVar(&cfg.HTTPPort, "port", cfg.HTTPPort, "HTTP 监听端口")
	flag.IntVar(&cfg.HTTPSPort, "https-port", cfg.HTTPSPort, "HTTPS 监听端口")
	flag.StringVar(&cfg.StoragePath, "storage", cfg.StoragePath, "文件存储目录")
	flag.Int64Var(&cfg.AutoCleanDays, "auto-clean", cfg.AutoCleanDays, "自动清理天数（0=不清理）")
	flag.StringVar(&cfg.Username, "user", cfg.Username, "Basic Auth 用户名")
	flag.StringVar(&cfg.Password, "pass", cfg.Password, "Basic Auth 密码")
	flag.StringVar(&cfg.CertFile, "cert", cfg.CertFile, "TLS 证书文件路径")
	flag.StringVar(&cfg.KeyFile, "key", cfg.KeyFile, "TLS 密钥文件路径")
	flag.BoolVar(&cfg.AutoTLS, "tls", cfg.AutoTLS, "自动生成自签名证书")
	flag.Parse()

	return cfg
}

// getLocalIPs 获取本机所有 IP 地址
func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}
	return ips
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg := loadConfig()

	// 确保存储目录存在
	if err := os.MkdirAll(cfg.StoragePath, os.ModePerm); err != nil {
		log.Fatalf("无法创建存储目录 %s: %v", cfg.StoragePath, err)
	}

	// 打印启动信息（不打印密码）
	log.Printf("CurlDrop 启动中...")
	log.Printf("  HTTP  端口: %d", cfg.HTTPPort)
	log.Printf("  存储目录: %s", cfg.StoragePath)
	log.Printf("  用户名: %s", cfg.Username)
	if cfg.AutoCleanDays > 0 {
		log.Printf("  自动清理: %d 天", cfg.AutoCleanDays)
	}

	// 启动文件自动清理
	if cfg.AutoCleanDays > 0 {
		startCleanup(cfg.StoragePath, cfg.AutoCleanDays)
		log.Printf("  文件自动清理已启动（%d 天）", cfg.AutoCleanDays)
	}

	app := &App{Config: cfg}
	mux := app.NewRouter()

	var wg sync.WaitGroup

	// 打印访问地址
	ips := getLocalIPs()
	fmt.Println()
	fmt.Println("=== CurlDrop 已启动 ===")
	fmt.Printf("  HTTP:  http://localhost:%d\n", cfg.HTTPPort)
	for _, ip := range ips {
		fmt.Printf("         http://%s:%d\n", ip, cfg.HTTPPort)
	}

	// 启动 HTTP 服务
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startHTTPServer(cfg, mux); err != nil {
			log.Fatalf("HTTP 服务启动失败: %v", err)
		}
	}()

	// 启动 HTTPS 服务
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		fmt.Printf("  HTTPS: https://localhost:%d (文件证书)\n", cfg.HTTPSPort)
		for _, ip := range ips {
			fmt.Printf("         https://%s:%d\n", ip, cfg.HTTPSPort)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startHTTPSWithFiles(cfg, mux); err != nil {
				log.Fatalf("HTTPS 服务启动失败: %v", err)
			}
		}()
	} else if cfg.AutoTLS {
		fmt.Printf("  HTTPS: https://localhost:%d (自签名证书)\n", cfg.HTTPSPort)
		for _, ip := range ips {
			fmt.Printf("         https://%s:%d\n", ip, cfg.HTTPSPort)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := startHTTPSWithAutoTLS(cfg, mux); err != nil {
				log.Fatalf("HTTPS 服务启动失败（自签名）: %v", err)
			}
		}()
	} else {
		log.Printf("  HTTPS 未启用（使用 -tls 自动生成证书，或 -cert/-key 指定证书文件）")
	}

	fmt.Println()
	fmt.Println("curl 使用示例:")
	fmt.Printf("  上传: curl -u %s:<密码> http://localhost:%d/upload -F \"file=@文件名\"\n", cfg.Username, cfg.HTTPPort)
	fmt.Printf("  上传: curl -u %s:<密码> -T 文件名 http://localhost:%d/upload/文件名\n", cfg.Username, cfg.HTTPPort)
	fmt.Printf("  下载: curl -u %s:<密码> -OJ http://localhost:%d/download?filename=文件名\n", cfg.Username, cfg.HTTPPort)
	fmt.Printf("  续传: curl -u %s:<密码> -C - -T 文件名 http://localhost:%d/upload/文件名\n", cfg.Username, cfg.HTTPPort)
	fmt.Println("========================")

	wg.Wait()
}
