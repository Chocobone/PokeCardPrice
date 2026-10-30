// Package server 는 HTTP 라우팅과 핸들러를 구성한다.
package server

import (
	"encoding/json"
	"net/http"
)

// New 는 API 라우터를 반환한다.
func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	return mux
}

// healthz 는 서버 생존 여부를 응답한다. DB 상태 확인은 #3 드라이버 선정 후 추가한다.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
