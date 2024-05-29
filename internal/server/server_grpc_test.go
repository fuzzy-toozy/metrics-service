package server

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/beevik/guid"
	"github.com/fuzzy-toozy/metrics-service/internal/encryption"
	"github.com/fuzzy-toozy/metrics-service/internal/log"
	"github.com/fuzzy-toozy/metrics-service/internal/metrics"
	pb "github.com/fuzzy-toozy/metrics-service/internal/proto"
	"github.com/fuzzy-toozy/metrics-service/internal/server/config"
	"github.com/fuzzy-toozy/metrics-service/internal/server/storage"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
)

var lis *bufconn.Listener

const bufSize = 1024 * 1024

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func generateCertificates(logger log.Logger, dir string) error {
	runCommand := func(cmd *exec.Cmd) error {
		output, err := cmd.CombinedOutput()
		logger.Infof("Command output: %s\n", output)

		if err != nil {
			return err
		}

		return nil
	}

	caCmd := exec.Command("openssl", "req", "-x509", "-nodes", "-newkey", "rsa:2048", "-keyout", dir+"ca.key", "-out", dir+"ca.crt", "-days", "365", "-subj", "/CN=MyCA")
	err := runCommand(caCmd)
	if err != nil {
		return err
	}

	serverCSR := exec.Command("openssl", "req", "-new", "-newkey", "rsa:2048", "-nodes", "-keyout", dir+"server.key", "-out", dir+"server.csr", "-subj", "/CN=localhost")
	err = runCommand(serverCSR)
	if err != nil {
		return err
	}

	extFileContent := []byte("subjectAltName = DNS:bufnet")

	extFileName := dir + "ext.conf"
	if err := os.WriteFile(extFileName, extFileContent, 0644); err != nil {
		return fmt.Errorf("Failed to write extension file: %v", err)
	}

	defer os.Remove(extFileName)

	serverCert := exec.Command("openssl", "x509", "-req", "-in", dir+"server.csr", "-CA", dir+"ca.crt", "-CAkey", dir+"ca.key", "-CAcreateserial", "-out", dir+"server.crt", "-days", "365", "-extfile", extFileName)
	err = runCommand(serverCert)
	if err != nil {
		return err
	}

	clientCSR := exec.Command("openssl", "req", "-new", "-newkey", "rsa:2048", "-nodes", "-keyout", dir+"client.key", "-out", dir+"client.csr", "-subj", "/CN=localhost")
	err = runCommand(clientCSR)
	if err != nil {
		return err
	}

	clientCert := exec.Command("openssl", "x509", "-req", "-in", dir+"client.csr", "-CA", dir+"ca.crt", "-CAkey", dir+"ca.key", "-CAcreateserial", "-out", dir+"client.crt", "-days", "365", "-extfile", extFileName)
	err = runCommand(clientCert)

	return err
}

func TestServerGRPC(t *testing.T) {
	runServerTest(t, false)
	runServerTest(t, true)
}

