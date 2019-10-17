package loopguard

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	guardHeader = "X-HttpBroadcast-Guard"
)

// LoopGuard is an HTTP handler wrapper that prevent infinite http requests
// by injecting a marker in the request
type LoopGuard struct {
	token string
	next  http.Handler
}

func (l *LoopGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		tokens   []string
		contains = false
	)

	for _, token := range strings.Split(r.Header.Get(guardHeader), ",") {
		if token = strings.TrimSpace(token); token != "" {
			tokens = append(tokens, token)

			if token == l.token {
				contains = true
			}
		}
	}

	if l.token != "" && contains {
		log.Warn("Request droped to prevent infinite loop")
		w.WriteHeader(http.StatusAccepted)
		h := w.Header()
		h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		h.Set("Connection", "keep-alive")

		return
	}

	if l.token != "" {
		tokens = append(tokens, l.token)
	}

	r.Header.Set(guardHeader, strings.Join(tokens, ", "))
	l.next.ServeHTTP(w, r)
}

// NewLoopGuard allocates and returns a new LoopGuard.
func NewLoopGuard(token string, next http.Handler) *LoopGuard {
	return &LoopGuard{
		token: token,
		next:  next,
	}
}
