package dto

import (
	"io/ioutil"
	"net/http"
	"strings"
)

// Request is a serializable representation of an http request
type Request struct {
	Method string
	Host   string
	Path   string
	Header http.Header
	Body   []byte
}

// NewRequestFromHTTP allocates and returns a new Request from an http Request.
func NewRequestFromHTTP(r *http.Request) *Request {
	defer r.Body.Close()
	b, _ := ioutil.ReadAll(r.Body)
	request := &Request{
		Method: strings.ToUpper(r.Method),
		Host:   r.Host,
		Path:   r.URL.Path,
		Header: r.Header,
		Body:   b,
	}

	return request
}
