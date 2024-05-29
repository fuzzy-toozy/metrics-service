// Package handlers Implements various handlers and middlewares for metrics server.
package mhttp

// Server's routing setup

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func SetupRouting(h *MetricRegistryHandler) http.Handler {
	r := chi.NewRouter()
	minfo := h.GetMetricURLInfo()
	swaggerHandler := http.FileServer(http.Dir("./docs"))
	r.Route("/swagger", func(r chi.Router) {
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix("/swagger/", swaggerHandler).ServeHTTP(w, r)
		})
	})

	r.Route("/ping", func(r chi.Router) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.HealthCheck(w, r)
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

	r.Route("/updates", func(r chi.Router) {
		handlerFunc := middleware.AllowContentType("application/json")
		handler := handlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.UpdateMetricsFromJSON(w, r)
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
