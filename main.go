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
	"gopkg.in/alecthomas/kingpin.v2"
)

const firehoseSubscriptionId = "paas-loki-exporter"

var (
	dopplerAddress = kingpin.Arg("doppler-addr", "Doppler Address").Required().Envar("DOPPLER_ADDR").String()
	authToken      = kingpin.Arg("cf-token", "CF access token").Required().Envar("CF_ACCESS_TOKEN").String()
	lokiURL        = kingpin.Arg("loki-url", "Loki URL").Required().Envar("LOKI_URL").URL()
)

func main() {
	kingpin.Parse()

	logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	promtailClient := createPromtailClient(logger, *lokiURL)

	firehoseConsumer := consumer.New(*dopplerAddress, &tls.Config{InsecureSkipVerify:true}, nil)
	msgChan, errorChan := firehoseConsumer.FilteredFirehose(firehoseSubscriptionId, *authToken, consumer.LogMessages)

	for {
		select {
		case err, ok := <-errorChan:
			if !ok {
				log.Fatalf("error channel was closed")
			} else {
				_ = logger.Log("level", "error", "message", err.Error())
			}
		case msg, ok := <-msgChan:
			if !ok {
				log.Fatalf("msg channel was closed")
			} else {
				forwardMsgToPromtail(promtailClient, logger, msg)
			}
		}
	}
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

func forwardMsgToPromtail(promtailClient *promtailclient.Client, logger kitlog.Logger, msg *events.Envelope) {
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
