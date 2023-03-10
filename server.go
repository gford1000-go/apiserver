package apiserver

import (
	"context"
	"encoding/json"
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

// RegisterApiVersion allows GET and POST path support,
// returning the prefix to the api (either /vXXX/ or just vX), which
// must be unique across all registration
type RegisterApiVersion func(get, post *mux.Router) (string, error)

// Server initialises and runs a http.Server to handle http requests
type Server struct {
	srv          *http.Server
	logger       *log.Logger
	readTimeout  int
	writeTimeout int
	exitTimeout  int
}

func (s *Server) NotAllowed(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Allowed", http.StatusNotFound)
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) getDefaultableEnv(name, defaultValue string) string {
	str := os.Getenv(name)
	if len(str) == 0 {
		str = defaultValue
	}
	return str
}

func (s *Server) getDefaultableEnvAsInt(name, defaultValue string) int {
	str := s.getDefaultableEnv(name, defaultValue)
	i, err := strconv.Atoi(str)
	if err != nil {
		s.logger.Panicf("could not convert (%s) to int, for '%s'", str, name)
	}
	return i
}

// Init performs default initialisation and then applies
// api version routes and handling
func (s *Server) Init(apis []RegisterApiVersion) error {

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
		Methods("POST", "GET").
		Schemes(scheme).Subrouter()

	post := d.
		PathPrefix(fmt.Sprintf("/%s/", apiPrefix)).
		HeadersRegexp("Content-Type", "application/json").
		Methods("POST", "GET").
		Schemes(scheme).Subrouter()

	// Initialise implementations of the API for each version.
	// Each registered version implementation must have a unique
	// prefix
	registered := map[string]bool{}
	for _, api := range apis {
		if version, err := api(get, post); err != nil {
			return err
		} else {
			// Prefix must look like /{version}/
			if version[0:] != "/" {
				version = "/" + version
			}
			if version[:len(version)-1] != "/" {
				version = version + "/"
			}
			if _, ok := registered[version]; ok {
				return fmt.Errorf("attempt to register %s twice", version)
			} else {
				registered[version] = true
			}
		}
	}

	// Add healthcheck
	get.HandleFunc(fmt.Sprintf("/%s", healthPath), s.HealthCheck).Methods("GET")

	// Bind to a port and pass our router in
	s.srv = &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("127.0.0.1:%s", port),
		WriteTimeout: time.Duration(s.writeTimeout) * time.Second,
		ReadTimeout:  time.Duration(s.readTimeout) * time.Second,
	}

	return nil
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
func NewServer(apis []RegisterApiVersion, logger *log.Logger) (*Server, error) {
	s := &Server{logger: logger}
	if err := s.Init(apis); err != nil {
		return nil, err
	}
	return s, nil
}
