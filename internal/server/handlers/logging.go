// Provides middleware to log request parameters.
package handlers

import (
	"net/http"
	"time"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
)

// WithLogging retunrs handler that logs request parameters.
func WithLogging(h http.Handler, log logging.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseData := logging.ResponseData{}
		lw := logging.LoggingRepsonseWriter{ResponseWriter: w, Data: &responseData}

		start := time.Now()
		h.ServeHTTP(&lw, r)
		duration := time.Since(start)

		log.Infof("uri: %v, method: %v, status: %v, duration: %v, size: %v",
			r.RequestURI,
			r.Method,
			responseData.Status,
			duration,
			responseData.Size,
		)
	})
}
