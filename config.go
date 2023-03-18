package apiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// healthCheck is the default health check call
func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// validPrefix ensures the prefix is correct for a SubRouter
func validPrefix(prefix string, requireTrailingSlash bool) string {
	prefix = strings.ToLower(prefix)

	if !regexp.MustCompile("^[/]?[a-z][a-z0-9]*[/]?$").MatchString(prefix) {
		panic(fmt.Errorf("prefix (%s) is invalid", prefix))
	}
	if prefix[0:] != "/" {
		prefix = "/" + prefix
	}
	if requireTrailingSlash && prefix[:len(prefix)-1] != "/" {
		prefix = prefix + "/"
	}

	return prefix
}

// NewConfig returns a new instance of Config, initialising many components
// using environment variables if defined.  All environment variable values are
// overridable via code.
func NewConfig() *Config {
	c := &Config{
		l:     log.Default(),
		hc:    healthCheck,
		specs: []*APISpecification{},
	}

	// Environment specific details
	c.Port(c.getDefaultableEnvAsInt("PORT", "8080"))
	c.Domain(c.getDefaultableEnv("DOMAIN", "localhost"))
	c.Subdomain(c.getDefaultableEnv("SUBDOMAIN", ""))               // {subdomain:[a-z]+} or www
	c.Scheme(c.getDefaultableEnv("SCHEME", "http"))                 // https
	c.ApiPathPrefix(c.getDefaultableEnv("APIPREFIX", "api"))        // api
	c.HealthcheckPath(c.getDefaultableEnv("HEALTHROUTE", "health")) // health
	c.WriteTimeout(c.getDefaultableEnvAsInt("WRITETIMEOUT", "15"))  // seconds
	c.ReadTimeout(c.getDefaultableEnvAsInt("READTIMEOUT", "15"))    // seconds
	c.ExitTimeout(c.getDefaultableEnvAsInt("EXITTIMEOUT", "10"))    // seconds

	return c
}

// Config allows the Server to be configured as required
type Config struct {
	apiPrefix    string
	domain       string
	exitTimeout  int
	hc           http.HandlerFunc
	healthPath   string
	l            *log.Logger
	port         string
	readTimeout  int
	scheme       string
	specs        []*APISpecification
	subdomain    string
	writeTimeout int
}

// getDefaultableEnv returns the value of an environment variable or default
func (c *Config) getDefaultableEnv(name, defaultValue string) string {
	str := os.Getenv(name)
	if len(str) == 0 {
		str = defaultValue
	}
	return str
}

// getDefaultableEnvAsInt casts environment variable value to an int
func (c *Config) getDefaultableEnvAsInt(name, defaultValue string) int {
	str := c.getDefaultableEnv(name, defaultValue)
	i, err := strconv.Atoi(str)
	if err != nil {
		c.l.Panicf("could not convert (%s) to int, for '%s'", str, name)
	}
	return i
}

// ApiPathPrefix represents the initial path prefix to the API, eg /api/...
func (c *Config) ApiPathPrefix(prefix string) *Config {
	c.apiPrefix = validPrefix(prefix, true)
	return c
}

// Domain represents the domain of the URL
func (c *Config) Domain(domain string) *Config {
	c.domain = domain
	return c
}

// Subdomain represents the subdomain of the URL
func (c *Config) Subdomain(subdomain string) *Config {
	c.subdomain = subdomain
	return c
}

// Scheme represents the scheme of the URL
func (c *Config) Scheme(scheme string) *Config {
	c.scheme = scheme
	return c
}

// HealthcheckPath represents the path from the ApiPathPrefix that identifies the health check
func (c *Config) HealthcheckPath(path string) *Config {
	c.healthPath = validPrefix(path, false)
	return c
}

// WriteTimeout sets the timeout in seconds for writes
func (c *Config) WriteTimeout(seconds int) *Config {
	if seconds < 0 {
		c.l.Panicf("invalid write time specified (%d) seconds", seconds)
	}
	c.writeTimeout = seconds
	return c
}

// ReadTimeout sets the timeout in seconds for reads
func (c *Config) ReadTimeout(seconds int) *Config {
	if seconds < 0 {
		c.l.Panicf("invalid read time specified (%d) seconds", seconds)
	}
	c.readTimeout = seconds
	return c
}

// ExitTimeout sets the timeout in seconds for a graceful exit
func (c *Config) ExitTimeout(seconds int) *Config {
	if seconds < 0 {
		c.l.Panicf("invalid exit time specified (%d) seconds", seconds)
	}
	c.exitTimeout = seconds
	return c
}

// Port allows the port to be specified
func (c *Config) Port(port int) *Config {
	if port < 0 || port > 65000 {
		c.l.Panicf("invalid port %v", port)
	}
	c.port = fmt.Sprint(port)
	return c
}

// Logger allows a specific Logger to be used
func (c *Config) Logger(l *log.Logger) *Config {
	c.l = l
	return c
}

// HealthCheck overrides the default healthcheck
func (c *Config) HealthCheck(hc http.HandlerFunc) *Config {
	c.hc = hc
	return c
}

// NewSpecification appends a new empty API specification, with the given prefix
// Attempting to create a specification with a pre-existing prefix will return
// the existing specification instance
func (c *Config) NewSpecification(prefix string) *APISpecification {
	if len(prefix) == 0 {
		panic(errors.New("prefix must be specfiied"))
	}

	prefix = validPrefix(prefix, true)

	spec := &APISpecification{
		l:      c.l,
		prefix: prefix,
		gets:   []APIPath{},
		posts:  []APIPath{},
	}

	c.specs = append(c.specs, spec)

	return spec
}
