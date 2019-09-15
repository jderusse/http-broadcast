package config

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptionsFormNew(t *testing.T) {
	testEnv := map[string]string{
		"AGENT_ENDPOINT":              "http://agent/",
		"AGENT_RETRY_DELAY":           "1m",
		"DEBUG":                       "1",
		"HUB_ENDPOINT":                "http://hub/",
		"HUB_GUARD_TOKEN":             "guard_token",
		"HUB_PUBLISH_TOKEN":           "pub_token",
		"HUB_SUBSCRIBE_TOKEN":         "sub_token",
		"HUB_TIMEOUT":                 "1m",
		"HUB_TOKEN":                   "token",
		"HUB_TOPIC":                   "HUB_TOPIC",
		"LOG_FORMAT":                  "json",
		"LOG_LEVEL":                   "warn",
		"SERVER_ADDR":                 "0.0.0.0:81",
		"SERVER_CORS_ALLOWED_ORIGINS": "example.com,bar.com",
		"SERVER_INSECURE":             "1",
		"SERVER_READ_TIMEOUT":         "1m",
		"SERVER_TLS_ACME_ADDR":        ":81",
		"SERVER_TLS_ACME_CERT_DIR":    "/tmp",
		"SERVER_TLS_ACME_HOSTS":       "example.com",
		"SERVER_TLS_CERT_FILE":        "/tmp/cert",
		"SERVER_TLS_KEY_FILE":         "/tmp/key",
		"SERVER_TRUSTED_IPS":          "127.0.0.1,1.2.3.4",
		"SERVER_WRITE_TIMEOUT":        "1m",
	}
	for k, v := range testEnv {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	opts, err := NewOptionsFromEnv()
	assert.Equal(t, &Options{
		Debug: true,
		Agent: AgentOptions{
			Endpoint:   parseSafeURL("http://agent/"),
			RetryDelay: 1 * time.Minute,
		},
		Hub: HubOptions{
			Endpoint:       parseSafeURL("http://hub/"),
			GuardToken:     "guard_token",
			PublishToken:   "pub_token",
			SubscribeToken: "sub_token",
			Timeout:        1 * time.Minute,
			Topic:          "HUB_TOPIC",
		},
		Server: ServerOptions{
			Addr:               "0.0.0.0:81",
			ReadTimeout:        1 * time.Minute,
			WriteTimeout:       1 * time.Minute,
			CorsAllowedOrigins: []string{"example.com", "bar.com"},
			Insecure:           true,
			TrustedIPs:         []string{"127.0.0.1", "1.2.3.4"},
			TLS: TLSServerOptions{
				AcmeAddr:    ":81",
				AcmeCertDir: "/tmp",
				AcmeHosts:   []string{"example.com"},
				CertFile:    "/tmp/cert",
				KeyFile:     "/tmp/key",
			},
		},
	}, opts)
	assert.Nil(t, err)
}

func TestMissingEnv(t *testing.T) {
	_, err := NewOptionsFromEnv()
	assert.EqualError(t, err, "the following environment variable must be defined: [HUB_ENDPOINT SERVER_ADDR/AGENT_ENDPOINT]")
}

func TestMissingKeyFile(t *testing.T) {
	os.Setenv("SERVER_TLS_CERT_FILE", "foo")
	defer os.Unsetenv("SERVER_TLS_CERT_FILE")

	_, err := NewOptionsFromEnv()
	assert.EqualError(t, err, "the following environment variable must be defined: [HUB_ENDPOINT SERVER_ADDR/AGENT_ENDPOINT SERVER_TLS_KEY_FILE]")
}

func TestMissingCertFile(t *testing.T) {
	os.Setenv("SERVER_TLS_KEY_FILE", "foo")
	defer os.Unsetenv("SERVER_TLS_KEY_FILE")

	_, err := NewOptionsFromEnv()
	assert.EqualError(t, err, "the following environment variable must be defined: [HUB_ENDPOINT SERVER_ADDR/AGENT_ENDPOINT SERVER_TLS_CERT_FILE]")
}

func TestInvalidDuration(t *testing.T) {
	vars := []string{"AGENT_RETRY_DELAY", "HUB_TIMEOUT", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT"}
	for _, elem := range vars {
		os.Setenv(elem, "1 MN (invalid)")
		defer os.Unsetenv(elem)
		_, err := NewOptionsFromEnv()
		assert.EqualError(t, err, elem+": time: unknown unit  MN (invalid) in duration 1 MN (invalid)")

		os.Unsetenv(elem)
	}
}

func TestInvalidUrl(t *testing.T) {
	vars := []string{"AGENT_ENDPOINT", "HUB_ENDPOINT"}
	for _, elem := range vars {
		os.Setenv(elem, "http://[::1]%23")
		defer os.Unsetenv(elem)
		_, err := NewOptionsFromEnv()
		assert.EqualError(t, err, elem+": parse http://[::1]%23: invalid port \"%23\" after host")

		os.Unsetenv(elem)
	}
}

func parseSafeURL(urlString string) *url.URL {
	u, _ := url.Parse(urlString)
	return u
}

func TestFallbackHub(t *testing.T) {
	var providerTests = []struct {
		endpoint         string
		topic            string
		expectedEndpoint string
		expectedTopic    string
	}{
		{"http://example.com", "topic", "http://example.com", "topic"},
		{"http://example.com", "", "http://example.com", "http-broadcast"},
		{"http://example.com?topic=bar", "", "http://example.com", "bar"},
		{"http://example.com?topic=bar", "baz", "http://example.com", "baz"},
	}

	os.Setenv("SERVER_ADDR", ":http")
	os.Setenv("HUB_TOKEN", "token")
	defer os.Unsetenv("SERVER_ADDR")
	defer os.Unsetenv("HUB_TOPIC")
	defer os.Unsetenv("HUB_TOKEN")
	for _, test := range providerTests {
		os.Setenv("HUB_ENDPOINT", test.endpoint)
		os.Setenv("HUB_TOPIC", test.topic)

		opts, err := NewOptionsFromEnv()
		require.Nil(t, err)
		assert.Equal(t, parseSafeURL(test.expectedEndpoint), opts.Hub.Endpoint)
		assert.Equal(t, test.expectedTopic, opts.Hub.Topic)
	}
}
