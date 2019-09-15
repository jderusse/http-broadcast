package agent

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jderusse/http-broadcast/pkg/config"
)

func TestNewAgent(t *testing.T) {
	NewAgent(&config.Options{})
}

func TestServe(t *testing.T) {
	server := eventsource.NewServer()
	hubServer := httptest.NewServer(server.Handler("foo"))

	var targetRequest *http.Request
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequest = r
	}))
	// The server has to be closed before the hubServer is closed.
	// Otherwise the hubServer has still an open connection and it can not close.
	defer hubServer.Close()
	defer server.Close()
	defer targetServer.Close()

	s := NewAgent(&config.Options{
		Agent: config.AgentOptions{
			Endpoint: parseSafeURL(targetServer.URL),
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL(hubServer.URL),
		},
	})

	err := s.listen()
	require.NoError(t, err)

	go s.serve()
	defer s.Shutdown()

	event, err := eventsource.NewDecoder(strings.NewReader("id: 123\ndata: {}\n\n")).Decode()
	require.Nil(t, err)
	server.Publish([]string{"foo"}, event)
	time.Sleep(100 * time.Millisecond)
	assert.NotNil(t, targetRequest)
}

func TestShutdown(t *testing.T) {
	server := eventsource.NewServer()
	hubServer := httptest.NewServer(server.Handler("foo"))

	var targetRequest *http.Request
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequest = r
	}))
	// The server has to be closed before the hubServer is closed.
	// Otherwise the hubServer has still an open connection and it can not close.
	defer hubServer.Close()
	defer server.Close()
	defer targetServer.Close()

	s := NewAgent(&config.Options{
		Agent: config.AgentOptions{
			Endpoint: parseSafeURL(targetServer.URL),
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL(hubServer.URL),
		},
	})

	err := s.listen()
	require.NoError(t, err)

	go s.serve()
	s.Shutdown()

	event, err := eventsource.NewDecoder(strings.NewReader("id: 123\ndata: {}\n\n")).Decode()
	require.Nil(t, err)
	server.Publish([]string{"foo"}, event)
	time.Sleep(100 * time.Millisecond)
	assert.Nil(t, targetRequest)
}

func parseSafeURL(urlString string) *url.URL {
	u, _ := url.Parse(urlString)
	return u
}
