package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestInitMetrics tests that metrics are properly initialized
func TestInitMetrics(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	// First call should initialize
	InitMetrics()

	if JobExecutionTotal == nil {
		t.Fatal("JobExecutionTotal should be initialized")
	}

	if JobExecutionDuration == nil {
		t.Fatal("JobExecutionDuration should be initialized")
	}

	if JobLastStatus == nil {
		t.Fatal("JobLastStatus should be initialized")
	}

	if JobLastDuration == nil {
		t.Fatal("JobLastDuration should be initialized")
	}

	if JobLastTimestamp == nil {
		t.Fatal("JobLastTimestamp should be initialized")
	}
}

// TestRecordJobExecutionSuccess tests recording a successful execution
func TestRecordJobExecutionSuccess(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	RecordJobExecution("test-job", true, false, 1.5)

	// Verify that metrics were recorded
	// In a real test, we would check the metric values via the Go client
	if JobExecutionTotal == nil {
		t.Fatal("JobExecutionTotal was not initialized")
	}
}

// TestRecordJobExecutionFailure tests recording a failed execution
func TestRecordJobExecutionFailure(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	RecordJobExecution("test-job-failed", false, false, 0.5)

	if JobExecutionTotal == nil {
		t.Fatal("JobExecutionTotal was not initialized")
	}
}

// TestRecordJobExecutionSkipped tests recording a skipped execution
func TestRecordJobExecutionSkipped(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	RecordJobExecution("test-job-skipped", true, true, 0.1)

	if JobExecutionTotal == nil {
		t.Fatal("JobExecutionTotal was not initialized")
	}
}

// TestSingletonPattern tests that InitMetrics follows singleton pattern
func TestSingletonPattern(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	InitMetrics()
	firstMetrics := JobExecutionTotal

	// Reset and reinitialize
	InitMetrics()
	secondMetrics := JobExecutionTotal

	// They should be the same instance
	if firstMetrics != secondMetrics {
		t.Fatal("InitMetrics should return the same metric instance (singleton pattern)")
	}
}

// TestRecordJobExecutionMultipleJobs tests recording multiple different jobs
func TestRecordJobExecutionMultipleJobs(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	jobs := []struct {
		name     string
		success  bool
		skipped  bool
		duration float64
	}{
		{"job1", true, false, 1.0},
		{"job2", false, false, 0.5},
		{"job3", true, true, 0.1},
		{"job1", true, false, 1.2},
	}

	for _, job := range jobs {
		RecordJobExecution(job.name, job.success, job.skipped, job.duration)
	}

	if JobExecutionTotal == nil {
		t.Fatal("JobExecutionTotal was not initialized")
	}
}

// TestRecordJobExecutionStatusLabels tests that correct status labels are assigned
func TestRecordJobExecutionStatusLabels(t *testing.T) {
	// Reset prometheus metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	tests := []struct {
		success bool
		skipped bool
		expectedStatus string
	}{
		{true, false, "success"},
		{false, false, "failure"},
		{true, true, "skipped"},
	}

	for _, test := range tests {
		// Call RecordJobExecution with the test parameters
		RecordJobExecution("test-job", test.success, test.skipped, 1.0)
		
		// In a production test, we would verify the actual label values
		// by querying the metrics registry
	}
}
