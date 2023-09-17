package main

import (
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/server"
)

func main() {
	s := server.NewDefaultHTTPServer()

	err := s.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}
