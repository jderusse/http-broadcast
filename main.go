package main

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	fluentd "github.com/joonix/log"
	log "github.com/sirupsen/logrus"

	"github.com/jderusse/http-broadcast/pkg/broadcaster"
)

func initLogger() {
	format := os.Getenv("LOG_FORMAT")
	switch strings.ToLower(format) {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "":
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	case "fluentd":
		log.SetFormatter(fluentd.NewFormatter())
	default:
		log.Error(fmt.Sprintf(`Unexpected log format "%s"`, format))
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		l, err := log.ParseLevel(level)
		if err != nil {
			log.Error(fmt.Sprintf(`Unexpected log level "%s"`, level))
		} else {
			log.SetLevel(l)
		}
	}
}

func main() {
	initLogger()
	broadcaster, err := broadcaster.NewBroadcasterFromEnv()
	if err != nil {
		log.Fatalln(err)
	}

	broadcaster.Run()
}
