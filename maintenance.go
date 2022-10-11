package traefik_maintenance_plugin

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var pwd = "/plugins-local/src/github.com/programic/traefik-maintenance-plugin"

type Config struct {
	LastModified bool `json:"lastModified,omitempty"`
}

func CreateConfig() *Config {
	return &Config{}
}

func ReadFile(file string) []byte {
	data, err := os.ReadFile(pwd + "/" + file)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func getIps(req *http.Request) []string {
	var ips []string

	if req.RemoteAddr != "" {
		ip, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			ip = req.RemoteAddr
		}
		ips = append(ips, ip)
	}

	forwardedFor := req.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		for _, ip := range strings.Split(forwardedFor, ",") {
			ips = append(ips, strings.TrimSpace(ip))
		}
	}

	return ips
}

func matchIps(req *http.Request) bool {

	return false
}

func matchHost(req *http.Request) bool {

	// https://github.com/tomMoulard/fail2ban/blob/v0.6.6/fail2ban.go#L250

	return true
}

type Maintenance struct {
	name     string
	next     http.Handler
	bodyHtml []byte
	bodyJson []byte
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Maintenance{
		name:     name,
		next:     next,
		bodyHtml: ReadFile("maintenance.html"),
		bodyJson: ReadFile("maintenance.json"),
	}, nil
}

func (a *Maintenance) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	block := false

	if matchHost(req) {
		if !matchIps(req) {
			block = true
		}
	}

	if block {

		var body []byte

		rw.Header().Del("Last-Modified")
		rw.Header().Del("Content-Length")

		if req.Header.Get("Accept") == "application/json" {
			rw.Header().Set("Content-Type", "application/json; charset=utf-8")

			body = a.bodyJson
		} else {
			body = a.bodyHtml
		}

		rw.WriteHeader(http.StatusServiceUnavailable)

		if _, err := rw.Write(body); err != nil {
			log.Printf("unable to write rewrited body: %v", err)
		}

		if flusher, ok := rw.(http.Flusher); ok {
			flusher.Flush()
		}

		return
	}

	a.next.ServeHTTP(rw, req)
}
