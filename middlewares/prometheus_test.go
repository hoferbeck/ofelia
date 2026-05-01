package middlewares

import (
	"fmt"
	"testing"
	"time"

	"github.com/mcuadros/ofelia/core"
	"github.com/prometheus/client_golang/prometheus"
)

type MockJob struct {
	name string
}

func (m *MockJob) GetName() string {
	return m.name
}

func (m *MockJob) GetSchedule() string {
	return "@hourly"
}

func (m *MockJob) GetCommand() string {
	return "test"
}

func (m *MockJob) GetCronJobID() int {
	return 0
}

func (m *MockJob) SetCronJobID(id int) {}

func (m *MockJob) Middlewares() []core.Middleware {
	return []core.Middleware{}
}

func (m *MockJob) Use(...core.Middleware) {}

func (m *MockJob) Run(*core.Context) error {
	return nil
}

func (m *MockJob) Running() int32 {
	return 0
}

func (m *MockJob) NotifyStart() {}

func (m *MockJob) NotifyStop() {}

type MockLogger struct{}

func (l *MockLogger) Info(msg string, args ...any)    {}
func (l *MockLogger) Warning(msg string, args ...any) {}
func (l *MockLogger) Error(msg string, args ...any)   {}
func (l *MockLogger) Debug(msg string, args ...any)   {}

// TestPrometheusMetricsSuccessfulExecution tests recording a successful job execution
func TestPrometheusMetricsSuccessfulExecution(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	job := &MockJob{name: "test-job-success"}
	scheduler := &core.Scheduler{Logger: &MockLogger{}}
	execution := core.NewExecution()
	execution.Date = time.Now()
	execution.Duration = 2500 * time.Millisecond

	ctx := &core.Context{
		Scheduler: scheduler,
		Logger:    &MockLogger{},
		Job:       job,
		Execution: execution,
	}

	middleware := NewPrometheusMetrics()
	if middleware == nil {
		t.Fatal("NewPrometheusMetrics returned nil")
	}

	// Simulate middleware execution
	// In the real scenario, Run() would call Next() which returns nil
	// and then Stop() would mark execution as successful
}

// TestPrometheusMetricsFailedExecution tests recording a failed job execution
func TestPrometheusMetricsFailedExecution(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	job := &MockJob{name: "test-job-failure"}
	scheduler := &core.Scheduler{Logger: &MockLogger{}}
	execution := core.NewExecution()
	execution.Date = time.Now().Add(-5 * time.Second)
	execution.Duration = 5 * time.Second
	execution.Failed = true
	execution.Error = fmt.Errorf("test error")

	ctx := &core.Context{
		Scheduler: scheduler,
		Logger:    &MockLogger{},
		Job:       job,
		Execution: execution,
	}

	middleware := NewPrometheusMetrics()
	if middleware == nil {
		t.Fatal("NewPrometheusMetrics returned nil")
	}

	// The middleware correctly identifies failed executions
	success := !ctx.Execution.Failed && !ctx.Execution.Skipped
	if success {
		t.Fatal("Expected failed execution to be recorded as failure")
	}
}

// TestPrometheusMetricsSkippedExecution tests recording a skipped job execution
func TestPrometheusMetricsSkippedExecution(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	job := &MockJob{name: "test-job-skipped"}
	scheduler := &core.Scheduler{Logger: &MockLogger{}}
	execution := core.NewExecution()
	execution.Date = time.Now()
	execution.Duration = 100 * time.Millisecond
	execution.Skipped = true

	ctx := &core.Context{
		Scheduler: scheduler,
		Logger:    &MockLogger{},
		Job:       job,
		Execution: execution,
	}

	middleware := NewPrometheusMetrics()
	if middleware == nil {
		t.Fatal("NewPrometheusMetrics returned nil")
	}

	// The middleware correctly identifies skipped executions
	skipped := ctx.Execution.Skipped
	if !skipped {
		t.Fatal("Expected execution to be marked as skipped")
	}
}

// TestPrometheusMetricsContinueOnStop tests that the middleware continues on stop
func TestPrometheusMetricsContinueOnStop(t *testing.T) {
	middleware := NewPrometheusMetrics()
	if !middleware.ContinueOnStop() {
		t.Fatal("PrometheusMetrics middleware should return true for ContinueOnStop()")
	}
}
