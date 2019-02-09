package integration_tests

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/loggregator/integration_tests/fakes"
	"code.cloudfoundry.org/loggregator/testservers"
	"github.com/grafana/loki/pkg/logproto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ = Describe("paas-loki-exporter", func() {
	var paasLokiExporterPath string

	BeforeSuite(func() {
		routerPath, err := gexec.Build("code.cloudfoundry.org/loggregator/router")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("ROUTER_BUILD_PATH", routerPath)
		Expect(err).NotTo(HaveOccurred())

		trafficControllerPath, err := gexec.Build("code.cloudfoundry.org/loggregator/trafficcontroller")
		Expect(err).NotTo(HaveOccurred())
		err = os.Setenv("TRAFFIC_CONTROLLER_BUILD_PATH", trafficControllerPath)
		Expect(err).NotTo(HaveOccurred())

		paasLokiExporterPath, err = gexec.Build("github.com/richardTowers/paas-loki-exporter")
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

		trafficControllerCleanup, tcPorts := testservers.StartTrafficController(
			testservers.BuildTrafficControllerConf(
				fmt.Sprintf("127.0.0.1:%d", dopplerPorts.GRPC),
				0,
				fmt.Sprintf("127.0.0.1:%d", 0),
			),
		)
		defer trafficControllerCleanup()

		envelope := loggregator_v2.Envelope{}
		Expect(ingressClient.Send(&envelope)).To(Succeed())

		lis, err := net.Listen("tcp", ":0")
		Expect(err).NotTo(HaveOccurred())

		grpcServer := grpc.NewServer()
		logproto.RegisterPusherServer(grpcServer, NewFakePusherServer())
		go func() {
			_ = grpcServer.Serve(lis)
		}()
		defer grpcServer.Stop()

		time.Sleep(5 * time.Second)

		// * Start paas-loki-exporter pointing at the stub firehose and stub loki
		Expect(os.Setenv("DOPPLER_ADDR", fmt.Sprintf("ws://127.0.0.1:%d", tcPorts.WS))).To(Succeed())
		Expect(os.Setenv("CF_ACCESS_TOKEN", "bearer fake-token")).To(Succeed())
		Expect(os.Setenv("LOKI_URL", fmt.Sprintf("%s://%s", lis.Addr().Network(), lis.Addr().String()))).To(Succeed())
		command := exec.Command(paasLokiExporterPath)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		// Eventually(session.Out, time.Minute).Should(gbytes.Say("."))
		Eventually(session).Should(gexec.Exit(0))
		// TODO we need to:
		// * Query loki to check that logs have arrived
	}, 10)
})

type FakePusherServer struct {
	Requests []logproto.PushRequest
}

func NewFakePusherServer() FakePusherServer {
	return FakePusherServer{
		Requests: []logproto.PushRequest{},
	}
}

func (f FakePusherServer) Push(context.Context, *logproto.PushRequest) (*logproto.PushResponse, error) {
	// response := logproto.PushResponse{}
	println("A thing happened!")
	return nil, fmt.Errorf("thing happening")
}
