package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/donovanhide/eventsource"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/jderusse/http-broadcast/pkg/config"
	"github.com/jderusse/http-broadcast/pkg/sync/atomic"
)

// Agent listen for request and dispatch it to a target
type Agent struct {
	stream  *eventsource.Stream
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

	hubRequest, err := http.NewRequest("GET", a.hubURL(), nil)

	if err != nil {
		return err
	}

	if a.options.Hub.SubscribeToken != "" {
		hubRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.options.Hub.SubscribeToken))
	}

	retry := backoff.NewExponentialBackOff()
	retry.MaxInterval = 5 * time.Second
	retry.MaxElapsedTime = 0
	ctx, cancel := context.WithCancel(context.Background())

	a.RegisterOnShutdown(func() { cancel() })

	err = backoff.Retry(func() error {
		stream, err := eventsource.SubscribeWith("", http.DefaultClient, hubRequest)
		if err == nil {
			a.mu.Lock()
			a.stream = stream
			a.mu.Unlock()

			return nil
		}

		return err
	}, backoff.WithContext(retry, ctx))

	if err != nil {
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
		case event := <-a.stream.Events:
			if event != nil {
				go a.replay(event.Id(), []byte(event.Data()))
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

	lnerr := a.closeStreamLocked()
	a.closeDoneChanLocked()

	for _, f := range a.onShutdown {
		go f()
	}

	log.Debug("agent: stopped")

	return lnerr
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

func (a *Agent) closeStreamLocked() error {
	if a.stream != nil {
		a.stream.Close()
	}

	return nil
}

// NewAgent allocates and returns a new Agent.
func NewAgent(options *config.Options) *Agent {
	return &Agent{
		options: options,
	}
}
