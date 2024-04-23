package handlers

// Provides middleware to limit request body size.

import (
	"net/http"
)

// WithBodySizeLimit sets response writer proxy to close connection,
// if request body size is larger than configured maximum.
func WithBodySizeLimit(h http.Handler, maxBytes uint64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
		h.ServeHTTP(w, r)
	})
}
