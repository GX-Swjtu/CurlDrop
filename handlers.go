package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FileInfo 文件信息（用于 JSON API 返回）
type FileInfo struct {
	Name     string    `json:"name"`     // 文件名
	Size     int64     `json:"size"`     // 文件大小（字节）
	Modified time.Time `json:"modified"` // 修改时间
	IsDir    bool      `json:"is_dir"`   // 是否为目录
}

// safePath 安全地解析文件路径，防止路径遍历攻击
// 返回完整路径和安全的文件名，如果路径不安全则返回错误
func (a *App) safePath(name string) (string, error) {
	// 提取纯文件名，去除目录部分
	name = filepath.Base(name)
	if name == "." || name == ".." || name == "" {
		return "", fmt.Errorf("无效的文件名")
	}

	fullPath := filepath.Join(a.Config.StoragePath, name)

	// 二次校验：确保绝对路径在存储目录内
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("路径解析失败")
	}
	absStorage, err := filepath.Abs(a.Config.StoragePath)
	if err != nil {
		return "", fmt.Errorf("存储路径解析失败")
	}
	if !strings.HasPrefix(absPath, absStorage) {
		return "", fmt.Errorf("禁止访问")
	}

	return fullPath, nil
}

// indexHandler 返回嵌入的 index.html 页面
func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

