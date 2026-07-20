package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouterServesHealthAndDocumentation(t *testing.T) {
	t.Parallel()

	router := Router(slog.New(slog.NewTextHandler(io.Discard, nil)), Handlers{})
	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantType    string
		wantContent string
	}{
		{"health", "/health", http.StatusOK, "application/json", `{"status":"ok"}`},
		{"health alias", "/healthz", http.StatusOK, "application/json", `{"status":"ok"}`},
		{"openapi", "/openapi.yaml", http.StatusOK, "application/yaml", "openapi: 3.0.3"},
		{"swagger UI", "/swagger/", http.StatusOK, "text/html", "Swagger UI"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if !strings.Contains(recorder.Header().Get("Content-Type"), tt.wantType) {
				t.Fatalf("Content-Type = %q, want it to contain %q", recorder.Header().Get("Content-Type"), tt.wantType)
			}
			if !strings.Contains(recorder.Body.String(), tt.wantContent) {
				t.Fatalf("body does not contain %q", tt.wantContent)
			}
		})
	}
}

func TestRouterRedirectsSwaggerWithoutTrailingSlash(t *testing.T) {
	t.Parallel()

	router := Router(slog.New(slog.NewTextHandler(io.Discard, nil)), Handlers{})
	request := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMovedPermanently || recorder.Header().Get("Location") != "/swagger/" {
		t.Fatalf("response = %d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
}
