package apiserver

import (
	"log"
	"net/http"
	"strings"
)

// PathVariableRetriever provides a mechanism to return a defaultable path value
type PathVariableRetriever func(variableName string, defaultValue interface{}) (interface{}, error)

// APIHandler extends http.HandleFunc to return values of path variables, allowing defaults
type APIHandler func(retriever PathVariableRetriever, w http.ResponseWriter, r *http.Request)

// APIPath describes a given path (which may include variables) and its handler
type APIPath struct {
	path    string
	handler APIHandler
}

// APISpecification describes the structure of an API
type APISpecification struct {
	l      *log.Logger
	prefix string
	gets   []APIPath
	posts  []APIPath
}

// AddGetPath adds a new GET method APIPath to the APISpecification.
// If the path already exists then the APISpecification panics
func (a *APISpecification) AddGetPath(path string, h APIHandler) *APISpecification {

	path = strings.ToLower(path)

	for _, p := range a.gets {
		if path == p.path {
			a.l.Panicf("duplicate GET path: %s", path)
		}
	}

	a.gets = append(a.gets,
		APIPath{
			path:    path,
			handler: h,
		})

	return a
}

// AddGetPath adds a new POST method APIPath to the APISpecification.
// If the path already exists then the APISpecification panics
func (a *APISpecification) AddPostPath(path string, h APIHandler) *APISpecification {

	path = strings.ToLower(path)

	for _, p := range a.posts {
		if path == p.path {
			a.l.Panicf("duplicate POST path: %s", path)
		}
	}

	a.posts = append(a.posts,
		APIPath{
			path:    path,
			handler: h,
		})

	return a
}
