package apiserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Server initialises and runs a http.Server to handle http requests
type Server struct {
	exitTimeout int
	logger      *log.Logger
	srv         *http.Server
}

// init performs default initialisation and then applies
// api version routes and handling
func (s *Server) init(config *Config) error {

	s.logger.Println("initialising")

	// Store exitTimeout
	s.exitTimeout = config.exitTimeout

	r := mux.NewRouter()

	// Subroute based on domain and subdomain if specified
	d := r
	if strings.ToLower(config.domain) != "localhost" {
		d = r.Host(fmt.Sprintf("%s.%s", config.subdomain, config.domain)).Subrouter()
	}

	// Api only, with only GET and POST requests processed, where
	// POST requests must provide JSON based request objects
	get := d.
		PathPrefix(config.apiPrefix).
		Methods("GET").
		Schemes(config.scheme).Subrouter()

	post := d.
		PathPrefix(config.apiPrefix).
		HeadersRegexp("Content-Type", "application/json").
		Methods("POST").
		Schemes(config.scheme).Subrouter()

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
	get.HandleFunc(fmt.Sprintf("/%s", config.healthPath), config.hc).Methods("GET")

	// Bind to a port and pass our router in
	s.srv = &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("127.0.0.1:%s", config.port),
		WriteTimeout: time.Duration(config.writeTimeout) * time.Second,
		ReadTimeout:  time.Duration(config.readTimeout) * time.Second,
	}

	return nil
}

// getRetriever creates a path variable retriever function instance, that can apply the
// specified default value if the path variable is missing.
// If the path variable exists, then an attempt to convert the value of the path variable is made,
// into the type of the default value.
//   If the default value is nil, then the value of the path variable is returned as a string
//   If the default value is a string, then the value is returned as a string
//   If the default value is of type int, *int, int32, *int32, int64, *int64, then a conversion is attempted
//   For all other types, an error is raised
func (s *Server) getRetriever(req *http.Request) PathVariableRetriever {
	m := mux.Vars(req)

	return func(variableName string, defaultValue interface{}) (interface{}, error) {
		variableName = strings.ToLower(variableName)
		s, ok := m[variableName]
		if !ok {
			return defaultValue, nil
		}
		if defaultValue == nil {
			return s, nil
		}
		switch defaultValue.(type) {
		case string:
			return s, nil
		case int:
			return strconv.Atoi(s)
		case *int:
			i, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}
			return &i, nil
		case int32:
			return strconv.ParseInt(s, 10, 32)
		case *int32:
			i, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				return nil, err
			}
			return &i, nil
		case int64:
			return strconv.ParseInt(s, 10, 64)
		case *int64:
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, err
			}
			return &i, nil
		default:
			return nil, fmt.Errorf("unable to convert to type %v", reflect.TypeOf(defaultValue))
		}
	}
}

// addMethodSpecification creates a subrouter for the specified prefix and applies
// the paths to it in the order defined
func (s *Server) addMethodSpecification(prefix string, r *mux.Router, paths []APIPath) {

	if len(paths) > 0 {
		api := r.PathPrefix(prefix).Subrouter()

		for _, path := range paths {
			api.HandleFunc(path.path, func(w http.ResponseWriter, req *http.Request) {
				path.handler(s.getRetriever(req), w, req)
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
