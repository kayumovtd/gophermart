package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/kayumovtd/gophermart/internal/logger"
)

func RequestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &loggingResponseWriter{ResponseWriter: w}

			next.ServeHTTP(ww, r)

			if ww.statusCode == 0 {
				ww.statusCode = http.StatusOK
			}

			log.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.statusCode),
				zap.Int("bytes", ww.size),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// loggingResponseWriter обёртка для логирования ответа
type loggingResponseWriter struct {
	// ResponseWriter базовый ответ
	http.ResponseWriter
	// statusCode код ответа
	statusCode int
	// size размер ответа
	size int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
	}
	size, err := lrw.ResponseWriter.Write(b)
	lrw.size += size
	return size, err
}
