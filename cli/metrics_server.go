package cli

import (
	"fmt"
	"net"
	"net/http"

	"github.com/mcuadros/ofelia/core"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer exposes Prometheus metrics via HTTP
type MetricsServer struct {
	Address string
	Port    int
	Logger  core.Logger
	server  *http.Server
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(address string, port int, logger core.Logger) *MetricsServer {
	return &MetricsServer{
		Address: address,
		Port:    port,
		Logger:  logger,
	}
}

// Start starts the metrics HTTP server in a goroutine
func (m *MetricsServer) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	listenAddr := net.JoinHostPort(m.Address, fmt.Sprint(m.Port))
	m.server = &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	go func() {
		m.Logger.Debug("Starting metrics server", "address", listenAddr)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			m.Logger.Error("Metrics server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the metrics server
func (m *MetricsServer) Stop() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}
