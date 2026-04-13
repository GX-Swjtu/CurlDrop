package main

import (
	_ "embed"
)

// indexHTML 嵌入的前端页面，编译时打包进二进制
//
//go:embed templates/index.html
var indexHTML []byte
