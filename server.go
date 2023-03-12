package apiserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Server initialises and runs a http.Server to handle http requests
type Server struct {
	srv          *http.Server
	logger       *log.Logger
	readTimeout  int
	writeTimeout int
	exitTimeout  int
}

// getDefaultableEnv returns the value of an environment variable or default
func (s *Server) getDefaultableEnv(name, defaultValue string) string {
	str := os.Getenv(name)
	if len(str) == 0 {
		str = defaultValue
	}
	return str
}

// getDefaultableEnvAsInt casts environment variable value to an int
func (s *Server) getDefaultableEnvAsInt(name, defaultValue string) int {
	str := s.getDefaultableEnv(name, defaultValue)
	i, err := strconv.Atoi(str)
	if err != nil {
		s.logger.Panicf("could not convert (%s) to int, for '%s'", str, name)
	}
	return i
}

// init performs default initialisation and then applies
// api version routes and handling
func (s *Server) init(config *Config) error {

	s.logger.Println("initialising")

	// Environment specific details
	port := s.getDefaultableEnv("PORT", "8080")                     // 8080
	domain := s.getDefaultableEnv("DOMAIN", "localhost")            // example.com
	subdomain := s.getDefaultableEnv("SUBDOMAIN", "")               // {subdomain:[a-z]+} or www
	scheme := s.getDefaultableEnv("SCHEME", "http")                 // https
	apiPrefix := s.getDefaultableEnv("APIPREFIX", "api")            // api
	healthPath := s.getDefaultableEnv("HEALTHROUTE", "health")      // health
	s.writeTimeout = s.getDefaultableEnvAsInt("WRITETIMEOUT", "15") // seconds
	s.readTimeout = s.getDefaultableEnvAsInt("READTIMEOUT", "15")   // seconds
	s.exitTimeout = s.getDefaultableEnvAsInt("EXITTIMEOUT", "10")   // seconds

	r := mux.NewRouter()

	// Subroute based on domain and subdomain if specified
	d := r
	if strings.ToLower(domain) != "localhost" {
		d = r.Host(fmt.Sprintf("%s.%s", subdomain, domain)).Subrouter()
	}

	// Api only, with only GET and POST requests processed, where
	// POST requests must provide JSON based request objects
	get := d.
		PathPrefix(fmt.Sprintf("/%s/", apiPrefix)).
		Methods("GET").
		Schemes(scheme).Subrouter()

	post := d.
		PathPrefix(fmt.Sprintf("/%s/", apiPrefix)).
		HeadersRegexp("Content-Type", "application/json").
		Methods("POST").
		Schemes(scheme).Subrouter()

	registered := map[string]bool{}
	for _, spec := range config.specs {
		if _, ok := registered[spec.prefix]; ok {
			return fmt.Errorf("attempt to register %s twice", spec.prefix)
		} else {
			registered[spec.prefix] = true
		}

		s.addSpecification(spec, get, post)
	}

	// Add healthcheck
	get.HandleFunc(fmt.Sprintf("/%s", healthPath), config.hc).Methods("GET")

	// Bind to a port and pass our router in
	s.srv = &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("127.0.0.1:%s", port),
		WriteTimeout: time.Duration(s.writeTimeout) * time.Second,
		ReadTimeout:  time.Duration(s.readTimeout) * time.Second,
	}

	return nil
}

// addMethodSpecification creates a subrouter for the specified prefix and applies
// the paths to it in the order defined
func (s *Server) addMethodSpecification(prefix string, r *mux.Router, paths []APIPath) {

	if len(paths) > 0 {
		api := r.PathPrefix(prefix).Subrouter()

		for _, path := range paths {
			api.HandleFunc(path.path, func(w http.ResponseWriter, req *http.Request) {
				path.handler(mux.Vars(req), w, req)
			})
		}
	}
}

// addSpecification creates GET and POST api handlers
func (s *Server) addSpecification(spec *APISpecification, getReqs *mux.Router, postReqs *mux.Router) {
	s.addMethodSpecification(spec.prefix, getReqs, spec.gets)
	s.addMethodSpecification(spec.prefix, postReqs, spec.posts)
}

// Start causes the Server to start handling requests
func (s *Server) Start() {

	log.Println("starting")

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := s.srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	s.logger.Println("stopping...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.exitTimeout))
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	go s.srv.Shutdown(ctx)

	// Wait for other services to finalize based on context cancellation.
	<-ctx.Done()

	s.logger.Println("stopped")
}

// NewServer returns a non-started, initialised Server instance
func NewServer(config Config) (*Server, error) {

	s := &Server{logger: config.l}

	// Initise once
	if err := s.init(&config); err != nil {
		return nil, err
	}
	return s, nil
}
