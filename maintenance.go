package traefik_maintenance_plugin

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	InformUrl string `json:"informUrl,omitempty"`
}

type Host struct {
	Regex    string
	AllowIps []string
}

type Maintenance struct {
	name   string
	next   http.Handler
	config *Config
}

// Global variables
var hosts []Host

func CreateConfig() *Config {
	return &Config{}
}

// Html body of the maintenance page
func tplBodyHtml() []byte {
	return []byte(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Under maintenance</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="text-center grid place-items-center h-screen">
	<div>
	<h1 class="text-3xl font-bold mb-2">
		This page is under maintenance
	</h1>
	<p>Please come back later.</p>
	</div>
</body>
</html>`)
}

// Json body of the maintenance page
func tplBodyJson() []byte {
	return []byte("{\"message\": \"This page is under maintenance. Please come back later.\"}")
}

// Inform if there are hosts in maintenance
func Inform(informUrl string) {
	t := time.NewTicker(5 * time.Second)
	for ; true; <-t.C {

		client := http.Client{
			Timeout: time.Second * 5,
		}

		req, _ := http.NewRequest(http.MethodGet, informUrl, nil)
		res, doErr := client.Do(req)
		if doErr != nil {
			log.Printf("Inform: %v", doErr) // Don't fatal, just go further
			continue
		}

		defer res.Body.Close()

		decoder := json.NewDecoder(res.Body)
		decodeErr := decoder.Decode(&hosts)
		if decodeErr != nil {
			log.Printf("Inform: %v", decodeErr) // Don't fatal, just go further
			continue
		}

		log.Printf("Inform response: %v", hosts)
	}
}

// Get all the client's ips
func GetClientIps(req *http.Request) []string {
	var ips []string

	if req.RemoteAddr != "" {
		ip, _, splitErr := net.SplitHostPort(req.RemoteAddr)
		if splitErr != nil {
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

// Check if one of the ips has access
func CheckIpAllowed(req *http.Request, host Host) bool {
	for _, ip := range GetClientIps(req) {
		for _, allowIp := range host.AllowIps {
			if ip == allowIp {
				return true
			}
		}
	}

	return false
}

// Check if the host is under maintenance
func CheckIfMaintenance(req *http.Request) bool {
	for _, host := range hosts {
		if matched, _ := regexp.Match(host.Regex, []byte(req.Host)); matched {
			return !CheckIpAllowed(req, host)
		}
	}

	return false
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	go Inform(config.InformUrl)

	return &Maintenance{
		name:   name,
		next:   next,
		config: config,
	}, nil
}

func (a *Maintenance) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if CheckIfMaintenance(req) {

		var body []byte

		rw.Header().Del("Last-Modified")
		rw.Header().Del("Content-Length")

		if req.Header.Get("Accept") == "application/json" {
			rw.Header().Set("Content-Type", "application/json; charset=utf-8")

			body = tplBodyJson()
		} else {
			body = tplBodyHtml()
		}

		rw.WriteHeader(http.StatusServiceUnavailable)

		if _, err := rw.Write(body); err != nil {
			log.Printf("ServeHTTP: %v", err)
		}

		if flusher, ok := rw.(http.Flusher); ok {
			flusher.Flush()
		}

		return
	}

	a.next.ServeHTTP(rw, req)
}
