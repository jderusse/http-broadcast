package agent

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/r3labs/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jderusse/http-broadcast/pkg/config"
)

var u string
var srv *sse.Server
var server *httptest.Server

func newServer() *sse.Server {
	srv = sse.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", srv.HTTPHandler)
	server = httptest.NewServer(mux)

	srv.CreateStream("foo")

	return srv
}

func cleanup() {
	server.CloseClientConnections()
	server.Close()
	srv.Close()
}

func TestNewAgent(t *testing.T) {
	NewAgent(&config.Options{})
}

func TestServe(t *testing.T) {
	newServer()
	defer cleanup()

	var targetRequest atomic.Value
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequest.Store(r)
	}))
	defer targetServer.Close()

	s := NewAgent(&config.Options{
		Agent: config.AgentOptions{
			Endpoint: parseSafeURL(targetServer.URL),
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL(server.URL + "/events?stream=foo"),
		},
	})

	err := s.listen()
	require.NoError(t, err)

	go s.serve()
	defer s.Shutdown()

	srv.Publish("foo", &sse.Event{ID: []byte("123"), Data: []byte("{}")})
	time.Sleep(100 * time.Millisecond)

	v, _ := targetRequest.Load().(*http.Request)
	assert.NotNil(t, v)
}

func TestShutdown(t *testing.T) {
	newServer()
	defer cleanup()

	var targetRequest atomic.Value
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequest.Store(r)
	}))
	defer targetServer.Close()

	s := NewAgent(&config.Options{
		Agent: config.AgentOptions{
			Endpoint: parseSafeURL(targetServer.URL),
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL(server.URL + "/events?stream=foo"),
		},
	})

	err := s.listen()
	require.NoError(t, err)

	go s.serve()
	s.Shutdown()

	srv.Publish("foo", &sse.Event{ID: []byte("123"), Data: []byte("{}")})
	time.Sleep(100 * time.Millisecond)

	v, _ := targetRequest.Load().(*http.Request)
	assert.Nil(t, v)
}

func parseSafeURL(urlString string) *url.URL {
	u, _ := url.Parse(urlString)
	return u
}
