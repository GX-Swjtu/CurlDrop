package main

import (
	"log"
	"net/http"
)

// basicAuth Basic Auth 认证中间件
func (a *App) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != a.Config.Username || password != a.Config.Password {
			if ok {
				log.Printf("认证失败: 用户=%s 来源=%s", username, r.RemoteAddr)
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="CurlDrop"`)
			http.Error(w, "需要认证", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
