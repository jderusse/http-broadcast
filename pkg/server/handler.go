package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/jderusse/http-broadcast/pkg/dto"
)

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	rStr, _ := httputil.DumpRequest(r, true)

	log.WithFields(log.Fields{"request": string(rStr)}).Debug("Server: Handling request")

	request := dto.NewRequestFromHTTP(r)

	// serializing original request
	data, err := json.Marshal(request)
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to encode Request"))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	// building Hub Request
	log.WithFields(log.Fields{"data": string(data), "topic": s.options.Hub.Topic, "target": s.options.Hub.Target, "hub": s.options.Hub.Endpoint}).Debug("Server: Pushing message to HUB")

	form := url.Values{}
	form.Set("topic", s.options.Hub.Topic)
	form.Set("target", s.options.Hub.Target)
	form.Set("data", string(data))
	formData := form.Encode()

	hubRequest, _ := http.NewRequest("POST", s.options.Hub.Endpoint.String(), strings.NewReader(formData))

	if s.options.Hub.PublishToken != "" {
		hubRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.options.Hub.PublishToken))
	}

	hubRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	hubRequest.Header.Add("Content-Length", strconv.Itoa(len(formData)))

	// pushing request to hub
	hubResponse, err := s.hubClient.Do(hubRequest)
	if err != nil {
		log.Error(errors.Wrap(err, "Server: Failed to push record"))
	}

	if hubResponse.StatusCode >= 400 {
		defer hubResponse.Body.Close()
		respStr, _ := httputil.DumpResponse(hubResponse, true)

		log.WithFields(log.Fields{"response": string(respStr)}).Error(errors.New("invalid response from Hub"))
	}

	if err != nil || hubResponse.StatusCode >= 400 {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	log.WithFields(log.Fields{"request": request}).Debug("Server: message Published")

	defer hubResponse.Body.Close()

	w.WriteHeader(http.StatusAccepted)
	h := w.Header()
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Connection", "keep-alive")
}
