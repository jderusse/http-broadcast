package broadcaster

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jderusse/http-broadcast/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewBroadcaster(t *testing.T) {
	c := &config.Options{}

	NewBroadcaster(c)
}

func TestNewBroadcasterFromEnv(t *testing.T) {
	os.Setenv("SERVER_ADDR", ":http")
	os.Setenv("HUB_TOKEN", "token")
	os.Setenv("HUB_ENDPOINT", "http://hub")
	defer os.Unsetenv("SERVER_ADDR")
	defer os.Unsetenv("HUB_TOKEN")
	defer os.Unsetenv("HUB_ENDPOINT")

	b, err := NewBroadcasterFromEnv()
	assert.NotNil(t, b)
	assert.Nil(t, err)
}

func TestNewHubFromEnvError(t *testing.T) {
	b, err := NewBroadcasterFromEnv()
	assert.Nil(t, b)
	assert.NotNil(t, err)
}

func TestRun(t *testing.T) {
	b := NewBroadcaster(&config.Options{
		Server: config.ServerOptions{
			Addr: ":8001",
		},
	})

	go b.Run()
	defer b.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.DefaultClient.Get("http://127.0.0.1:8001")
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}
