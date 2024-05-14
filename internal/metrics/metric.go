package metrics

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	StageInitializing = iota + 1
	StageCatchup
	StageServing
	StageUpdating
	StageReorg
)

func fqn(name string) string {
	return prometheus.BuildFQName("nubit", "modular_committee", name)
}

var (
	Version = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: fqn("version"),
			Help: "Service version number",
		},
		[]string{"version"},
	)

	Stage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: fqn("stage"),
		Help: "Service stage (e.g. initializing, catchup)",
	})

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fqn("dbquery_duration"),
			Help:    "Duration of database queries",
			Buckets: []float64{0.02, 0.05, 0.1, 0.2, 0.5, 1, 5},
		},
		[]string{"op"},
	)

	CurrentHeight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: fqn("current_height"),
		Help: "Current height during catchup or serving",
	})

	HttpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fqn("http_duration"),
			Help:    "HTTP request duration",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 5, 15},
		},
		[]string{"method", "path", "status"},
	)
)

func ObserveDBQuery(op string, started time.Time) {
	DBQueryDuration.WithLabelValues(op).Observe(time.Since(started).Seconds())
}

func HTTP(c *gin.Context) {
	started := time.Now()

	c.Next()

	HttpDuration.WithLabelValues(
		c.Request.Method,
		c.Request.URL.Path,
		strconv.Itoa(c.Writer.Status()),
	).Observe(time.Since(started).Seconds())
}

func init() {
	prometheus.MustRegister(
		Version,
		Stage,
		DBQueryDuration,
		CurrentHeight,
		HttpDuration,
	)
}

func ListenAndServe(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	if err := (&http.Server{Addr: addr, Handler: mux}).ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
