package middlewares

import (
	"github.com/mcuadros/ofelia/core"
	"github.com/mcuadros/ofelia/metrics"
)

// PrometheusMetrics middleware records job execution metrics in Prometheus format
type PrometheusMetrics struct{}

// ContinueOnStop asks to continue on stop to ensure we record all metrics
func (m *PrometheusMetrics) ContinueOnStop() bool {
	return true
}

// Run executes the next middleware and records metrics on completion
func (m *PrometheusMetrics) Run(ctx *core.Context) error {
	err := ctx.Next()
	ctx.Stop(err)

	// Record the execution metrics
	durationSeconds := ctx.Execution.Duration.Seconds()
	success := !ctx.Execution.Failed && !ctx.Execution.Skipped
	skipped := ctx.Execution.Skipped

	metrics.RecordJobExecution(ctx.Job.GetName(), success, skipped, durationSeconds)

	return err
}

// NewPrometheusMetrics returns a new PrometheusMetrics middleware
func NewPrometheusMetrics() core.Middleware {
	return &PrometheusMetrics{}
}
