package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
	prompb "github.com/prometheus/client_model/go"
)

func TestHTTP(t *testing.T) {
	const (
		elapsed     = 100 * time.Millisecond
		rootPath    = "/"
		metricsPath = "/metrics"
	)

	g := gin.New()
	g.Use(HTTP)
	g.GET(rootPath, func(*gin.Context) { time.Sleep(elapsed) })
	g.GET(metricsPath, gin.WrapH(promhttp.Handler()))

	testServer := httptest.NewServer(g.Handler())

	rsp, err := testServer.Client().Get(testServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = rsp.Body.Close()

	if rsp, err = testServer.Client().Get(testServer.URL + metricsPath); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rsp.Body.Close() }()

	l := promlint.New(rsp.Body)
	l.AddCustomValidations(func(mf *prompb.MetricFamily) []error {
		for _, metric := range mf.GetMetric() {
			if h := metric.Histogram; h != nil {
				if sum := time.Duration(*h.SampleSum * float64(time.Second)); sum <= elapsed {
					t.Fatal(sum)
				}
				if count := *h.SampleCount; count != 1 {
					t.Fatal(count)
				}
				if v := *metric.Label[0].Value; v != http.MethodGet {
					t.Fatal(v)
				}
				if v := *metric.Label[1].Value; v != rootPath {
					t.Fatal(v)
				}
				if v := *metric.Label[2].Value; v != "200" {
					t.Fatal(v)
				}
			}
		}
		return nil
	})
	if problems, err := l.Lint(); err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
}
