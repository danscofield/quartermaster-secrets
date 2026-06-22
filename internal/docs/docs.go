package docs

import (
	_ "embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed index.html
var indexHTML []byte

// Mount registers documentation routes on the router.
func Mount(r chi.Router) {
	r.Get("/docs", redirectToDocs)
	r.Get("/docs/", serveIndex)
	r.Get("/openapi.yaml", serveOpenAPI)
}

func redirectToDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/", http.StatusFound)
}

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}

func serveIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexHTML)
}
