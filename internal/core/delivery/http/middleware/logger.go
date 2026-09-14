package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/akordium-id/mergiate-core/internal/core/domain/shared"
)

// RequestLogger returns a Chi middleware that logs HTTP requests using slog.
func RequestLogger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				latency := time.Since(start)
				tenantID, _ := shared.GetTenantID(r.Context())

				ip := middleware.GetClientIP(r.Context())
				if ip == "" {
					ip = r.RemoteAddr
				}

				attrs := []any{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Duration("latency", latency),
					slog.String("ip", ip),
				}

				if tenantID != shared.NilID() {
					attrs = append(attrs, slog.String("tenant_id", tenantID.String()))
				}

				if ww.Status() >= 500 {
					slog.Error("http request error", attrs...)
				} else {
					slog.Info("http request", attrs...)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
