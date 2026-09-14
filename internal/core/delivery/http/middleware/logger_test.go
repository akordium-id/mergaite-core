package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestRequestLogger_ClientIP(t *testing.T) {
	tests := []struct {
		name         string
		setupHeader  string
		remoteAddr   string
		expectedIPIn string
	}{
		{
			name:         "Direct connection without proxy header uses RemoteAddr",
			setupHeader:  "",
			remoteAddr:   "192.168.1.50:54321",
			expectedIPIn: "192.168.1.50:54321",
		},
		{
			name:         "Behind proxy with X-Real-IP uses header IP",
			setupHeader:  "203.0.113.195",
			remoteAddr:   "172.18.0.2:40000",
			expectedIPIn: "203.0.113.195",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			oldLogger := slog.Default()
			slog.SetDefault(logger)
			defer slog.SetDefault(oldLogger)

			handler := chimiddleware.ClientIPFromHeader("X-Real-IP")(RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.setupHeader != "" {
				req.Header.Set("X-Real-IP", tt.setupHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			logOutput := buf.String()
			if !strings.Contains(logOutput, tt.expectedIPIn) {
				t.Errorf("expected log output to contain IP %q, got %s", tt.expectedIPIn, logOutput)
			}
		})
	}
}
