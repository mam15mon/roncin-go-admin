package server

import (
	"context"
	"net/http"
)

type ReadinessChecker interface {
	Ping(context.Context) error
}

func registerHealthHandlers(srv interface {
	HandleFunc(string, http.HandlerFunc)
}, checker ReadinessChecker) {
	srv.HandleFunc("/health/live", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		if request.Method == http.MethodGet {
			_, _ = response.Write([]byte(`{"status":"ok"}`))
		}
	})

	srv.HandleFunc("/health/ready", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := checker.Ping(request.Context()); err != nil {
			response.WriteHeader(http.StatusServiceUnavailable)
			if request.Method == http.MethodGet {
				_, _ = response.Write([]byte(`{"status":"not_ready"}`))
			}
			return
		}
		response.WriteHeader(http.StatusOK)
		if request.Method == http.MethodGet {
			_, _ = response.Write([]byte(`{"status":"ok"}`))
		}
	})
}
