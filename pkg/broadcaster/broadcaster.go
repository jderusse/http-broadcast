package broadcaster

import (
	"os"
	"os/signal"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/jderusse/http-broadcast/pkg/agent"
	"github.com/jderusse/http-broadcast/pkg/config"
	"github.com/jderusse/http-broadcast/pkg/server"
)

// Broadcaster is responsible of starting and stoping a Server and a Agent
type Broadcaster struct {
	agent  *agent.Agent
	server *server.Server
	wg     sync.WaitGroup
}

// Run starts the Server and the Agent
func (b *Broadcaster) Run() {
	log.Debug("Broadcaster: starting")

	if b.agent != nil {
		b.wg.Add(1)

		go func() {
			if err := b.agent.ListenAndServe(); err != nil {
				if err != agent.ErrServerClosed {
					log.Error(err)
				}
				b.Stop()
			}
			b.wg.Done()
		}()
	}

	if b.server != nil {
		b.wg.Add(1)

		go func() {
			if err := b.server.ListenAndServe(); err != nil {
				if err != server.ErrServerClosed {
					log.Error(err)
				}
				b.Stop()
			}
			b.wg.Done()
		}()
	}

	b.wg.Wait()

	log.Debug("Broadcaster: stopped")
	b.Stop()
}

// Stop stops the Server and the Agent
func (b *Broadcaster) Stop() {
	if b.agent != nil {
		b.agent.Shutdown()
	}

	if b.server != nil {
		b.server.Shutdown()
	}
}

func (b *Broadcaster) handleShutdown() {
	if b.server != nil {
		b.server.RegisterOnShutdown(func() {
			b.Stop()
		})
	}

	if b.agent != nil {
		b.agent.RegisterOnShutdown(func() {
			b.Stop()
		})
	}

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		for {
			<-sigint
			log.Infoln("My Baby Shot Me Down")
			b.Stop()
		}
	}()
}

// NewBroadcasterFromEnv allocates and returns a new Broadcaster using the configuration set in env vars.
func NewBroadcasterFromEnv() (*Broadcaster, error) {
	options, err := config.NewOptionsFromEnv()
	if err != nil {
		return nil, err
	}

	return NewBroadcaster(options), nil
}

// NewBroadcaster allocates and returns a new Broadcaster.
func NewBroadcaster(options *config.Options) *Broadcaster {
	var (
		a *agent.Agent
		s *server.Server
	)

	if options.Server.Addr != "" {
		s = server.NewServer(options)
	}

	if nil != options.Agent.Endpoint {
		a = agent.NewAgent(options)
	}

	b := &Broadcaster{
		agent:  a,
		server: s,
	}
	b.handleShutdown()

	return b
}
