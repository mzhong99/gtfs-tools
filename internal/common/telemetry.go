package common

import (
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	HttpTTFBSeconds     *prometheus.HistogramVec
	HttpReadBodySeconds *prometheus.HistogramVec
	HttpBytesTotal      *prometheus.CounterVec
	HttpErrorsTotal     *prometheus.CounterVec
}

func NewMetrics(registry *prometheus.Registry) *Metrics {
	metrics := &Metrics{
		HttpTTFBSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gtfs_http_ttfb_seconds",
				Help:    "Time from API GET to first byte for HTTP requests",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"endpoint"},
		),
		HttpReadBodySeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gtfs_http_read_body_seconds",
				Help:    "Time to read body of HTTP request from a GTFS-RT HTTP GET response",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"endpoint"},
		),
		HttpBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gtfs_http_bytes_total",
				Help: "Bytes downloaded per endpoint",
			},
			[]string{"endpoint"},
		),
		HttpErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gtfs_http_errors_total",
				Help: "Errors incurred from sustained interaction with an API endpoint",
			},
			[]string{"endpoint"},
		),
	}

	registry.MustRegister(
		metrics.HttpTTFBSeconds,
		metrics.HttpReadBodySeconds,
		metrics.HttpBytesTotal,
		metrics.HttpErrorsTotal,
	)

	return metrics
}

type TelemetryServer struct {
	addr     string
	mux      *http.ServeMux
	registry *prometheus.Registry

	server   *http.Server
	listener net.Listener
}

func NewTelemetryServer(addr string) *TelemetryServer {
	telemetry := &TelemetryServer{
		addr:     addr,
		registry: prometheus.NewRegistry(),
		mux:      http.NewServeMux(),
	}

	telemetry.mux.Handle(
		"/metrics",
		promhttp.HandlerFor(telemetry.registry, promhttp.HandlerOpts{}),
	)

	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gtfs_build_info",
			Help: "Build metadata",
		},
		[]string{"version", "git_commit"},
	)

	telemetry.registry.MustRegister(
		collectors.NewGoCollector(), // Go runtime metrics
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		buildInfo,
	)

	buildInfo.WithLabelValues(Version, GitCommit).Set(1)

	telemetry.mux.HandleFunc("/debug/pprof/", pprof.Index)
	telemetry.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	telemetry.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	telemetry.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	telemetry.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return telemetry
}

func (telemetry *TelemetryServer) GetRegistry() *prometheus.Registry {
	return telemetry.registry
}

func (telemetry *TelemetryServer) Start() error {
	telemetry.server = &http.Server{
		Addr:              telemetry.addr,
		Handler:           telemetry.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", telemetry.addr)
	if err != nil {
		return err
	}

	telemetry.listener = listener

	go telemetry.server.Serve(telemetry.listener)

	fmt.Printf("Telemetry server started: %s\n", telemetry.addr)
	return nil
}

func (telemetry TelemetryServer) Stop() error {
	if telemetry.server == nil {
		return nil
	}

	return telemetry.server.Close()
}
