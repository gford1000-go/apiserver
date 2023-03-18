[![Go Reference](https://pkg.go.dev/badge/github.com/gford1000-go/apiserver.svg)](https://pkg.go.dev/github.com/gford1000-go/apiserver)

apiserver
========

apiserver provides a simple mechanism to initialise a REST based api server which leverages [gorilla mux](https://github.com/gorilla/mux).


## Use

The `Server` initialises via environment variables, establishing two distinct routes that support `GET` and `POST` methods respectively.

The `POST` route additionally restricts request objects to be `application/json` and both only return responses of this content type.

The `Config` struct provides a simple way to configure the basic `Server` details and construct the various API specifications.

The `Server` is initialised as part of the `NewServer` function and begins handling requests once `Start` is called.


```go
func main() {

	c := NewConfig()

	// Version 1 of the API returns a single pong for every ping
	c.NewSpecification("v1").
		AddGetPath("/ping", func(vars map[string]string, w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Encoding", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"ping": "pong"})
		})

	// Version 2 allows one or more pongs, specified by the request
	c.NewSpecification("v2").
		AddGetPath("/ping/{num_pings}", func(vars map[string]string, w http.ResponseWriter, r *http.Request) {
			s, ok := vars["num_pings"]
			if !ok {
				s = "1"
			}
			n, err := strconv.Atoi(s)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid number of pings: '%s'", s), http.StatusBadRequest)
				return
			}
			if n < 1 {
				http.Error(w, fmt.Sprintf("Invalid number of pings: '%d'", n), http.StatusBadRequest)
				return
			}
			w.Header().Add("Content-Encoding", "application/json")
			pongs := []string{}
			for i := 0; i < n; i++ {
				pongs = append(pongs, "pong")
			}
			json.NewEncoder(w).Encode(map[string][]string{"ping": pongs})
		})

	s, err := NewServer(c)
	if err != nil {
		log.Fatal(err)
	}

	s.Start()

	os.Exit(0)
}
```

## How?

The command line is all you need.

```
go get github.com/gford1000-go/apiserver
```
