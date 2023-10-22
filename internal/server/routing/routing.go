package routing

import (
	"fmt"
	"net/http"

	"github.com/fuzzy-toozy/metrics-service/internal/server/handlers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func SetupRouting(h *handlers.MetricRegistryHandler) http.Handler {
	r := chi.NewRouter()
	minfo := h.GetMetricURLInfo()

	r.Route("/ping", func(r chi.Router) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.CheckDatabaseConnection(w, r)
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		})
	})

	r.Route("/update", func(r chi.Router) {
		r.Post(fmt.Sprintf("/{%v}/{%v}/{%v}", minfo.Type, minfo.Name, minfo.Value),
			func(w http.ResponseWriter, r *http.Request) {
				h.UpdateMetric(w, r)
			})

		handlerFunc := middleware.AllowContentType("application/json")
		handler := handlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.UpdateMetricFromJSON(w, r)
		}))

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		})
	})

	r.Route("/value", func(r chi.Router) {
		r.Get(fmt.Sprintf("/{%v}/{%v}", minfo.Type, minfo.Name), func(w http.ResponseWriter, r *http.Request) {
			h.GetMetric(w, r)
		})

		handlerFunc := middleware.AllowContentType("application/json")
		handler := handlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.GetMetricJSON(w, r)
		}))

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r)
		})
	})

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			h.GetAllMetrics(w, r)
		})
	})

	return r
}
