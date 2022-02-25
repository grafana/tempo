package remotewrite

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	gokitlog "github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	prometheus_common_config "github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/exemplar"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/stretchr/testify/assert"
)

func Test_remoteWriteAppendable(t *testing.T) {
	theTime := time.Now()

	var capturedTimeseries []prompb.TimeSeries
	clientCfg, cleanup := createTimeSeriesServer(t, &capturedTimeseries)
	defer cleanup()

	clientCfg.SendExemplars = true

	cfg := &Config{
		Enabled: true,
		Client:  clientCfg,
	}
	tenantID := "my-tenant"

	appendable := NewAppendable(cfg, gokitlog.NewLogfmtLogger(os.Stdout), tenantID, NewMetrics(prometheus.NewRegistry()))

	appender := appendable.Appender(context.Background())

	_, err := appender.Append(0, labels.Labels{{Name: "label", Value: "append-before-rollback"}}, theTime.UnixMilli(), 0.1)
	assert.NoError(t, err)

	// Rollback the appender, this should discard previously appended samples
	err = appender.Rollback()
	assert.NoError(t, err)

	err = appender.Commit()
	assert.NoError(t, err)

	assert.Len(t, capturedTimeseries, 0)

	_, err = appender.Append(0, labels.Labels{{Name: "label", Value: "value"}}, theTime.UnixMilli(), 0.2)
	assert.NoError(t, err)
	_, err = appender.AppendExemplar(0, labels.Labels{{Name: "label", Value: "value"}}, exemplar.Exemplar{
		Labels: labels.Labels{{Name: "traceID", Value: "123"}},
		Value:  0.2,
		Ts:     theTime.UnixMilli(),
		HasTs:  true,
	})
	assert.NoError(t, err)

	err = appender.Commit()
	assert.NoError(t, err)

	assert.Len(t, capturedTimeseries, 2)
	assert.Len(t, capturedTimeseries[0].Labels, 1)
	assert.Equal(t, `name:"label" value:"value" `, capturedTimeseries[0].Labels[0].String())
	assert.Len(t, capturedTimeseries[0].Samples, 1)
	assert.Equal(t, fmt.Sprintf(`value:0.2 timestamp:%d `, theTime.UnixMilli()), capturedTimeseries[0].Samples[0].String())
	assert.Len(t, capturedTimeseries[1].Exemplars, 1)
}

func Test_remoteWriteAppendable_disabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		t.Fatal("server should never be called")
	}))
	defer server.Close()

	url, err := url.Parse(fmt.Sprintf("http://%s/receive", server.Listener.Addr().String()))
	assert.NoError(t, err)

	clientCfg := config.DefaultRemoteWriteConfig
	clientCfg.URL = &prometheus_common_config.URL{URL: url}

	cfg := &Config{
		Enabled: false,
		Client:  clientCfg,
	}

	appendable := NewAppendable(cfg, gokitlog.NewLogfmtLogger(os.Stdout), "", NewMetrics(prometheus.NewRegistry()))

	appender := appendable.Appender(context.Background())

	_, err = appender.Append(0, labels.Labels{{Name: "label", Value: "value"}}, time.Now().UnixMilli(), 0.1)
	assert.NoError(t, err)

	err = appender.Commit()
	assert.NoError(t, err)
}

func Test_remoteWriteAppendable_dontSendExemplars(t *testing.T) {
	theTime := time.Now()

	var capturedTimeseries []prompb.TimeSeries
	clientCfg, cleanup := createTimeSeriesServer(t, &capturedTimeseries)
	defer cleanup()

	clientCfg.SendExemplars = false

	cfg := &Config{
		Enabled: true,
		Client:  clientCfg,
	}
	tenantID := "my-tenant"

	appendable := NewAppendable(cfg, gokitlog.NewLogfmtLogger(os.Stdout), tenantID, NewMetrics(prometheus.NewRegistry()))

	appender := appendable.Appender(context.Background())

	_, err := appender.AppendExemplar(0, labels.Labels{{Name: "label", Value: "value"}}, exemplar.Exemplar{
		Labels: labels.Labels{{Name: "traceID", Value: "123"}},
		Value:  0.2,
		Ts:     theTime.UnixMilli(),
		HasTs:  true,
	})
	assert.NoError(t, err)

	err = appender.Commit()
	assert.NoError(t, err)

	assert.Len(t, capturedTimeseries, 0)
}

func createTimeSeriesServer(t *testing.T, capturedTimeseries *[]prompb.TimeSeries) (config.RemoteWriteConfig, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		writeRequest, err := remote.DecodeWriteRequest(req.Body)
		assert.NoError(t, err)

		*capturedTimeseries = append(*capturedTimeseries, writeRequest.GetTimeseries()...)
	}))

	url, err := url.Parse(fmt.Sprintf("http://%s/receive", server.Listener.Addr().String()))
	assert.NoError(t, err)

	clientCfg := config.DefaultRemoteWriteConfig
	clientCfg.URL = &prometheus_common_config.URL{URL: url}

	return clientCfg, func() {
		server.Close()
	}
}
