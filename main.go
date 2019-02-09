package main

import (
	"crypto/tls"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	kitlog "github.com/go-kit/kit/log"
	promtailclient "github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
)

const firehoseSubscriptionId = "paas-loki-exporter"

func main() {
	dopplerAddress, authToken, parsedLokiUrl := readEnvironment()
	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	promtailClient := createPromtailClient(logger, parsedLokiUrl)

	firehoseConsumer := consumer.New(dopplerAddress, &tls.Config{}, nil)
	msgChan, errorChan := firehoseConsumer.FilteredFirehose(firehoseSubscriptionId, authToken, consumer.LogMessages)

	go readErrorChan(logger, errorChan)
	forwardMsgChanToPromtail(promtailClient, logger, msgChan)
}

func readEnvironment() (dopplerAddress string, authToken string, parsedLokiUrl *url.URL) {
	dopplerAddress, ok := os.LookupEnv("DOPPLER_ADDR")
	if !ok {
		log.Fatalf("variable DOPPLER_ADDR must be set")
	}
	authToken, ok = os.LookupEnv("CF_ACCESS_TOKEN")
	if !ok {
		log.Fatalf("variable CF_ACCESS_TOKEN must be set")
	}
	lokiUrl, ok := os.LookupEnv("LOKI_URL")
	if !ok {
		log.Fatalf("variable LOKI_URL must be set")
	}
	parsedLokiUrl, err := url.Parse(lokiUrl)
	if err != nil {
		log.Fatalf("could not parse loki URL (%s) as a URL: %v", lokiUrl, err)
	}
	return dopplerAddress, authToken, parsedLokiUrl
}

func createPromtailClient(logger kitlog.Logger, parsedLokiUrl *url.URL) *promtailclient.Client {
	promtailConfig := promtailclient.Config{
		URL: flagext.URLValue{
			URL: parsedLokiUrl,
		},
	}
	promtailClient, err := promtailclient.New(promtailConfig, logger)
	if err != nil {
		log.Fatalf("could not create promtail client: %v", err)
	}
	return promtailClient
}

func readErrorChan(logger kitlog.Logger, errorChan <-chan error) {
	for err := range errorChan {
		_ = logger.Log("level", "error", "message", err.Error())
	}
}

func forwardMsgChanToPromtail(promtailClient *promtailclient.Client, logger kitlog.Logger, msgChan <-chan *events.Envelope) {
	for msg := range msgChan {
		logMessage := msg.LogMessage
		labelSet := model.LabelSet{
			"app":          model.LabelValue(logMessage.GetAppId()),
			"source_type":  model.LabelValue(logMessage.GetSourceType()),
			"source":       model.LabelValue(logMessage.GetSourceInstance()),
			"message_type": model.LabelValue(logMessage.GetMessageType().String()),
		}
		promtailError := promtailClient.Handle(labelSet, time.Now(), string(logMessage.Message))
		if promtailError != nil {
			_ = logger.Log("level", "error", "message", promtailError.Error())
		}
		print(".")
	}
}
