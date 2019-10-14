package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/unrolled/secure"
	"golang.org/x/crypto/acme/autocert"

	"github.com/jderusse/http-broadcast/pkg/config"
	"github.com/jderusse/http-broadcast/pkg/server/middleware/forwardedheaders"
	"github.com/jderusse/http-broadcast/pkg/server/middleware/loopguard"
	"github.com/jderusse/http-broadcast/pkg/sync/atomic"
)

// Server listen for incoming request and push them into the hub.
type Server struct {
	httpServer *http.Server
	acmeServer *http.Server
	hubClient  *http.Client
	options    *config.Options

	inShutdown atomic.Bool
	mu         sync.Mutex
	onShutdown []func()
}

// ErrServerClosed is returned by the Server's ListenAndServe methods after a call to Shutdown or Close.
var ErrServerClosed = errors.New("server: Server closed")

// ListenAndServe listens on the TCP socket and then handle requests.
//
// ListenAndServe always returns a non-nil error. After Shutdown or Close,
// the returned error is ErrServerClosed.
func (s *Server) ListenAndServe() error {
	if s.shuttingDown() {
		return ErrServerClosed
	}

	ln, err := s.listen()
	if err != nil {
		return err
	}

	defer ln.Close()

	return s.serve(ln)
}

// RegisterOnShutdown registers a function to call on Shutdown.
// This function should start protocol-specific graceful shutdown,
// but should not wait for shutdown to complete.
func (s *Server) RegisterOnShutdown(f func()) {
	s.mu.Lock()
	s.onShutdown = append(s.onShutdown, f)
	s.mu.Unlock()
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.Load()
}

func (s *Server) listen() (net.Listener, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Debug("server: starting")

	s.httpServer = &http.Server{
		Handler:      s.chainHandlers(),
		ReadTimeout:  s.options.Server.ReadTimeout,
		WriteTimeout: s.options.Server.WriteTimeout,
	}
	s.httpServer.RegisterOnShutdown(func() {
		s.Shutdown()
	})

	ln, err := net.Listen("tcp", s.options.Server.Addr)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{"address": s.options.Server.Addr, "protocol": "http"}).Info("server: listening")

	return ln, nil
}

func (s *Server) serve(ln net.Listener) error {
	if s.isTLS() {
		return s.serveTLS(ln)
	}

	return s.servePlain(ln)
}

func (s *Server) isTLS() bool {
	return len(s.options.Server.TLS.AcmeHosts) > 0 || s.options.Server.TLS.CertFile != ""
}

func (s *Server) serveTLS(ln net.Listener) error {
	if len(s.options.Server.TLS.AcmeHosts) > 0 {
		s.startAcmeServer()
	}

	err := s.httpServer.ServeTLS(ln, s.options.Server.TLS.CertFile, s.options.Server.TLS.KeyFile)
	if err == http.ErrServerClosed {
		return ErrServerClosed
	}

	return err
}

func (s *Server) startAcmeServer() {
	s.mu.Lock()
	defer s.mu.Unlock()

	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.options.Server.TLS.AcmeHosts...),
	}

	if s.options.Server.TLS.AcmeCertDir != "" {
		certManager.Cache = autocert.DirCache(s.options.Server.TLS.AcmeCertDir)
	}

	s.httpServer.TLSConfig = certManager.TLSConfig()

	// Mandatory for Let's Encrypt http-01 challenge
	log.WithFields(log.Fields{"hosts": s.options.Server.TLS.AcmeHosts}).Debug("server: acme server starting")

	s.acmeServer = &http.Server{
		Handler: certManager.HTTPHandler(nil),
	}
	s.acmeServer.RegisterOnShutdown(func() {
		s.Shutdown()
	})

	ln, err := net.Listen("tcp", s.options.Server.TLS.AcmeAddr)
	if err != nil {
		log.Error(errors.Wrap(err, "fail to start acme server"))
	}

	log.WithFields(log.Fields{"hosts": s.options.Server.TLS.AcmeHosts, "addr": s.options.Server.TLS.AcmeAddr}).Info("server: acme server listening")

	go func() {
		defer ln.Close()
		s.acmeServer.Serve(ln)
	}()
}

func (s *Server) servePlain(ln net.Listener) error {
	err := s.httpServer.Serve(ln)
	if err == http.ErrServerClosed {
		return ErrServerClosed
	}

	return err
}

func (s *Server) chainHandlers() http.Handler {
	var h http.Handler
	h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.handle(w, r)
	})

	h = loopguard.NewLoopGuard(s.options.Hub.GuardToken, h)

	if len(s.options.Server.CorsAllowedOrigins) > 0 {
		log.WithFields(log.Fields{"origin": s.options.Server.CorsAllowedOrigins}).Debug("server: configure handlers CORS")

		h = handlers.CORS(
			handlers.AllowCredentials(),
			handlers.AllowedOrigins(s.options.Server.CorsAllowedOrigins),
			handlers.AllowedHeaders([]string{"authorization", "cache-control"}),
		)(h)
	}

	h = secure.New(secure.Options{
		IsDevelopment:         s.options.Debug,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self'",
	}).Handler(h)
	h = handlers.CombinedLoggingHandler(os.Stderr, h)
	h = handlers.RecoveryHandler(
		handlers.RecoveryLogger(log.New()),
		handlers.PrintRecoveryStack(s.options.Debug),
	)(h)
	h, _ = forwardedheaders.NewXForwarded(s.options.Server.Insecure, s.options.Server.TrustedIPs, h)

	return h
}

// Shutdown gracefully shuts down the agent without interrupting any
// active event. Shutdown works by closing stream.
//
// When Shutdown is called, ListenAndServe immediately return
// ErrServerClosed. Make sure the program doesn't exit and waits
// instead for Shutdown to return.
//
// Once Shutdown has been called on a server, it may not be reused;
// future calls to methods such as Serve will return ErrServerClosed.
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shuttingDown() {
		return nil
	}

	log.Debug("server: stopping")
	s.inShutdown.Store(true)

	lnerr := s.closeHTTPServerLocked()

	for _, f := range s.onShutdown {
		go f()
	}

	log.Debug("server: stopped")

	return lnerr
}

func (s *Server) closeHTTPServerLocked() error {
	if s.acmeServer != nil {
		s.acmeServer.Shutdown(context.Background())
	}

	return s.httpServer.Shutdown(context.Background())
}

// NewServer allocates and returns a new Server.
func NewServer(options *config.Options) *Server {
	return &Server{
		options: options,
		hubClient: &http.Client{
			Timeout: options.Hub.Timeout,
		},
	}
}
