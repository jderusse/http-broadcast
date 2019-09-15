package server

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jderusse/http-broadcast/pkg/config"
)

func TestNewServer(t *testing.T) {
	NewServer(&config.Options{})
}

func TestServe(t *testing.T) {
	s := NewServer(&config.Options{
		Server: config.ServerOptions{
			Addr: ":8004",
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL("http://127.0.0.1:1234"),
		},
	})

	ln, err := s.listen()
	require.NoError(t, err)

	defer ln.Close()
	defer s.Shutdown()
	go s.serve(ln)

	resp, err := http.DefaultClient.Get("http://127.0.0.1:8004")
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}

func TestAcme(t *testing.T) {
	s := NewServer(&config.Options{
		Server: config.ServerOptions{
			Addr: ":8005",
			TLS: config.TLSServerOptions{
				AcmeAddr:  ":9000",
				AcmeHosts: []string{"example.com"},
			},
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL("http://127.0.0.1:1234"),
		},
	})

	ln, err := s.listen()
	require.NoError(t, err)

	defer ln.Close()
	defer s.Shutdown()
	go s.serve(ln)

	// wait for acme server to start
	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get("http://127.0.0.1:9000")
	assert.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 302, resp.StatusCode)

	resp, err = client.Get("http://127.0.0.1:9000/.well-known/acme-challenge/does-not-exists")
	assert.Nil(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestShutdown(t *testing.T) {
	s := NewServer(&config.Options{
		Server: config.ServerOptions{
			Addr: ":8006",
		},
		Hub: config.HubOptions{
			Endpoint: parseSafeURL("http://127.0.0.1:1234"),
		},
	})

	ln, err := s.listen()
	require.NoError(t, err)

	defer ln.Close()
	go s.serve(ln)
	s.Shutdown()

	resp, err := http.DefaultClient.Get("http://127.0.0.1:8006")
	assert.NotNil(t, err)
	assert.Nil(t, resp)
}

func parseSafeURL(urlString string) *url.URL {
	u, _ := url.Parse(urlString)
	return u
}
