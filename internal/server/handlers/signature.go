package handlers

// Signature checking middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
)

func WithSignatureCheck(h http.Handler, log logging.Logger, key []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get("HashSHA256")
		if len(signature) == 0 {
			h.ServeHTTP(w, r)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Failed to read request body: %v", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		err = r.Body.Close()
		if err != nil {
			log.Errorf("Failed to close request body: %v", err)
		}

		err = encryption.CheckData(body, key, signature)
		if err != nil {
			log.Errorf("Failed to validate body signature: %v", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		log.Debugf("Signature validated succesfully")

		r.Body = io.NopCloser(bytes.NewReader(body))

		h.ServeHTTP(w, r)
	})
}
