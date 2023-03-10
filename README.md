[![Go Reference](https://pkg.go.dev/badge/github.com/gford1000-go/apiserver.svg)](https://pkg.go.dev/github.com/gford1000-go/apiserver)

apiserver
========

apiserver provides a simple mechanism to initialise a REST based api server which leverages [gorilla mux](https://github.com/gorilla/mux).


## Use

The `Server` initialises via environment variables, establishing two distinct routes that support `GET` and `POST` methods respectively.

The `POST` route additionally restricts request objects to be `application/json` and both only return responses of this content type.

Multiple apis can be registered using `RegisterApiVersion`, but must have a unique path prefix.

The `Server` also contains a rudimentary health check at `/api/health` which is applied by default if one is not specified in the call to `NewServer`.


```go
func addApi(getReqs *mux.Router, postReqs *mux.Router) (string, error) {
	prefix := "/v1/"

	// Simple ping
	api := getReqs.PathPrefix(prefix).Subrouter()
	api.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Encoding", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ping":"pong"})
	})

	return prefix, nil
}

func main() {
	s, err := NewServer(
		[]RegisterApiVersion{
			addApi,
		},
		log.Default(),
		nil,  // Use default health check
	)
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
