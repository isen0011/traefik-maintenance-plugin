package traefik_maintenance_plugin

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var pwd = "/plugins-local/src/github.com/programic/traefik-maintenance-plugin"

type Config struct {
	InformUrl string `json:"informUrl,omitempty"`
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

var hosts Hosts

type Host struct {
	Regex    string
	AllowIps []string
}

type Hosts struct {
	Hosts []Host
}

func inform(informUrl string) {
	t := time.NewTicker(5 * time.Second)
	for ; true; <-t.C {

		client := http.Client{
			Timeout: time.Second * 5,
		}

		req, _ := http.NewRequest(http.MethodGet, informUrl, nil)
		res, doErr := client.Do(req)
		if doErr != nil {
			log.Fatal(doErr)
		}

		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
		decodeErr := decoder.Decode(&hosts)
		if decodeErr != nil {
			log.Fatal(decodeErr)
		}
	}
}

type Maintenance struct {
	name     string
	next     http.Handler
	config   *Config
	bodyHtml []byte
	bodyJson []byte
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	go inform(config.InformUrl)

	return &Maintenance{
		name:     name,
		next:     next,
		config:   config,
		bodyHtml: ReadFile("maintenance.html"),
		bodyJson: ReadFile("maintenance.json"),
	}, nil
}

func (a *Maintenance) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	block := false

	log.Println(hosts.Hosts)

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
