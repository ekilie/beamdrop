// Package metrics provides Prometheus instrumentation for Beamdrop.
//
// It registers counters, histograms, and gauges with the default
// Prometheus registry and exposes helpers to update them from
// middleware, handlers, and background goroutines.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "beamdrop"

// ---------------------------------------------------------------------------
// Counters
// ---------------------------------------------------------------------------

// RequestsTotal counts every HTTP request, labelled by method, path, and
// status code so operators can build per-route dashboards.
var RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "requests_total",
	Help:      "Total number of HTTP requests.",
}, []string{"method", "path", "status"})

// AuthFailuresTotal counts authentication failures by reason
// (invalid_token, missing_token, invalid_password, etc.).
var AuthFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "auth_failures_total",
	Help:      "Total number of authentication failures.",
}, []string{"reason"})

// UploadsTotal counts completed file uploads.
var UploadsTotal = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "uploads_total",
	Help:      "Total number of completed file uploads.",
})

// DownloadsTotal counts completed file downloads.
var DownloadsTotal = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "downloads_total",
	Help:      "Total number of completed file downloads.",
})

// ---------------------------------------------------------------------------
// Histograms
// ---------------------------------------------------------------------------

// RequestDurationSeconds measures per-request latency. The default
// bucket boundaries are tuned for a file-server workload: most
// requests complete in under 100ms, but uploads/downloads can take seconds.
var RequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "request_duration_seconds",
	Help:      "HTTP request duration in seconds.",
	Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
}, []string{"method", "path", "status"})

// UploadSizeBytes tracks individual file upload sizes.
var UploadSizeBytes = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "upload_size_bytes",
	Help:      "Size of uploaded files in bytes.",
	Buckets:   prometheus.ExponentialBuckets(1024, 4, 10),
})

// ---------------------------------------------------------------------------
// Gauges
// ---------------------------------------------------------------------------

// StorageBytes reports the total number of bytes used in the shared
// directory. Updated periodically by a background collector.
var StorageBytes = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "storage_bytes",
	Help:      "Total bytes used by stored files.",
})

// ObjectsTotal reports the current number of objects (files) in the
// shared directory. Updated periodically by a background collector.
var ObjectsTotal = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "objects_total",
	Help:      "Total number of stored objects (files).",
})

// ActiveConnections tracks the number of in-flight HTTP requests.
var ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "active_connections",
	Help:      "Number of currently active HTTP connections.",
})

// StorageFreeBytes reports the free bytes on the filesystem where the
// shared directory resides.
var StorageFreeBytes = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "storage_free_bytes",
	Help:      "Free bytes on the shared-directory filesystem.",
})

// StorageTotalBytes reports the total capacity of the filesystem where the
// shared directory resides.
var StorageTotalBytes = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "storage_total_bytes",
	Help:      "Total capacity bytes of the shared-directory filesystem.",
})

// GoroutinesCount tracks the current number of goroutines.
var GoroutinesCount = promauto.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "goroutines_count",
	Help:      "Current number of goroutines.",
})
