package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
)

func staticFileName(urlPath string) (string, bool) {
	if strings.Contains(urlPath, "\x00") {
		return "", false
	}
	clean := path.Clean("/" + urlPath)
	name := strings.TrimPrefix(clean, "/")
	if strings.HasSuffix(urlPath, "/") {
		name = path.Join(name, "index.html")
	}
	if name == "." || name == "" {
		name = "index.html"
	}
	return name, true
}

func clientAddressLogFields(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteIP = host
	}

	return fmt.Sprintf("remote_addr=%q remote_ip=%q x_forwarded_for=%q x_real_ip=%q forwarded=%q",
		r.RemoteAddr,
		remoteIP,
		r.Header.Get("X-Forwarded-For"),
		r.Header.Get("X-Real-IP"),
		r.Header.Get("Forwarded"),
	)
}

func wasmBundleLogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &responseLogRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		name, ok := staticFileName(r.URL.Path)
		if !ok || (!strings.HasPrefix(name, "app.") && name != "app.wasm") || !strings.HasSuffix(name, ".wasm") || recorder.Status() < 200 || recorder.Status() >= 300 {
			return
		}
		log.Printf("wasm bundle served method=%s path=%q status=%d bytes=%d %s",
			r.Method,
			r.URL.Path,
			recorder.Status(),
			recorder.bytes,
			clientAddressLogFields(r),
		)
	})
}
