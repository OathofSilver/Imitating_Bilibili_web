package middleware

import "net/http"

// CORS 允许本地前端开发服务器跨域访问各域 api。
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// Auth 是登录态校验的占位实现。
// 骨架阶段不做鉴权；按 identity/auth 规范，受保护接口必须在服务端校验登录态，
// 该逻辑将在认证相关变更中落地，届时此处替换为真实校验。
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return next
}
