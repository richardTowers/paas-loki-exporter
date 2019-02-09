package integration_tests

import (
	"fmt"
	"github.com/onsi/gomega/gexec"
	"os"
	"time"

	"code.cloudfoundry.org/loggregator/integration_tests/endtoend"
	"code.cloudfoundry.org/loggregator/integration_tests/fakes"
	"code.cloudfoundry.org/loggregator/testservers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("paas-loki-exporter", func() {
	BeforeSuite(func () {
		routerPath, err := gexec.Build("code.cloudfoundry.org/loggregator/router")
        Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ROUTER_BUILD_PATH", routerPath)
		Expect(err).NotTo(HaveOccurred())
		trafficControllerPath, err := gexec.Build("code.cloudfoundry.org/loggregator/trafficcontroller")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("TRAFFIC_CONTROLLER_BUILD_PATH", trafficControllerPath)
		Expect(err).NotTo(HaveOccurred())
	})

	It("reads messages from the firehose and forwards them to loki", func() {
		dopplerCleanup, dopplerPorts := testservers.StartRouter(
			testservers.BuildRouterConfig(0, 0),
		)
		defer dopplerCleanup()

		ingressCleanup, ingressClient := fakes.DopplerIngressV2Client(
			fmt.Sprintf("127.0.0.1:%d", dopplerPorts.GRPC),
		)
		defer ingressCleanup()

		trafficcontrollerCleanup, tcPorts := testservers.StartTrafficController(
			testservers.BuildTrafficControllerConf(
				fmt.Sprintf("127.0.0.1:%d", dopplerPorts.GRPC),
				0,
				fmt.Sprintf("127.0.0.1:%d", 0),
			),
		)
		defer trafficcontrollerCleanup()

		// firehoseReader := endtoend.NewFirehoseReader(tcPorts.WS)

		// TODO we need to:
		// * Build paas-loki-exporter
		// * Start a stub loki
		// * Start paas-loki-exporter pointing at the stub firehose and stub loki
		// * Query loki to check that logs have arrived
	}, 10)
})

