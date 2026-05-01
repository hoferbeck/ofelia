package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	once sync.Once

	// JobExecutionTotal tracks the total number of job executions by status
	JobExecutionTotal *prometheus.CounterVec

	// JobExecutionDuration tracks the duration of job executions
	JobExecutionDuration *prometheus.HistogramVec

	// JobLastStatus tracks the status of the last execution (1 for success, 0 for failure/skipped)
	JobLastStatus *prometheus.GaugeVec

	// JobLastDuration tracks the duration of the last execution
	JobLastDuration *prometheus.GaugeVec

	// JobLastTimestamp tracks time of the last execution
	JobLastTimestamp *prometheus.GaugeVec
)

// InitMetrics initializes Prometheus metrics (singleton pattern)
func InitMetrics() {
	once.Do(func() {
		JobExecutionTotal = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ofelia_job_execution_total",
				Help: "Total number of job executions by status (success, failure, skipped)",
			},
			[]string{"job_name", "status"},
		)

		JobExecutionDuration = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ofelia_job_execution_duration_seconds",
				Help:    "Job execution duration in seconds",
				Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
			},
			[]string{"job_name", "status"},
		)

		JobLastStatus = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ofelia_job_last_status",
				Help: "Status of the last job execution (1 for success, 0 for failure/skipped)",
			},
			[]string{"job_name"},
		)

		JobLastDuration = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ofelia_job_last_duration_seconds",
				Help: "Duration of the last job execution in seconds",
			},
			[]string{"job_name"},
		)

		JobLastTimestamp = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ofelia_job_last_execution_timestamp_seconds",
				Help: "Timestamp of the last job execution",
			},
			[]string{"job_name"},
		)
	})
}

// RecordJobExecution records metrics for a completed job execution
func RecordJobExecution(jobName string, success bool, skipped bool, durationSeconds float64) {
	InitMetrics()

	status := "success"
	if skipped {
		status = "skipped"
	} else if !success {
		status = "failure"
	}

	// Record execution counter
	JobExecutionTotal.WithLabelValues(jobName, status).Inc()

	// Record execution duration
	JobExecutionDuration.WithLabelValues(jobName, status).Observe(durationSeconds)

	// Update last status
	lastStatus := 1.0
	if !success || skipped {
		lastStatus = 0.0
	}
	JobLastStatus.WithLabelValues(jobName).Set(lastStatus)

	// Update last duration
	JobLastDuration.WithLabelValues(jobName).Set(durationSeconds)

	// Update last timestamp
	JobLastTimestamp.WithLabelValues(jobName).SetToCurrentTime()
}
