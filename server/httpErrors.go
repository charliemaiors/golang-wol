package server

import (
	"errors"
	"net/http"
)

type MethodNotAllowed struct {
}

type NotFound struct {
}

func (mn *MethodNotAllowed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, errors.New("Method not Allowed"), 405)
}

func (nf *NotFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, errors.New("Resource not Found"), 404)
}
