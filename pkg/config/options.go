package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Options stores the Broadcaster's options
type Options struct {
	Debug  bool
	Agent  AgentOptions
	Hub    HubOptions
	Server ServerOptions
}

// ServerOptions stores the Server's options
type ServerOptions struct {
	Addr               string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	CorsAllowedOrigins []string
	Insecure           bool
	TrustedIPs         []string
	TLS                TLSServerOptions
}

// TLSServerOptions stores the Server's TLS options
type TLSServerOptions struct {
	AcmeAddr    string
	AcmeCertDir string
	AcmeHosts   []string
	CertFile    string
	KeyFile     string
}

// AgentOptions stores the Agent options
type AgentOptions struct {
	Endpoint   *url.URL
	RetryDelay time.Duration
}

// HubOptions stores the Hub options
type HubOptions struct {
	Endpoint       *url.URL
	GuardToken     string
	PublishToken   string
	SubscribeToken string
	Timeout        time.Duration
	Topic          string
	Target         string
}

// NewOptionsFromEnv creates a new option instance from environment
// It returns an error if mandatory env env vars are missing
//nolint:gocognit
func NewOptionsFromEnv() (*Options, error) {
	agentRetryDelay, err := time.ParseDuration(getEnv("AGENT_RETRY_DELAY", "60s"))
	if err != nil {
		return nil, errors.Wrap(err, "AGENT_RETRY_DELAY")
	}

	hubTimeout, err := time.ParseDuration(getEnv("HUB_TIMEOUT", "5s"))
	if err != nil {
		return nil, errors.Wrap(err, "HUB_TIMEOUT")
	}

	readTimeout, err := time.ParseDuration(getEnv("SERVER_READ_TIMEOUT", "0s"))
	if err != nil {
		return nil, errors.Wrap(err, "SERVER_READ_TIMEOUT")
	}

	writeTimeout, err := time.ParseDuration(getEnv("SERVER_WRITE_TIMEOUT", "0s"))
	if err != nil {
		return nil, errors.Wrap(err, "SERVER_WRITE_TIMEOUT")
	}

	agentEndpoint, err := parseURL(os.Getenv("AGENT_ENDPOINT"))
	if err != nil {
		return nil, errors.Wrap(err, "AGENT_ENDPOINT")
	}

	hubEndpoint, err := parseURL(os.Getenv("HUB_ENDPOINT"))
	if err != nil {
		return nil, errors.Wrap(err, "HUB_ENDPOINT")
	}

	hubTopic := os.Getenv("HUB_TOPIC")
	if hubEndpoint != nil {
		if hubTopic == "" {
			hubTopic = hubEndpoint.Query().Get("topic")
		}

		q := hubEndpoint.Query()
		q.Del("topic")
		hubEndpoint.RawQuery = q.Encode()
	}

	if hubTopic == "" {
		hubTopic = "http-broadcast"
	}

	hubTarget := os.Getenv("HUB_TARGET")
	if hubEndpoint != nil {
		if hubTarget == "" {
			hubTarget = hubEndpoint.Query().Get("target")
		}

		q := hubEndpoint.Query()
		q.Del("target")
		hubEndpoint.RawQuery = q.Encode()
	}

	options := &Options{
		Debug: getEnv("DEBUG", "0") == "1",
		Agent: AgentOptions{
			Endpoint:   agentEndpoint,
			RetryDelay: agentRetryDelay,
		},
		Hub: HubOptions{
			Endpoint:       hubEndpoint,
			GuardToken:     getEnv("HUB_GUARD_TOKEN", hubTopic),
			PublishToken:   getEnv("HUB_PUBLISH_TOKEN", os.Getenv("HUB_TOKEN")),
			SubscribeToken: getEnv("HUB_SUBSCRIBE_TOKEN", os.Getenv("HUB_TOKEN")),
			Timeout:        hubTimeout,
			Topic:          hubTopic,
			Target:         hubTarget,
		},
		Server: ServerOptions{
			Addr:               os.Getenv("SERVER_ADDR"),
			ReadTimeout:        readTimeout,
			WriteTimeout:       writeTimeout,
			CorsAllowedOrigins: splitVar(os.Getenv("SERVER_CORS_ALLOWED_ORIGINS")),
			Insecure:           getEnv("SERVER_INSECURE", getEnv("DEBUG", "0")) == "1",
			TrustedIPs:         splitVar(os.Getenv("SERVER_TRUSTED_IPS")),
			TLS: TLSServerOptions{
				AcmeAddr:    getEnv("SERVER_TLS_ACME_ADDR", ":http"),
				AcmeCertDir: os.Getenv("SERVER_TLS_ACME_CERT_DIR"),
				AcmeHosts:   splitVar(os.Getenv("SERVER_TLS_ACME_HOSTS")),
				CertFile:    os.Getenv("SERVER_TLS_CERT_FILE"),
				KeyFile:     os.Getenv("SERVER_TLS_KEY_FILE"),
			},
		},
	}

	missingEnv := []string{}
	if options.Hub.Endpoint == nil {
		missingEnv = append(missingEnv, "HUB_ENDPOINT")
	}

	if len(options.Hub.Topic) == 0 {
		missingEnv = append(missingEnv, "HUB_TOPIC")
	}

	if len(options.Server.Addr) == 0 && nil == options.Agent.Endpoint {
		missingEnv = append(missingEnv, "SERVER_ADDR/AGENT_ENDPOINT")
	}

	if len(options.Server.Addr) != 0 && len(options.Hub.PublishToken) == 0 {
		missingEnv = append(missingEnv, "HUB_PUBLISH_TOKEN/HUB_TOKEN")
	}

	if len(options.Server.TLS.CertFile) != 0 && len(options.Server.TLS.KeyFile) == 0 {
		missingEnv = append(missingEnv, "SERVER_TLS_KEY_FILE")
	}

	if len(options.Server.TLS.KeyFile) != 0 && len(options.Server.TLS.CertFile) == 0 {
		missingEnv = append(missingEnv, "SERVER_TLS_CERT_FILE")
	}

	if len(missingEnv) > 0 {
		return nil, fmt.Errorf("the following environment variable must be defined: %s", missingEnv)
	}

	return options, nil
}

func getEnv(k string, d string) string {
	v := os.Getenv(k)
	if v != "" {
		return v
	}

	return d
}

func splitVar(v string) []string {
	if v == "" {
		return []string{}
	}

	return strings.Split(v, ",")
}

func parseURL(v string) (*url.URL, error) {
	if v == "" {
		return nil, nil
	}

	return url.Parse(v)
}