func runServerTest(t *testing.T, useTLS bool) {
	var appendTestName string
	conf := &config.Config{}
	r := require.New(t)
	logger := log.NewDevZapLogger()
	creds := insecure.NewCredentials()

	if useTLS {
		dir := "test_certs/"
		os.RemoveAll(dir)
		r.NoError(os.Mkdir(dir, 0744))

		appendTestName = "TLS"
		r.NoError(generateCertificates(logger, dir))
		conf.CaCertPath = dir + "ca.crt"
		conf.ServerCertPath = dir + "server.crt"
		conf.EncKeyPath = dir + "server.key"
		agentCertPath := dir + "client.crt"
		agentKeyPath := dir + "client.key"

		tlsCreds, err := encryption.SetupClientTLS(conf.CaCertPath, agentKeyPath, agentCertPath)
		r.NoError(err)
		creds = tlsCreds

		defer os.RemoveAll(dir)
	}

	lis = bufconn.Listen(bufSize)
	conf.MaxBodySize = bufSize
	s, err := NewServerGRPC(conf, logger.Desugar(), storage.NewCommonMetricsRepository(nil, logger))
	r.NoError(err)

	s.SetListener(lis)
	servWg := sync.WaitGroup{}
	servWg.Add(1)

	defer func() {
		s.Stop(context.Background())
		servWg.Wait()
	}()

	go func() {
		defer servWg.Done()
		s.Run()
	}()

	resolver.SetDefaultScheme("passthrough")

	clientConn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(creds),
	)
	r.NoError(err)
	defer clientConn.Close()
	client := pb.NewMetricsServiceClient(clientConn)

	ctx := context.Background()
	allMetrics := make(map[string]*pb.Metric)
	procMericsCtr := make(map[string]*pb.Metric)
	procMetricsGauge := make(map[string]*pb.Metric)
	t.Run("UpdateCounterMetric"+appendTestName, func(t *testing.T) {
		for i := 0; i < 100; i++ {
			reqM := &pb.Metric{
				Id:    guid.NewString(),
				Mtype: metrics.CounterMetricType,
				Delta: rand.Int63(),
			}

			m, err := client.UpdateMetric(ctx, &pb.MetricUpdateRequest{
				Metric: reqM,
			})

			r.NoError(err)
			r.Equal(m.Id, reqM.Id)
			r.Equal(m.Delta, reqM.Delta)
			r.Equal(m.Mtype, reqM.Mtype)

			m, err = client.UpdateMetric(ctx, &pb.MetricUpdateRequest{
				Metric: reqM,
			})

			r.NoError(err)
			r.Equal(m.Id, reqM.Id)
			r.Equal(m.Delta, reqM.Delta*2)
			r.Equal(m.Mtype, reqM.Mtype)

			procMericsCtr[m.Id] = m
		}
	})

	t.Run("UpdateGaugeMetric"+appendTestName, func(t *testing.T) {
		for i := 0; i < 100; i++ {
			reqM := &pb.Metric{
				Id:    guid.NewString(),
				Mtype: metrics.GaugeMetricType,
				Value: rand.Float64(),
			}

			r.NoError(err)

			m, err := client.UpdateMetric(ctx, &pb.MetricUpdateRequest{
				Metric: reqM,
			})

			r.NoError(err)
			r.Equal(m.Id, reqM.Id)
			r.Equal(m.Value, reqM.Value)
			r.Equal(m.Mtype, reqM.Mtype)

			procMetricsGauge[m.Id] = m
		}
	})

	t.Run("UpdateMetrics"+appendTestName, func(t *testing.T) {
		reqMetrics := make([]*pb.Metric, 0, 200)
		reqMetricsMap := make(map[string]*pb.Metric)
		for i := 0; i < 100; i++ {
			gm := &pb.Metric{
				Id:    guid.NewString(),
				Mtype: metrics.GaugeMetricType,
				Value: rand.Float64(),
			}

			cm := &pb.Metric{
				Id:    guid.NewString(),
				Mtype: metrics.CounterMetricType,
				Delta: rand.Int63(),
			}

			reqMetricsMap[gm.Id] = gm
			reqMetricsMap[cm.Id] = cm

			reqMetrics = append(reqMetrics, gm, cm)
		}

		respMetrics, err := client.UpdateMetrics(ctx, &pb.MetricsUpdateRequest{
			Metrics: reqMetrics,
		})

		r.NoError(err)

		for _, m := range respMetrics.Metrics {
			golden, ok := reqMetricsMap[m.Id]
			r.True(ok)

			r.Equal(m.Id, golden.Id)
			r.Equal(m.Mtype, golden.Mtype)

			if m.Mtype == metrics.GaugeMetricType {
				r.Equal(m.Value, golden.Value)
			} else if m.Mtype == metrics.CounterMetricType {
				r.Equal(m.Delta, golden.Delta)
			} else {
				r.True(false)
			}

			allMetrics[m.Id] = m
		}
	})

	t.Run("GetMetric"+appendTestName, func(t *testing.T) {
		for _, m := range procMericsCtr {
			reqM := &pb.MetricRequest{
				Id:    m.Id,
				Mtype: m.Mtype,
			}

			respM, err := client.GetMetric(ctx, reqM)
			r.NoError(err)
			r.Equal(m.Id, respM.Id)
			r.Equal(m.Delta, respM.Delta)
			r.Equal(m.Mtype, reqM.Mtype)
		}

		for _, m := range procMetricsGauge {
			reqM := &pb.MetricRequest{
				Id:    m.Id,
				Mtype: m.Mtype,
			}

			respM, err := client.GetMetric(ctx, reqM)
			r.NoError(err)
			r.Equal(m.Id, respM.Id)
			r.Equal(m.Value, respM.Value)
			r.Equal(m.Mtype, reqM.Mtype)
		}
	})

	for k, v := range procMericsCtr {
		allMetrics[k] = v
	}

	for k, v := range procMetricsGauge {
		allMetrics[k] = v
	}

	t.Run("GetMetricsAll"+appendTestName, func(t *testing.T) {
		respM, err := client.GetAllMetrics(ctx, &empty.Empty{})
		r.NoError(err)

		for _, m := range respM.Metrics {
			expM, ok := allMetrics[m.Id]
			r.True(ok)

			r.Equal(m.Id, expM.Id)
			r.Equal(m.Mtype, expM.Mtype)
			if m.Mtype == metrics.GaugeMetricType {
				r.Equal(m.Value, expM.Value)
			} else if m.Mtype == metrics.CounterMetricType {
				r.Equal(m.Delta, expM.Delta)
			} else {
				r.True(false)
			}
		}

		for _, m := range procMetricsGauge {
			reqM := &pb.MetricRequest{
				Id:    m.Id,
				Mtype: m.Mtype,
			}

			respM, err := client.GetMetric(ctx, reqM)
			r.NoError(err)
			r.Equal(m.Id, respM.Id)
			r.Equal(m.Value, respM.Value)
			r.Equal(m.Mtype, reqM.Mtype)
		}
	})
}
