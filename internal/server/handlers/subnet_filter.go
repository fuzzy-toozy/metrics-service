package handlers

import (
	"net"
	"net/http"

	logging "github.com/fuzzy-toozy/metrics-service/internal/log"
)

func WithSubnetFilter(h http.Handler, log logging.Logger, subnet *net.IPNet) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subnet == nil {
			h.ServeHTTP(w, r)
			return
		}

		realIP := r.Header.Get("X-Real-IP")

		parsedIP := net.ParseIP(realIP)

		if parsedIP == nil {
			log.Debugf("Failed to parse ip from X-Real-IP header. Header value: '%s'", realIP)
			http.Error(w, "", http.StatusForbidden)
			return
		}

		maskedIP := parsedIP.Mask(subnet.Mask)
		if !maskedIP.Equal(subnet.IP) {
			log.Debugf("Client IP address is not in trusted subnet. Address: %s. Trusted Subnet: %s", realIP, subnet.String())
			http.Error(w, "", http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}
