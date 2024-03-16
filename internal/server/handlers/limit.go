package handlers

import (
	"net/http"
)

func WithBodySizeLimit(h http.Handler, maxBytes uint64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
		h.ServeHTTP(w, r)
	})
}
