package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jderusse/http-broadcast/pkg/config"
)

func TestHandle(t *testing.T) {
	var hubRequest *http.Request
	var hubRequestBody []byte
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hubRequestBody, _ = ioutil.ReadAll(r.Body)
		hubRequest = r
	}))
	defer httpServer.Close()

	s := NewServer(&config.Options{
		Server: config.ServerOptions{
			Addr: ":8002",
		},
		Hub: config.HubOptions{
			Endpoint:   parseSafeURL(httpServer.URL),
			Topic:      "my-topic",
			Target:     "my-target",
			GuardToken: "-",
		},
	})

	ln, err := s.listen()
	require.NoError(t, err)

	defer ln.Close()
	defer s.Shutdown()
	go s.serve(ln)

	resp, err := http.DefaultClient.Post("http://127.0.0.1:8002", "text/plain", bytes.NewBuffer([]byte("Hello")))
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 202, resp.StatusCode)
	require.NotNil(t, hubRequest)
	assert.Equal(t, "POST", hubRequest.Method)
	assert.Equal(t, "application/x-www-form-urlencoded", hubRequest.Header.Get("Content-Type"))

	form, err := url.ParseQuery(string(hubRequestBody))
	require.NoError(t, err)
	hostname, _ := os.Hostname()
	require.NoError(t, err)

	assert.Equal(t, "my-topic", form.Get("topic"))
	assert.Equal(t, "my-target", form.Get("target"))
	assert.Equal(t, `{"Method":"POST","Host":"127.0.0.1:8002","Path":"/","Header":{"Accept-Encoding":["gzip"],"Content-Length":["5"],"Content-Type":["text/plain"],"User-Agent":["Go-http-client/1.1"],"X-Forwarded-Host":["127.0.0.1:8002"],"X-Forwarded-Port":["8002"],"X-Forwarded-Proto":["http"],"X-Forwarded-Server":["`+hostname+`"],"X-Httpbroadcast-Guard":["-"],"X-Real-Ip":["127.0.0.1"]},"Body":"SGVsbG8="}`, form.Get("data"))
}

func TestHandleWithoutHub(t *testing.T) {
	s := NewServer(&config.Options{
		Server: config.ServerOptions{
			Addr: ":8003",
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL("http://127.0.0.1:666"),
		},
	})

	ln, err := s.listen()
	require.NoError(t, err)

	defer ln.Close()
	defer s.Shutdown()
	go s.serve(ln)

	resp, err := http.DefaultClient.Get("http://127.0.0.1:8003")
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 500, resp.StatusCode) // hub in not running
}
