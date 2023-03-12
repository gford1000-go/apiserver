package apiserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// healthCheck is the default health check call
func healthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// NewConfig returns a new instance of Config
func NewConfig() *Config {
	return &Config{
		l:     log.Default(),
		hc:    healthCheck,
		specs: []*APISpecification{},
	}
}

// Config allows the Server to be configured as required
type Config struct {
	l     *log.Logger
	hc    http.HandlerFunc
	specs []*APISpecification
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

	prefix = strings.ToLower(prefix)

	if !regexp.MustCompile("^[/]?[a-z][a-z0-9]*[/]?$").MatchString(prefix) {
		panic(fmt.Errorf("prefix (%s) is invalid", prefix))
	}
	if prefix[0:] != "/" {
		prefix = "/" + prefix
	}
	if prefix[:len(prefix)-1] != "/" {
		prefix = prefix + "/"
	}

	for _, spec := range c.specs {
		if spec.prefix == prefix {
			return spec
		}

	}

	spec := &APISpecification{
		l:      c.l,
		prefix: prefix,
		gets:   []APIPath{},
		posts:  []APIPath{},
	}

	c.specs = append(c.specs, spec)

	return spec
}
