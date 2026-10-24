package api

import (
	_ "embed"
	"net/http"
)

//go:embed docs/swagger.html
var swaggerHTML []byte

//go:embed docs/openapi.yaml
var openapiYAML []byte

func (s *Server) handleDocsUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://unpkg.com; img-src 'self' data:; connect-src 'self'")
	w.WriteHeader(http.StatusOK)
	w.Write(swaggerHTML) //nolint:errcheck
}

func (s *Server) handleDocsSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	w.Write(openapiYAML) //nolint:errcheck
}
