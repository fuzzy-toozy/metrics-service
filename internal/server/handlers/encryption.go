package handlers

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
)

func WithDecryption(h http.Handler, log logging.Logger, privKey *rsa.PrivateKey) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength == 0 {
			h.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body: %v", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		decBody, err := encryption.DecryptRequestBody(bytes.NewBuffer(body), privKey)
		if err != nil {
			log.Errorf("Failed to decrypt request body: %v", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(decBody)

		h.ServeHTTP(w, r)
	})
}