// uploadHandler 处理 multipart 表单文件上传（POST /upload）
// 使用流式读取，内存占用极低，支持超大文件
func (a *App) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "无法解析 multipart 表单: "+err.Error(), http.StatusBadRequest)
		return
	}

	var uploaded []string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "读取表单失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 跳过非文件字段
		if part.FileName() == "" {
			part.Close()
			continue
		}

		// 安全检查文件名
		safeName := filepath.Base(part.FileName())
		if safeName == "." || safeName == ".." {
			part.Close()
			continue
		}

		fullPath, err := a.safePath(safeName)
		if err != nil {
			part.Close()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 创建目标文件，流式写入
		dst, err := os.Create(fullPath)
		if err != nil {
			part.Close()
			http.Error(w, "创建文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = io.Copy(dst, part)
		dst.Close()
		part.Close()

		if err != nil {
			http.Error(w, "写入文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		uploaded = append(uploaded, safeName)
		log.Printf("上传文件: %s 来源: %s", safeName, r.RemoteAddr)
	}

	if len(uploaded) == 0 {
		http.Error(w, "未收到文件", http.StatusBadRequest)
		return
	}

	// 返回上传结果
	w.WriteHeader(http.StatusOK)
	for _, name := range uploaded {
		fmt.Fprintf(w, "%s 上传成功\n", name)
	}
}

// putUploadHandler 处理 PUT 方式的文件上传（PUT /upload/{filename}）
// 支持断点续传：通过 Content-Range 头指定偏移量
func (a *App) putUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "仅支持 PUT 方法", http.StatusMethodNotAllowed)
		return
	}

	// 从 URL 路径提取文件名: /upload/filename
	fileName := strings.TrimPrefix(r.URL.Path, "/upload/")
	if fileName == "" {
		http.Error(w, "缺少文件名", http.StatusBadRequest)
		return
	}

	fullPath, err := a.safePath(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查是否为断点续传（Content-Range 头）
	contentRange := r.Header.Get("Content-Range")
	var offset int64
	isResume := false

	if contentRange != "" {
		// 解析 Content-Range: bytes start-end/total
		offset, err = parseContentRangeStart(contentRange)
		if err != nil {
			http.Error(w, "无效的 Content-Range: "+err.Error(), http.StatusBadRequest)
			return
		}
		isResume = true
	}

	if isResume {
		// 断点续传：打开已有文件并定位到偏移量
		dst, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			http.Error(w, "打开文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := dst.Seek(offset, io.SeekStart); err != nil {
			http.Error(w, "定位偏移量失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, r.Body); err != nil {
			http.Error(w, "写入文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("续传文件: %s 偏移: %d 来源: %s", filepath.Base(fullPath), offset, r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s 续传成功\n", filepath.Base(fullPath))
	} else {
		// 全新上传：创建文件
		dst, err := os.Create(fullPath)
		if err != nil {
			http.Error(w, "创建文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, r.Body); err != nil {
			http.Error(w, "写入文件失败: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("上传文件: %s 来源: %s", filepath.Base(fullPath), r.RemoteAddr)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "%s 上传成功\n", filepath.Base(fullPath))
	}
}

// uploadRouter 根据请求方法路由上传请求
func (a *App) uploadRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/upload" {
		a.uploadHandler(w, r)
	} else if r.Method == http.MethodPut {
		a.putUploadHandler(w, r)
	} else {
		http.Error(w, "不支持的方法", http.StatusMethodNotAllowed)
	}
}

// downloadHandler 处理文件下载（GET /download?filename=xxx）
// http.ServeFile 原生支持 Range 请求，自动支持断点续下载
func (a *App) downloadHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.FormValue("filename")
	if fileName == "" {
		http.Error(w, "缺少 filename 参数", http.StatusBadRequest)
		return
	}

	fullPath, err := a.safePath(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	safeName := filepath.Base(fullPath)
	// 正确引用 Content-Disposition 中的文件名
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(safeName, `"`, `\"`)))
	log.Printf("下载文件: %s 来源: %s", safeName, r.RemoteAddr)
	http.ServeFile(w, r, fullPath)
}

// videoHandler 处理视频流播放（GET /video?filename=xxx）
// 使用 http.ServeContent 支持 Range 请求（视频拖拽进度条）
func (a *App) videoHandler(w http.ResponseWriter, r *http.Request) {
	fileName := r.FormValue("filename")
	if fileName == "" {
		http.Error(w, "缺少 filename 参数", http.StatusBadRequest)
		return
	}

	fullPath, err := a.safePath(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "文件不存在", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 根据文件扩展名设置 Content-Type
	ext := strings.ToLower(filepath.Ext(fileName))
	contentType := "video/mp4"
	switch ext {
	case ".webm":
		contentType = "video/webm"
	case ".ogg", ".ogv":
		contentType = "video/ogg"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	log.Printf("播放视频: %s 来源: %s", filepath.Base(fullPath), r.RemoteAddr)
	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

// apiFilesHandler 返回文件列表的 JSON API（GET /api/files）
func (a *App) apiFilesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.Config.StoragePath)
	if err != nil {
		http.Error(w, "无法读取目录", http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:     entry.Name(),
			Size:     info.Size(),
			Modified: info.ModTime(),
			IsDir:    entry.IsDir(),
		})
	}

	// 确保返回空数组而不是 null
	if files == nil {
		files = []FileInfo{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(files)
}

// apiDeleteHandler 删除指定文件（POST /api/delete）
func (a *App) apiDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	fileName := r.FormValue("filename")
	if fileName == "" {
		http.Error(w, "缺少 filename 参数", http.StatusBadRequest)
		return
	}

	fullPath, err := a.safePath(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "文件不存在", http.StatusNotFound)
		} else {
			http.Error(w, "删除失败: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("删除文件: %s 来源: %s", filepath.Base(fullPath), r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s 已删除\n", filepath.Base(fullPath))
}

// filesHandler 处理目录浏览和文件直接访问（GET /files/...）
func (a *App) filesHandler(w http.ResponseWriter, r *http.Request) {
	// 去掉 /files/ 前缀，获取相对路径
	relPath := strings.TrimPrefix(r.URL.Path, "/files/")
	if relPath == "" {
		relPath = "."
	}

	// 清理路径，防止遍历
	relPath = filepath.Clean(relPath)
	fullPath := filepath.Join(a.Config.StoragePath, relPath)

	// 安全校验
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	absStorage, err := filepath.Abs(a.Config.StoragePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !strings.HasPrefix(absPath, absStorage) {
		http.Error(w, "禁止访问", http.StatusForbidden)
		return
	}

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if fileInfo.IsDir() {
		// 目录：生成 HTML 文件列表
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			http.Error(w, "无法读取目录", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>目录列表</title>`)
		fmt.Fprint(w, `<style>body{font-family:sans-serif;margin:2em}a{text-decoration:none;color:#0366d6}a:hover{text-decoration:underline}li{margin:0.3em 0}</style>`)
		fmt.Fprint(w, `</head><body><h1>目录列表</h1><ul>`)

		// 如果不在根目录，添加返回上级链接
		if relPath != "." {
			parent := filepath.Dir(relPath)
			if parent == "." {
				parent = ""
			}
			fmt.Fprintf(w, `<li><a href="/files/%s">../</a></li>`, parent)
		}

		for _, entry := range entries {
			name := entry.Name()
			entryPath := relPath + "/" + name
			if relPath == "." {
				entryPath = name
			}
			if entry.IsDir() {
				fmt.Fprintf(w, `<li><a href="/files/%s/">%s/</a></li>`, entryPath, name)
			} else {
				fmt.Fprintf(w, `<li><a href="/files/%s">%s</a></li>`, entryPath, name)
			}
		}

		fmt.Fprint(w, `</ul></body></html>`)
	} else {
		// 文件：直接下载
		http.ServeFile(w, r, fullPath)
	}
}

// parseContentRangeStart 解析 Content-Range 头中的起始偏移量
// 格式: bytes start-end/total 或 bytes start-end/*
func parseContentRangeStart(header string) (int64, error) {
	// Content-Range: bytes 1000-1999/5000
	header = strings.TrimPrefix(header, "bytes ")
	parts := strings.Split(header, "-")
	if len(parts) < 2 {
		return 0, fmt.Errorf("格式错误")
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("起始偏移量无效: %v", err)
	}
	return start, nil
}
