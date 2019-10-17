package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/jderusse/http-broadcast/pkg/dto"
)

func (a *Agent) replay(requestID string, data []byte) {
	var request dto.Request

	if err := json.Unmarshal(data, &request); err != nil {
		log.Error(errors.Wrap(err, "parse Request"))
		return
	}

	log.WithFields(log.Fields{"requestID": requestID, "request": request}).Debug("Agent: playing request")

	targetURL, _ := url.Parse(a.options.Agent.Endpoint.String())
	targetURL.Path = fmt.Sprintf("%s/%s", strings.TrimRight(a.options.Agent.Endpoint.Path, "/"), strings.TrimLeft(request.Path, "/"))

	req, _ := http.NewRequest(request.Method, targetURL.String(), bytes.NewBuffer(request.Body))
	req.Header = request.Header
	req.Host = request.Host

	retry := backoff.NewExponentialBackOff()
	retry.MaxInterval = 5 * time.Second
	retry.MaxElapsedTime = a.options.Agent.RetryDelay
	err := backoff.RetryNotify(func() error {
		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
			reqStr, _ := httputil.DumpRequest(req, true)
			respStr, _ := httputil.DumpResponse(resp, true)
			log.WithFields(log.Fields{"requestID": requestID, "request": string(reqStr), "response": string(respStr)}).Debug("Agent: request played")
			if resp.StatusCode >= 400 {
				err = errors.New(fmt.Sprintf(`Server respond with "%d" code.`, resp.StatusCode))
			}
		}
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{"requestID": requestID}).Info("Agent: request played")

		return nil
	}, retry, func(err error, d time.Duration) {
		log.WithFields(log.Fields{"requestID": requestID}).Warn(err)
	})

	if err != nil {
		log.WithFields(log.Fields{"requestID": requestID}).Error(errors.Wrap(err, "replay request"))
	}
}
