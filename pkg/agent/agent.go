package agent

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/r3labs/sse"
	log "github.com/sirupsen/logrus"

	"github.com/jderusse/http-broadcast/pkg/config"
	"github.com/jderusse/http-broadcast/pkg/sync/atomic"
)

// Agent listen for request and dispatch it to a target
type Agent struct {
	events chan *sse.Event

	options *config.Options

	inShutdown atomic.Bool
	mu         sync.Mutex
	doneChan   chan struct{}
	onShutdown []func()
}

// ErrServerClosed is returned by the Agent's ListenAndServe methods after a call to Shutdown or Close.
var ErrServerClosed = errors.New("agent: Server closed")

// ListenAndServe listens on the mercure HUB endpoint and then handle requests on stream.
//
// ListenAndServe always returns a non-nil error. After Shutdown or Close,
// the returned error is ErrServerClosed.
func (a *Agent) ListenAndServe() error {
	if a.shuttingDown() {
		return ErrServerClosed
	}

	if err := a.listen(); err != nil {
		return err
	}

	return a.serve()
}

// RegisterOnShutdown registers a function to call on Shutdown.
// This function should start protocol-specific graceful shutdown,
// but should not wait for shutdown to complete.
func (a *Agent) RegisterOnShutdown(f func()) {
	a.mu.Lock()
	a.onShutdown = append(a.onShutdown, f)
	a.mu.Unlock()
}

func (a *Agent) shuttingDown() bool {
	return a.inShutdown.Load()
}

func (a *Agent) listen() error {
	log.Debug("agent: starting")

	client := sse.NewClient(a.hubURL())
	if a.options.Hub.SubscribeToken != "" {
		client.Headers["Authorization"] = fmt.Sprintf("Bearer %s", a.options.Hub.SubscribeToken)
	}

	client.OnDisconnect(func(c *sse.Client) {
		log.WithFields(log.Fields{"hub": a.hubURL()}).Warn("agent: disconnected")
	})

	ctx, cancel := context.WithCancel(context.Background())

	a.RegisterOnShutdown(func() { cancel() })

	retry := backoff.NewExponentialBackOff()
	retry.MaxInterval = defaultMaxInterval
	retry.MaxElapsedTime = 0
	client.ReconnectStrategy = backoff.WithContext(retry, ctx)

	if err := client.SubscribeChanWithContext(ctx, "", a.events); err != nil {
		return errors.Wrap(err, "subscribe to stream")
	}

	log.WithFields(log.Fields{"hub": a.hubURL()}).Info("agent: listening")

	return nil
}

func (a *Agent) serve() error {
	for {
		select {
		case <-a.getDoneChan():
			return ErrServerClosed
		case event := <-a.events:
			if event != nil && len(event.Data) > 0 {
				go a.replay(string(event.ID), event.Data)
			}
		}
	}
}

func (a *Agent) hubURL() string {
	hubURL, _ := url.Parse(a.options.Hub.Endpoint.String())
	q := hubURL.Query()
	q.Set("topic", a.options.Hub.Topic)
	hubURL.RawQuery = q.Encode()

	return hubURL.String()
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
func (a *Agent) Shutdown() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.shuttingDown() {
		return nil
	}

	log.Debug("agent: stopping")

	a.inShutdown.Store(true)

	a.closeDoneChanLocked()

	for _, f := range a.onShutdown {
		go f()
	}

	log.Debug("agent: stopped")

	return nil
}

func (a *Agent) closeDoneChanLocked() {
	ch := a.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by a.mu.
		close(ch)
	}
}

func (a *Agent) getDoneChan() <-chan struct{} {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.getDoneChanLocked()
}

func (a *Agent) getDoneChanLocked() chan struct{} {
	if a.doneChan == nil {
		a.doneChan = make(chan struct{})
	}

	return a.doneChan
}

// NewAgent allocates and returns a new Agent.
func NewAgent(options *config.Options) *Agent {
	return &Agent{
		events:  make(chan *sse.Event),
		options: options,
	}
}
