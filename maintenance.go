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

type rewriteBody struct {
	name         string
	next         http.Handler
	lastModified bool
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &rewriteBody{
		name:         name,
		next:         next,
		lastModified: config.LastModified,
	}, nil
}

func (r *rewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if true {
		rw.WriteHeader(http.StatusServiceUnavailable)
		rw.Header().Del("Content-Length")

		bodyBytes := []byte("Deze pagina is even niet bereikbaar.")

		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write rewrited body: %v", err)
		}

		if flusher, ok := rw.(http.Flusher); ok {
			flusher.Flush()
		}

		return
	}

	r.next.ServeHTTP(rw, req)
}
