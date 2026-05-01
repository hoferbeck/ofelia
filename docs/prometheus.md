# Prometheus Metrics Implementation

This document describes the Prometheus metrics endpoint that has been added to Ofelia.

## Overview

A Prometheus metrics endpoint has been integrated into Ofelia to track job execution metrics including:
- Whether a task was successful
- How long a task took to execute
- Execution counts by status (success, failure, skipped)

## Features

### Metrics Exposed

1. **ofelia_job_execution_total** (Counter)
   - Total number of job executions grouped by job name and status
   - Labels: `job_name`, `status` (success, failure, skipped)
   - Use case: Tracking total job runs and failure rates

2. **ofelia_job_execution_duration_seconds** (Histogram)
   - Distribution of job execution durations
   - Labels: `job_name`, `status` (success, failure, skipped)
   - Buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100 seconds
   - Use case: Performance analysis and anomaly detection

3. **ofelia_job_last_status** (Gauge)
   - Status of the last job execution (1 for success, 0 for failure/skipped)
   - Labels: `job_name`
   - Use case: Quick status check via alerting systems

4. **ofelia_job_last_duration_seconds** (Gauge)
   - Duration of the last job execution in seconds
   - Labels: `job_name`
   - Use case: Tracking recent performance changes

5. **ofelia_job_last_execution_timestamp_seconds** (Gauge)
   - Unix timestamp of the last job execution
   - Labels: `job_name`
   - Use case: Detecting stale jobs or missed executions

## Configuration

### Command-line Flags

When running `ofelia daemon`, the following new flags are available:

```bash
--metrics-bind-address string    Metrics server bind address (default: "0.0.0.0")
--metrics-bind-port int          Metrics server bind port (default: 8080)
--disable-metrics-server         Disable Prometheus metrics server
```

### Example Usage

Enable metrics on default port (8080):
```bash
ofelia daemon --config=/etc/ofelia.conf
```

Enable metrics on custom port:
```bash
ofelia daemon --config=/etc/ofelia.conf --metrics-bind-port=9090
```

Disable metrics server:
```bash
ofelia daemon --config=/etc/ofelia.conf --disable-metrics-server
```

## Accessing Metrics

Once the Ofelia daemon is running, metrics are available at:
```
http://localhost:8080/metrics
```

Or on a custom port/address configured via flags.

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'ofelia'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
```

## Example Queries

### Alert on job failures
```promql
# Last 3 executions all failed
increase(ofelia_job_execution_total{status="failure"}[5m]) >= 3
```

### Track average job duration
```promql
rate(ofelia_job_execution_duration_seconds_sum[5m]) / rate(ofelia_job_execution_duration_seconds_count[5m])
```

### Check job success rate
```promql
# Success rate over last hour
sum(increase(ofelia_job_execution_total{status="success"}[1h])) / 
sum(increase(ofelia_job_execution_total[1h]))
```

### Detect long-running jobs
```promql
# Jobs taking longer than 30 seconds
histogram_quantile(0.95, rate(ofelia_job_execution_duration_seconds_bucket[5m])) > 30
```

### Detect missed executions
```promql
# Jobs not executed in the last 30 minutes
time() - ofelia_job_last_execution_timestamp_seconds > 1800
```

## Implementation Details

### Files Added/Modified

**New Files:**
- `metrics/metrics.go` - Core metrics definitions and recording functions
- `middlewares/prometheus.go` - Prometheus metrics middleware
- `cli/metrics_server.go` - HTTP server for exposing metrics
- `docs/prometheus.md` - This documentation file

**Modified Files:**
- `go.mod` - Added `github.com/prometheus/client_golang` dependency
- `cli/daemon.go` - Added metrics server startup and shutdown
- `cli/config.go` - Integrated Prometheus metrics middleware

### Architecture

1. **Metrics Package** (`metrics/metrics.go`)
   - Initializes all Prometheus metrics using the promauto pattern
   - Provides `RecordJobExecution()` function for recording execution data
   - Singleton pattern ensures metrics are only registered once

2. **Middleware** (`middlewares/prometheus.go`)
   - Wraps job execution context
   - Calls `RecordJobExecution()` after each job completes
   - Captures success/failure status and duration

3. **HTTP Server** (`cli/metrics_server.go`)
   - Exposes metrics on `GET /metrics` endpoint
   - Configurable address and port
   - Gracefully handles shutdown

4. **Integration** (`cli/daemon.go`, `cli/config.go`)
   - Starts metrics server during daemon startup
   - Stops metrics server during daemon shutdown
   - Adds Prometheus middleware to scheduler

## Grafana Dashboard

Here's a sample Grafana dashboard JSON for visualizing Ofelia metrics:

```json
{
  "dashboard": {
    "title": "Ofelia Job Scheduler",
    "panels": [
      {
        "title": "Job Success Rate",
        "targets": [
          {
            "expr": "sum(rate(ofelia_job_execution_total{status=\"success\"}[5m])) by (job_name) / sum(rate(ofelia_job_execution_total[5m])) by (job_name)"
          }
        ]
      },
      {
        "title": "Job Execution Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(ofelia_job_execution_duration_seconds_bucket[5m])) by (job_name)"
          }
        ]
      },
      {
        "title": "Last Execution Status",
        "targets": [
          {
            "expr": "ofelia_job_last_status"
          }
        ]
      }
    ]
  }
}
```

## Notes

- Metrics are exported in the Prometheus OpenMetrics format
- All metrics are thread-safe due to Prometheus client library design
- By default, metrics are collected for all jobs automatically
- No configuration needed in the INI file - just use the command-line flags
- Disabling the metrics server has no performance impact on job execution
