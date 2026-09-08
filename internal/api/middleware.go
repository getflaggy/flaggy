package api

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// RequestLogger logs each request using slog.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var reqBody string
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			reqBody = string(b)
			r.Body = io.NopCloser(bytes.NewBuffer(b))
		}

		stream := r.Header.Get("Accept") == "text/event-stream"
		if stream {
			slog.Info("stream connected", "path", r.URL.Path)
		}
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK, captureBody: !stream}
		next.ServeHTTP(ww, r)

		if stream {
			slog.Info("stream closed", "path", r.URL.Path, "status", ww.status, "duration", time.Since(start).String())
			return
		}

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration", time.Since(start).String(),
		}
		if q := r.URL.RawQuery; q != "" {
			attrs = append(attrs, "query", q)
		}
		if reqBody != "" {
			attrs = append(attrs, "req_body", reqBody)
		}
		if len(ww.body) > 0 {
			attrs = append(attrs, "res_body", string(ww.body))
		}

		if ww.status >= 500 {
			slog.Error("request", attrs...)
		} else if ww.status >= 400 {
			slog.Warn("request", attrs...)
		} else {
			slog.Info("request", attrs...)
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	body        []byte
	captureBody bool
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.captureBody {
		w.body = append(w.body, b...)
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying ResponseWriter for Flush/Hijack.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// CORS handles Cross-Origin Resource Sharing for all requests.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// JSONContentType sets Content-Type to application/json for all responses.
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
