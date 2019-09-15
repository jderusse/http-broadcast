package agent

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jderusse/http-broadcast/pkg/config"
)

func TestReplay(t *testing.T) {
	var targetRequest *http.Request
	var targetRequestBody []byte
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequestBody, _ = ioutil.ReadAll(r.Body)
		targetRequest = r
	}))
	defer targetServer.Close()

	s := NewAgent(&config.Options{
		Agent: config.AgentOptions{
			Endpoint: parseSafeURL(targetServer.URL),
		},
	})

	s.replay("random", []byte(`{"Method":"POST","Host":"127.0.0.1:8765","Path":"/","Header":{"Accept-Encoding":["gzip"],"Content-Length":["5"],"Content-Type":["text/plain"],"User-Agent":["Go-http-client/1.1"],"X-Forwarded-Host":["127.0.0.1:8765"],"X-Forwarded-Port":["8765"],"X-Forwarded-Proto":["http"],"X-Forwarded-Server":["FooBar"],"X-Real-Ip":["127.0.0.1"]},"Body":"SGVsbG8="}`))
	require.NotNil(t, targetRequest)
	assert.Equal(t, "POST", targetRequest.Method)
	assert.Equal(t, "text/plain", targetRequest.Header.Get("Content-Type"))
	assert.Equal(t, "Hello", string(targetRequestBody))
}
