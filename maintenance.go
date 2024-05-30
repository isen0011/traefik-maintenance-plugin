package traefik_maintenance_plugin

import (
  "bytes"
  "context"
  "encoding/json"
  "fmt"
  "log"
  "mime"
  "net"
  "net/http"
  "os"
  "regexp"
  "strings"
  "time"
)

type Config struct {
  InformUrl      string        `yaml:"informUrl"`
  InformInterval time.Duration `yaml:"informInterval"`
  InformTimeout  time.Duration `yaml:"informTimeout"`
  BaseTemplatePath string      `yaml:"baseTemplatePath"`
}

type Host struct {
  Regex    string
  AllowIps []string
  Template string
  Heading  string
  Message  string
}

type Maintenance struct {
  name   string
  next   http.Handler
  config *Config
}

type ResponseWriter struct {
  buffer bytes.Buffer

  http.ResponseWriter
}

// Global variables
var hosts []Host

func CreateConfig() *Config {
  return &Config{
    InformInterval: 60,
    InformTimeout:  5,
  }
}

// Inform if there are hosts in maintenance
func Inform(config *Config) {
  t := time.NewTicker(time.Second * config.InformInterval)
  defer t.Stop()

  for ; true; <-t.C {
    client := http.Client{
      Timeout: time.Second * config.InformTimeout,
    }

    req, _ := http.NewRequest(http.MethodGet, config.InformUrl, nil)
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

    for i, host := range hosts {
      hosts[i].Template = config.BaseTemplatePath + host.Template
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

// Check if one of the client ips has access
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

func GetHost(req *http.Request) Host {
  for _, host := range hosts {
    if matched, _ := regexp.Match(host.Regex, []byte(req.Host)); matched {
      return host
    }
  }
  return Host{}
}

func (rw *ResponseWriter) Header() http.Header {
  return rw.ResponseWriter.Header()
}

func (rw *ResponseWriter) Write(bytes []byte) (int, error) {
  return rw.buffer.Write(bytes)
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
  rw.ResponseWriter.Header().Del("Last-Modified")
  rw.ResponseWriter.Header().Del("Content-Length")

  rw.ResponseWriter.WriteHeader(http.StatusServiceUnavailable)
}

func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
  go Inform(config)

  return &Maintenance{
    name:   name,
    next:   next,
    config: config,
  }, nil
}

func (a *Maintenance) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

  if CheckIfMaintenance(req) {
    wrappedWriter := &ResponseWriter{
      ResponseWriter: rw,
    }

    a.next.ServeHTTP(wrappedWriter, req)

    bytes := []byte{}

    contentType := wrappedWriter.Header().Get("Content-Type")
    if contentType != "" {
      mt, _, _ := mime.ParseMediaType(contentType)
      host := GetHost(req)
      bytes = getTemplate(mt, host)
    }

    rw.Write(bytes)

    if flusher, ok := rw.(http.Flusher); ok {
      flusher.Flush()
    }

    return
  }

  a.next.ServeHTTP(rw, req)
}

// Maintenance page templates
func getTemplate(mediaType string, host Host) []byte {
  switch mediaType {

  case "text/html":
    dat, err := os.ReadFile(host.Template)
    if err != nil {
      log.Printf("Template read error: %v", err)
    }
    return []byte(fmt.Sprintf(string(dat),host.Heading,host.Message))

  case "text/plain":
    return []byte(host.Heading + host.Message)

  case "application/json":
    return []byte("{\"message\": \"" + host.Heading + host.Message + "\"}")
  }

  return []byte{}
}
