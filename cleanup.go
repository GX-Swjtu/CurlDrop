package main

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// startCleanup 启动文件自动清理协程
// 启动时立即清理一次，之后每小时检查一次
func startCleanup(storagePath string, maxAgeDays int64) {
	if maxAgeDays <= 0 {
		return
	}

	// 启动时立即执行一次清理
	go cleanOldFiles(storagePath, maxAgeDays)

	// 定时清理
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			cleanOldFiles(storagePath, maxAgeDays)
		}
	}()
}

// cleanOldFiles 删除超过指定天数的文件
func cleanOldFiles(storagePath string, maxAgeDays int64) {
	maxAge := time.Duration(maxAgeDays) * 24 * time.Hour
	cutoff := time.Now().Add(-maxAge)

	entries, err := os.ReadDir(storagePath)
	if err != nil {
		log.Printf("清理：无法读取目录 %s: %v", storagePath, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // 仅清理文件，不递归子目录
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(storagePath, entry.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("清理：删除失败 %s: %v", path, err)
			} else {
				log.Printf("清理：已删除过期文件 %s（修改时间：%s）", entry.Name(), info.ModTime().Format("2006-01-02 15:04:05"))
			}
		}
	}
}
