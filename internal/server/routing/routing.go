package routing

import (
	"fmt"
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/go-chi/chi"
)

func SetupRouting(h *handlers.MetricRegistryHandler) http.Handler {
	r := chi.NewRouter()
	minfo := h.GetMetricURLInfo()
	r.Route("/update", func(r chi.Router) {
		r.Post(fmt.Sprintf("/{%v}/{%v}/{%v}", minfo.Type, minfo.Name, minfo.Value),
			func(w http.ResponseWriter, r *http.Request) {
				h.UpdateMetric(w, r)
			})
	})

	r.Route("/value", func(r chi.Router) {
		r.Get(fmt.Sprintf("/{%v}/{%v}", minfo.Type, minfo.Name), func(w http.ResponseWriter, r *http.Request) {
			h.GetMetric(w, r)
		})
	})

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			h.GetAllMetrics(w, r)
		})
	})

	return r
}
