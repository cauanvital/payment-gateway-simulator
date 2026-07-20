package api

import (
	"io/fs"
	"net/http"

	projectdocs "github.com/cauanvital/payment-gateway-simulator/docs"
)

func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	spec, err := projectdocs.Files.ReadFile("openapi.yaml")
	if err != nil {
		http.Error(w, "OpenAPI specification unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(spec)
}

func SwaggerUIHandler() (http.Handler, error) {
	staticFiles, err := fs.Sub(projectdocs.Files, "swagger-ui")
	if err != nil {
		return nil, err
	}

	return http.StripPrefix(
		"/swagger/",
		http.FileServer(http.FS(staticFiles)),
	), nil
}
