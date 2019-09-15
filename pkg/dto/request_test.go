package dto

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestFromHTTP(t *testing.T) {
	req, _ := http.NewRequest("method", "http://endpoint/path", bytes.NewBuffer([]byte("body")))
	r := NewRequestFromHTTP(req)

	assert.Equal(t, "METHOD", r.Method)
	assert.Equal(t, "/path", r.Path)
	assert.Equal(t, "endpoint", r.Host)
	assert.Equal(t, "body", string(r.Body))
}
