package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rizesql/kerberos/internal/o11y/logging"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func WithLogging(log *logging.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			clientName := r.RemoteAddr

			log.Info(fmt.Sprintf("Client %s Connected", clientName))
			log.Info("Server received a request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)

			lrw := newLoggingResponseWriter(w)

			defer func() {
				log.Info(fmt.Sprintf("Client %s received response: status %d", clientName, lrw.statusCode))
				log.Info("request finished",
					"method", r.Method,
					"path", r.URL.Path,
					"status", lrw.statusCode,
					"duration", time.Since(start),
				)
			}()

			next.ServeHTTP(lrw, r)
		}
	}
}
