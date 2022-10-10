package traefik_maintenance_plugin

import (
	"context"
	"log"
	"net/http"
)

type Config struct {
	LastModified bool `json:"lastModified,omitempty"`
}

func CreateConfig() *Config {
	return &Config{}
}

type Maintenance struct {
	name         string
	next         http.Handler
	lastModified bool
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Maintenance{
		name:         name,
		next:         next,
		lastModified: config.LastModified,
	}, nil
}

func (a *Maintenance) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if true {
		bodyBytes := []byte("Deze pagina is in onderhoud.")

		rw.Header().Del("Last-Modified")
		rw.Header().Del("Content-Length")

		if req.Header.Get("Accept") == "application/json" {
			rw.Header().Set("Content-Type", "application/json")
			bodyBytes = []byte("{}")
		}

		rw.WriteHeader(http.StatusServiceUnavailable)

		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write rewrited body: %v", err)
		}

		if flusher, ok := rw.(http.Flusher); ok {
			flusher.Flush()
		}

		return
	}

	a.next.ServeHTTP(rw, req)
}
