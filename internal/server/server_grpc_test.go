package server

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"testing"

	"github.com/beevik/guid"
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
	"google.golang.org/protobuf/proto"
)

var lis *bufconn.Listener

const bufSize = 1024 * 1024

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestServerGRPC(t *testing.T) {
	r := require.New(t)
	lis = bufconn.Listen(bufSize)
	conf := &config.Config{}
	conf.MaxBodySize = bufSize
	s, err := NewServerGRPC(conf, log.NewDevZapLogger(), storage.NewCommonMetricsRepository(), nil)
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
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	r.NoError(err)
	defer clientConn.Close()
	client := pb.NewMetricsServiceClient(clientConn)

	ctx := context.Background()
	procMericsCtr := make(map[string]*pb.Metric)
	procMetricsGauge := make(map[string]*pb.Metric)
	t.Run("UpdateCounterMetric", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			reqM := &pb.Metric{
				Id:    guid.NewString(),
				Mtype: metrics.CounterMetricType,
				Delta: rand.Int63(),
			}

			data, err := proto.Marshal(reqM)
			r.NoError(err)

			m, err := client.UpdateMetric(ctx, &pb.UpdateRequest{
				Data: data,
			})

			r.NoError(err)
			r.Equal(m.Id, reqM.Id)
			r.Equal(m.Delta, reqM.Delta)
			r.Equal(m.Mtype, reqM.Mtype)

			m, err = client.UpdateMetric(ctx, &pb.UpdateRequest{
				Data: data,
			})

			r.NoError(err)
			r.Equal(m.Id, reqM.Id)
			r.Equal(m.Delta, reqM.Delta*2)
			r.Equal(m.Mtype, reqM.Mtype)

			procMericsCtr[m.Id] = m
		}
	})

	t.Run("UpdateGaugeMetric", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			reqM := &pb.Metric{
				Id:    guid.NewString(),
				Mtype: metrics.GaugeMetricType,
				Value: rand.Float64(),
			}

			data, err := proto.Marshal(reqM)
			r.NoError(err)

			m, err := client.UpdateMetric(ctx, &pb.UpdateRequest{
				Data: data,
			})

			r.NoError(err)
			r.Equal(m.Id, reqM.Id)
			r.Equal(m.Value, reqM.Value)
			r.Equal(m.Mtype, reqM.Mtype)

			procMetricsGauge[m.Id] = m
		}
	})

	t.Run("GetMetric", func(t *testing.T) {
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

	allMetrics := make(map[string]*pb.Metric, len(procMericsCtr)+len(procMetricsGauge))
	for k, v := range procMericsCtr {
		allMetrics[k] = v
	}

	for k, v := range procMetricsGauge {
		allMetrics[k] = v
	}
	t.Run("GetMetricsAll", func(t *testing.T) {
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
